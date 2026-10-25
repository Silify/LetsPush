package fcm

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func keyFile(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	raw, _ := json.Marshal(map[string]string{
		"type": "service_account", "project_id": "my-project",
		"client_email": "relay@my-project.iam.gserviceaccount.com", "private_key": string(pemKey),
	})
	return raw
}

// google is a fake Google: it answers sign-in and send, and counts both.
type google struct {
	server    *httptest.Server
	signIns   atomic.Int32
	sends     atomic.Int32
	signInHit func()
	sendReply func(w http.ResponseWriter)
	lastSend  atomic.Value // []byte
	lastAuth  atomic.Value // string
	lastForm  atomic.Value // map[string]string

	mu     sync.Mutex
	pauses []time.Duration // what the client slept between attempts
}

func newGoogle(t *testing.T) *google {
	t.Helper()
	g := &google{}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		g.signIns.Add(1)
		if g.signInHit != nil {
			g.signInHit()
		}
		r.ParseForm()
		g.lastForm.Store(map[string]string{"grant_type": r.Form.Get("grant_type"), "assertion": r.Form.Get("assertion")})
		json.NewEncoder(w).Encode(map[string]any{"access_token": "access-1", "expires_in": 3600})
	})
	mux.HandleFunc("/v1/projects/my-project/messages:send", func(w http.ResponseWriter, r *http.Request) {
		g.sends.Add(1)
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		g.lastSend.Store(body)
		g.lastAuth.Store(r.Header.Get("Authorization"))
		if g.sendReply != nil {
			g.sendReply(w)
			return
		}
		w.Write([]byte(`{"name":"projects/my-project/messages/1"}`))
	})
	g.server = httptest.NewServer(mux)
	t.Cleanup(g.server.Close)
	return g
}

func (g *google) client(now func() time.Time) *Client {
	c := NewClient()
	c.TokenURL = g.server.URL + "/token"
	c.SendBase = g.server.URL
	c.Sleep = func(_ context.Context, d time.Duration) error {
		g.mu.Lock()
		g.pauses = append(g.pauses, d)
		g.mu.Unlock()
		return nil
	}
	if now != nil {
		c.Now = now
	}
	return c
}

func creds(t *testing.T, key *rsa.PrivateKey) Credentials {
	t.Helper()
	c, err := ParseCredentials(keyFile(t, key))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

var message = Message{Token: "device", Title: "Site", Body: "Hello", Data: map[string]string{"a": "1"}}

func TestParseCredentials(t *testing.T) {
	key := newKey(t)
	if c, err := ParseCredentials(keyFile(t, key)); err != nil || c.ProjectID != "my-project" {
		t.Fatalf("a good key was refused: %v", err)
	}
	for name, raw := range map[string]string{
		"not json":   "nope",
		"wrong type": `{"type":"authorized_user"}`,
		"no project": `{"type":"service_account","client_email":"a","private_key":"b"}`,
		"not pem":    `{"type":"service_account","project_id":"p","client_email":"a","private_key":"b"}`,
		"not a key":  `{"type":"service_account","project_id":"p","client_email":"a","private_key":"-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----"}`,
	} {
		if _, err := ParseCredentials([]byte(raw)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestSignInIsASignedAssertion(t *testing.T) {
	key := newKey(t)
	g := newGoogle(t)
	fixed := time.Unix(1_700_000_000, 0)
	if err := g.client(func() time.Time { return fixed }).Send(context.Background(), creds(t, key), message); err != nil {
		t.Fatal(err)
	}

	form := g.lastForm.Load().(map[string]string)
	if form["grant_type"] != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
		t.Fatalf("wrong grant type %q", form["grant_type"])
	}
	parts := strings.Split(form["assertion"], ".")
	if len(parts) != 3 {
		t.Fatalf("not a JWT: %q", form["assertion"])
	}
	var claims map[string]any
	raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
	json.Unmarshal(raw, &claims)
	if claims["iss"] != "relay@my-project.iam.gserviceaccount.com" || claims["scope"] != scope ||
		claims["aud"] != defaultTokenURL || claims["iat"] != float64(1_700_000_000) || claims["exp"] != float64(1_700_003_600) {
		t.Fatalf("unexpected claims %v", claims)
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, sum[:], sig); err != nil {
		t.Fatalf("the signature does not verify: %v", err)
	}
}

func TestSendShapeAndAuthorization(t *testing.T) {
	g := newGoogle(t)
	if err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), message); err != nil {
		t.Fatal(err)
	}
	if g.lastAuth.Load() != "Bearer access-1" {
		t.Fatalf("wrong authorization %v", g.lastAuth.Load())
	}
	var sent struct {
		Message struct {
			Token        string            `json:"token"`
			Notification map[string]string `json:"notification"`
			Data         map[string]string `json:"data"`
			Android      map[string]string `json:"android"`
		} `json:"message"`
	}
	json.Unmarshal(g.lastSend.Load().([]byte), &sent)
	m := sent.Message
	if m.Token != "device" || m.Notification["title"] != "Site" || m.Notification["body"] != "Hello" ||
		m.Data["a"] != "1" || m.Android["priority"] != "high" {
		t.Fatalf("unexpected message %+v", m)
	}
}

func TestOneSignInServesManySendsEvenWhenTheyStartTogether(t *testing.T) {
	g := newGoogle(t)
	release := make(chan struct{})
	g.signInHit = func() { <-release }
	c := g.client(nil)
	cr := creds(t, newKey(t))

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.Send(context.Background(), cr, message); err != nil {
				t.Error(err)
			}
		}()
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()
	c.Send(context.Background(), cr, message)

	if g.signIns.Load() != 1 || g.sends.Load() != 6 {
		t.Fatalf("expected 1 sign-in and 6 sends, got %d and %d", g.signIns.Load(), g.sends.Load())
	}
}

