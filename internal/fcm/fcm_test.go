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

func TestARefusedTokenIsForgotten(t *testing.T) {
	g := newGoogle(t)
	c := g.client(nil)
	cr := creds(t, newKey(t))
	g.sendReply = func(w http.ResponseWriter) { w.WriteHeader(401) }

	var apiErr *Error
	if err := c.Send(context.Background(), cr, message); !errors.As(err, &apiErr) || apiErr.Status != 401 {
		t.Fatalf("expected a 401 error, got %v", err)
	}
	g.sendReply = nil
	c.Send(context.Background(), cr, message)
	if g.signIns.Load() != 2 {
		t.Fatalf("the refused token should be replaced: %d sign-ins", g.signIns.Load())
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

func TestOtherFailuresCarryOnlyTheStatus(t *testing.T) {
	g := newGoogle(t)
	g.sendReply = func(w http.ResponseWriter) {
		w.WriteHeader(503)
		w.Write([]byte(`{"error":{"message":"secret detail"}}`))
	}
	err := g.client(nil).Send(context.Background(), creds(t, newKey(t)), message)

	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != 503 || errors.Is(err, ErrUnregistered) {
		t.Fatalf("unexpected error %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("Google's text leaked into the error")
	}
}

func TestAFailedSignInIsAnError(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400) }))
	defer bad.Close()
	c := NewClient()
	c.TokenURL = bad.URL

	var apiErr *Error
	err := c.Send(context.Background(), creds(t, newKey(t)), message)
	if !errors.As(err, &apiErr) || apiErr.Stage != "sign-in" {
		t.Fatalf("expected a sign-in error, got %v", err)
	}
}
