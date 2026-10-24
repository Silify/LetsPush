package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Silify/letspush/internal/fcm"
	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
)

var serviceAccount string

func TestMain(m *testing.M) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	raw, _ := json.Marshal(map[string]string{
		"type": "service_account", "project_id": "p", "client_email": "a@p.iam.gserviceaccount.com",
		"private_key": string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
	})
	serviceAccount = string(raw)
	os.Exit(m.Run())
}

type recordingSender struct{ messages []fcm.Message }

func (r *recordingSender) Send(_ context.Context, _ fcm.Credentials, m fcm.Message) error {
	r.messages = append(r.messages, m)
	return nil
}

const serverSecret = "0123456789abcdef0123456789abcdef"

func build(t *testing.T, ui http.Handler) (http.Handler, *store.Store, *recordingSender) {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	box, _ := secret.NewBox(serverSecret)
	sender := &recordingSender{}
	h, err := New(Options{Store: db, Box: box, Sender: sender, Secret: serverSecret, UI: ui})
	if err != nil {
		t.Fatal(err)
	}
	return h, db, sender
}

func send(h http.Handler, method, path, body string, headers map[string]string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestTheWholeJourneyFromSetupToAPush(t *testing.T) {
	h, db, sender := build(t, nil)

	if created, err := EnsureAdmin(db, "admin@example.com", "a-long-password"); err != nil || !created {
		t.Fatalf("the first admin was not created: %v", err)
	}

	// The admin signs in and sets up an app with a key.
	login := send(h, "POST", "/admin/api/login", `{"email":"admin@example.com","password":"a-long-password"}`, nil, nil)
	if login.Code != 200 {
		t.Fatalf("login: %d %s", login.Code, login.Body.String())
	}
	session := login.Result().Cookies()[0]
	admin := map[string]string{"X-Requested-With": "letspush"}

	account, _ := json.Marshal(serviceAccount)
	app := send(h, "POST", "/admin/api/apps", `{"name":"sayan","serviceAccount":`+string(account)+`}`, admin, session)
	if app.Code != 201 {
		t.Fatalf("create app: %d %s", app.Code, app.Body.String())
	}
	var created map[string]any
	json.Unmarshal(app.Body.Bytes(), &created)
	appID := int(created["id"].(float64))

	keyResponse := send(h, "POST", "/admin/api/apps/"+itoa(appID)+"/keys", `{"description":"School"}`, admin, session)
	var key map[string]any
	json.Unmarshal(keyResponse.Body.Bytes(), &key)
	accessKey := key["key"].(string)

	// The site registers a device and sends a push, as Moodle does.
	headers := map[string]string{"X-AN-APP-NAME": "sayan", "X-AN-APP-KEY": accessKey}
	register := send(h, "POST", "/tokens/DEVICE-1", url.Values{"device": {"android"}}.Encode(),
		mergeHeaders(headers, map[string]string{"Content-Type": "application/x-www-form-urlencoded"}), nil)
	if register.Code != 200 || !strings.Contains(register.Body.String(), `"ok"`) {
		t.Fatalf("register: %d %s", register.Code, register.Body.String())
	}
	push := send(h, "POST", "/api/v2/push/",
		`{"device":"android","token":"DEVICE-1","extra":{"processor":"moodle","sitefullname":"School","smallmessage":"Hi"}}`,
		mergeHeaders(headers, map[string]string{"Content-Type": "application/json"}), nil)
	if push.Code != 202 {
		t.Fatalf("push: %d %s", push.Code, push.Body.String())
	}
	if len(sender.messages) != 1 || sender.messages[0].Title != "School" || sender.messages[0].Body != "Hi" {
		t.Fatalf("unexpected messages %+v", sender.messages)
	}

	// The admin sees it in the log and in the counts.
	logs := send(h, "GET", "/admin/api/apps/"+itoa(appID)+"/logs", "", nil, session)
	if !strings.Contains(logs.Body.String(), "sent to DEVI…") {
		t.Fatalf("the push is not in the log: %s", logs.Body.String())
	}
	if strings.Contains(logs.Body.String(), "DEVICE-1") {
		t.Fatal("the whole device token is in the log")
	}
}

func TestEnsureAdminOnlyActsOnce(t *testing.T) {
	_, db, _ := build(t, nil)
	if created, _ := EnsureAdmin(db, "", ""); created {
		t.Fatal("nothing should be created without settings")
	}
	EnsureAdmin(db, "first@example.com", "a-long-password")
	if created, _ := EnsureAdmin(db, "second@example.com", "another-password"); created {
		t.Fatal("a second admin must not appear from the settings")
	}
	if n, _ := db.CountAdmins(); n != 1 {
		t.Fatalf("expected 1 admin, got %d", n)
	}
}

func TestMoodlesConnectivityCheckGetsThePage(t *testing.T) {
	ui := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("page")) })
	h, _, _ := build(t, ui)
	if rec := send(h, "GET", "/", "", nil, nil); rec.Code != 200 || rec.Body.String() != "page" {
		t.Fatalf("GET / answered %d %q", rec.Code, rec.Body.String())
	}
	if rec := send(h, "GET", "/apps/3", "", nil, nil); rec.Body.String() != "page" {
		t.Fatal("interface routes should reach the page")
	}
	if rec := send(h, "GET", "/admin/api/unknown", "", nil, nil); rec.Header().Get("Content-Type") == "text/html" || rec.Body.String() == "page" {
		t.Fatal("the API prefix must never fall through to the page")
	}
}

func TestEveryAnswerCarriesSecurityHeaders(t *testing.T) {
	h, _, _ := build(t, nil)
	for _, path := range []string{"/", "/admin/api/me", "/tokens/x"} {
		rec := send(h, "GET", path, "", nil, nil)
		for _, header := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy", "Referrer-Policy"} {
			if rec.Header().Get(header) == "" {
				t.Errorf("%s: missing %s", path, header)
			}
		}
	}
}

func TestAPanicBecomesAJSONError(t *testing.T) {
	var out bytes.Buffer
	log.SetOutput(&out)
	defer log.SetOutput(os.Stderr)

	h, _, _ := build(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("boom") }))
	rec := send(h, "GET", "/", "", nil, nil)
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), "something went wrong") {
		t.Fatalf("unexpected answer: %d %q", rec.Code, rec.Body.String())
	}
}

func TestTheAccessLogHidesDeviceTokens(t *testing.T) {
	var out bytes.Buffer
	log.SetOutput(&out)
	defer log.SetOutput(os.Stderr)

	h, _, _ := build(t, nil)
	send(h, "DELETE", "/tokens/SECRET-DEVICE-TOKEN", "", map[string]string{"X-AN-APP-NAME": "x", "X-AN-APP-KEY": "y"}, nil)
	send(h, "POST", "/api/v2/tokens/ANOTHER-SECRET", "", nil, nil)
	if strings.Contains(out.String(), "SECRET") {
		t.Fatalf("a device token reached the log: %s", out.String())
	}
	if !strings.Contains(out.String(), "/tokens/…") {
		t.Fatalf("the request was not logged: %s", out.String())
	}
}

func TestPurgeLogsDeletesOldEntriesAndStops(t *testing.T) {
	_, db, _ := build(t, nil)
	app, _ := db.CreateApp("sayan", "", "")
	db.AddLog(app.ID, "push", "kept")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { PurgeLogs(ctx, db, 30, time.Hour); close(done) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("PurgeLogs did not stop")
	}
	if logs, _ := db.RecentLogs(app.ID, 10); len(logs) != 1 {
		t.Fatal("a recent entry was purged")
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func mergeHeaders(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