func TestAnExpiringTokenIsRenewed(t *testing.T) {
	g := newGoogle(t)
	now := time.Unix(1_700_000_000, 0)
	c := g.client(func() time.Time { return now })
	cr := creds(t, newKey(t))

	c.Send(context.Background(), cr, message)
	now = now.Add(50 * time.Minute)
	c.Send(context.Background(), cr, message)
	if g.signIns.Load() != 1 {
		t.Fatalf("renewed too early: %d", g.signIns.Load())
	}
	now = now.Add(9*time.Minute + 30*time.Second)
	c.Send(context.Background(), cr, message)
	if g.signIns.Load() != 2 {
		t.Fatalf("not renewed near expiry: %d", g.signIns.Load())
	}
}

func TestARefusedTokenIsReplacedAndTheSendRetried(t *testing.T) {
	g := newGoogle(t)
	c := g.client(nil)
	cr := creds(t, newKey(t))
	g.sendReply = func(w http.ResponseWriter) {
		g.sendReply = nil // only the first answer is a refusal
		w.WriteHeader(401)
	}

	if err := c.Send(context.Background(), cr, message); err != nil {
		t.Fatalf("the retry with a fresh token should have worked: %v", err)
	}
	if g.signIns.Load() != 2 || g.sends.Load() != 2 {
		t.Fatalf("expected 2 sign-ins and 2 sends, got %d and %d", g.signIns.Load(), g.sends.Load())
	}
}

func TestATokenThatIsAlwaysRefusedGivesUp(t *testing.T) {
	g := newGoogle(t)
	g.sendReply = func(w http.ResponseWriter) { w.WriteHeader(401) }

	var apiErr *Error
	err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), message)
	if !errors.As(err, &apiErr) || apiErr.Status != 401 || g.sends.Load() != 3 {
		t.Fatalf("expected a 401 after 3 sends, got %v after %d", err, g.sends.Load())
	}
}

func TestADeadDeviceIsReported(t *testing.T) {
	for name, body := range map[string]string{
		"details": `{"error":{"status":"NOT_FOUND","details":[{"errorCode":"UNREGISTERED"}]}}`,
		"status":  `{"error":{"status":"NOT_FOUND"}}`,
	} {
		g := newGoogle(t)
		g.sendReply = func(w http.ResponseWriter) { w.WriteHeader(404); w.Write([]byte(body)) }
		if err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), message); !errors.Is(err, ErrUnregistered) {
			t.Errorf("%s: expected ErrUnregistered, got %v", name, err)
		}
	}
}

func TestAFailureCarriesGooglesReason(t *testing.T) {
	cases := map[string]struct{ body, want string }{
		"status and message": {
			`{"error":{"code":403,"status":"PERMISSION_DENIED","message":"Cloud Messaging API has not\nbeen used in project 1 before."}}`,
			"PERMISSION_DENIED: Cloud Messaging API has not been used in project 1 before.",
		},
		"with an error code": {
			`{"error":{"status":"PERMISSION_DENIED","message":"Sender id mismatch","details":[{"errorCode":"SENDER_ID_MISMATCH"}]}}`,
			"PERMISSION_DENIED/SENDER_ID_MISMATCH: Sender id mismatch",
		},
		"not JSON": {`<html>forbidden</html>`, ""},
		"too long": {`{"error":{"status":"X","message":"` + strings.Repeat("a", 500) + `"}}`, "X: " + strings.Repeat("a", reasonLimit-3) + "…"},
	}
	for name, c := range cases {
		g := newGoogle(t)
		g.sendReply = func(w http.ResponseWriter) { w.WriteHeader(403); w.Write([]byte(c.body)) }
		err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), message)

		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Status != 403 || apiErr.Reason != c.want {
			t.Errorf("%s: got %v, want reason %q", name, err, c.want)
		}
		if g.sends.Load() != 1 {
			t.Errorf("%s: a refusal that cannot change was retried", name)
		}
	}

	g := newGoogle(t)
	g.sendReply = func(w http.ResponseWriter) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":{"status":"PERMISSION_DENIED","message":"no"}}`))
	}
	err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), message)
	if err == nil || err.Error() != "firebase send failed with status 403: PERMISSION_DENIED: no" {
		t.Fatalf("unexpected text %v", err)
	}
}

func TestABusyGoogleIsRetriedWithGrowingPauses(t *testing.T) {
	g := newGoogle(t)
	replies := []int{429, 503, 200}
	g.sendReply = func(w http.ResponseWriter) {
		code := replies[0]
		replies = replies[1:]
		w.WriteHeader(code)
	}

	if err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), message); err != nil {
		t.Fatal(err)
	}
	if g.sends.Load() != 3 || len(g.pauses) != 2 {
		t.Fatalf("expected 3 sends and 2 pauses, got %d and %d", g.sends.Load(), len(g.pauses))
	}
	// The pause doubles from 300 ms, with up to half again as jitter.
	if g.pauses[0] < 150*time.Millisecond || g.pauses[0] >= 450*time.Millisecond ||
		g.pauses[1] < 300*time.Millisecond || g.pauses[1] >= 900*time.Millisecond {
		t.Fatalf("unexpected pauses %v", g.pauses)
	}
}

func TestRetryAfterIsHonouredAndRetriesStop(t *testing.T) {
	g := newGoogle(t)
	g.sendReply = func(w http.ResponseWriter) {
		w.Header().Set("Retry-After", "4")
		w.WriteHeader(429)
	}
	err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), message)

	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != 429 || g.sends.Load() != 3 {
		t.Fatalf("expected a 429 after 3 sends, got %v after %d", err, g.sends.Load())
	}
	if g.pauses[0] != 4*time.Second {
		t.Fatalf("Retry-After was not honoured: %v", g.pauses)
	}
}

func TestRetriesStopWhenTheCallerGivesUp(t *testing.T) {
	g := newGoogle(t)
	g.sendReply = func(w http.ResponseWriter) { w.WriteHeader(503) }
	c := g.client(nil)
	c.Sleep = func(context.Context, time.Duration) error { return context.Canceled }

	if err := c.Send(context.Background(), creds(t, newKey(t)), message); err == nil || g.sends.Load() != 1 {
		t.Fatalf("expected the 503 after one send, got %v after %d", err, g.sends.Load())
	}
}

func TestADroppedConnectionIsRetried(t *testing.T) {
	g := newGoogle(t)
	var calls atomic.Int32
	g.sendReply = func(w http.ResponseWriter) {
		if calls.Add(1) == 1 {
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}
		w.Write([]byte(`{}`))
	}
	if err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), message); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 sends, got %d", calls.Load())
	}
}

func TestAFailedSignInIsAnErrorWithGooglesReason(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"Invalid JWT Signature."}`))
	}))
	defer bad.Close()
	c := NewClient()
	c.TokenURL = bad.URL

	var apiErr *Error
	err := c.Send(context.Background(), creds(t, newKey(t)), message)
	if !errors.As(err, &apiErr) || apiErr.Stage != "sign-in" || apiErr.Reason != "invalid_grant: Invalid JWT Signature." {
		t.Fatalf("expected a sign-in error with a reason, got %v", err)
	}
}

func TestAMessageWithoutTextIsDataOnly(t *testing.T) {
	g := newGoogle(t)
	err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), Message{Token: "device", Data: map[string]string{"a": "1"}})
	if err != nil {
		t.Fatal(err)
	}
	var sent map[string]map[string]any
	json.Unmarshal(g.lastSend.Load().([]byte), &sent)
	if _, has := sent["message"]["notification"]; has {
		t.Fatal("a message with no text must not carry a notification block")
	}
	if sent["message"]["data"] == nil {
		t.Fatal("the data was dropped")
	}
}

func sentMessage(g *google) map[string]any {
	var sent map[string]map[string]any
	json.Unmarshal(g.lastSend.Load().([]byte), &sent)
	return sent["message"]
}

func TestTheDefaultsAreHighPriorityAndNoExtras(t *testing.T) {
	g := newGoogle(t)
	g.client(nil).Send(context.Background(), creds(t, newKey(t)), message)
	m := sentMessage(g)

	if m["android"].(map[string]any)["priority"] != "high" {
		t.Fatalf("unexpected android part %v", m["android"])
	}
	if m["apns"].(map[string]any)["headers"].(map[string]any)["apns-priority"] != "10" {
		t.Fatalf("unexpected apns part %v", m["apns"])
	}
	for _, absent := range []string{"webpush"} {
		if _, has := m[absent]; has {
			t.Errorf("%s was sent although nothing asked for it", absent)
		}
	}
	if _, has := m["notification"].(map[string]any)["image"]; has {
		t.Error("an image was sent although there was none")
	}
}

func TestDeliveryOptionsReachEveryPlatform(t *testing.T) {
	g := newGoogle(t)
	now := time.Unix(1_700_000_000, 0)
	msg := Message{Token: "device", Title: "T", Body: "B", Image: "https://example.com/a.png",
		Priority: "normal", TTL: time.Hour, Sound: "ding"}
	if err := g.client(func() time.Time { return now }).Send(context.Background(), creds(t, newKey(t)), msg); err != nil {
		t.Fatal(err)
	}
	m := sentMessage(g)

	if n := m["notification"].(map[string]any); n["image"] != "https://example.com/a.png" || n["title"] != "T" {
		t.Errorf("unexpected notification %v", n)
	}
	android := m["android"].(map[string]any)
	if android["priority"] != "normal" || android["ttl"] != "3600s" ||
		android["notification"].(map[string]any)["sound"] != "ding" {
		t.Errorf("unexpected android part %v", android)
	}
	apns := m["apns"].(map[string]any)
	headers := apns["headers"].(map[string]any)
	if headers["apns-priority"] != "5" || headers["apns-expiration"] != "1700003600" ||
		apns["payload"].(map[string]any)["aps"].(map[string]any)["sound"] != "ding" {
		t.Errorf("unexpected apns part %v", apns)
	}
	web := m["webpush"].(map[string]any)["headers"].(map[string]any)
	if web["Urgency"] != "normal" || web["TTL"] != "3600" {
		t.Errorf("unexpected webpush part %v", web)
	}
}
