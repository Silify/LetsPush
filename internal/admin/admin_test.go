package admin

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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
		"type": "service_account", "project_id": "my-project", "client_email": "a@p.iam.gserviceaccount.com",
		"private_key": string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
	})
	serviceAccount = string(raw)
	os.Exit(m.Run())
}

const (
	adminEmail    = "admin@example.com"
	adminPassword = "a-long-password"
	serverSecret  = "0123456789abcdef0123456789abcdef"
)

type env struct {
	t       *testing.T
	store   *store.Store
	handler http.Handler
	now     time.Time
	cookie  *http.Cookie
}

func newEnv(t *testing.T) *env {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	hash, _ := secret.HashPassword(adminPassword)
	if _, err := s.CreateAdmin(adminEmail, hash); err != nil {
		t.Fatal(err)
	}
	box, _ := secret.NewBox(serverSecret)
	e := &env{t: t, store: s, now: time.Unix(1_700_000_000, 0)}
	handler, err := New(s, box, serverSecret, func() time.Time { return e.now })
	if err != nil {
		t.Fatal(err)
	}
	e.handler = handler
	return e
}

func (e *env) do(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "203.0.113.5:4000"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if e.cookie != nil {
		req.AddCookie(e.cookie)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

var csrf = map[string]string{"X-Requested-With": "letspush"}

func (e *env) login() {
	e.t.Helper()
	rec := e.do("POST", "/admin/api/login", `{"email":"`+adminEmail+`","password":"`+adminPassword+`"}`, nil)
	if rec.Code != 200 {
		e.t.Fatalf("login failed: %d %s", rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName {
			e.cookie = c
		}
	}
	if e.cookie == nil {
		e.t.Fatal("no session cookie")
	}
}

func (e *env) call(method, path, body string) *httptest.ResponseRecorder {
	e.t.Helper()
	return e.do(method, path, body, csrf)
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not a JSON object: %q", rec.Body.String())
	}
	return out
}

func status(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, want %d (%s)", rec.Code, want, rec.Body.String())
	}
}

func TestSigningIn(t *testing.T) {
	e := newEnv(t)

	status(t, e.do("GET", "/admin/api/me", "", nil), 401)

	bad := e.do("POST", "/admin/api/login", `{"email":"`+adminEmail+`","password":"wrong"}`, nil)
	status(t, bad, 401)
	unknown := e.do("POST", "/admin/api/login", `{"email":"nobody@example.com","password":"`+adminPassword+`"}`, nil)
	status(t, unknown, 401)
	if bad.Body.String() != unknown.Body.String() {
		t.Fatal("the two failures must look the same")
	}

	e.login()
	me := e.do("GET", "/admin/api/me", "", nil)
	status(t, me, 200)
	if decode(t, me)["email"] != adminEmail {
		t.Fatal("wrong admin")
	}

	cookie := e.cookie
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Secure {
		t.Fatalf("unexpected cookie flags %+v", cookie)
	}
}

func TestTheCookieIsSecureBehindHTTPSAndRefusesTampering(t *testing.T) {
	e := newEnv(t)
	rec := e.do("POST", "/admin/api/login", `{"email":"`+adminEmail+`","password":"`+adminPassword+`"}`, map[string]string{"X-Forwarded-Proto": "https"})
	if c := rec.Result().Cookies()[0]; !c.Secure {
		t.Fatal("the cookie should be Secure behind HTTPS")
	}

	e.login()
	good := e.cookie.Value
	for name, value := range map[string]string{
		"altered": good[:len(good)-2] + "00",
		"no dot":  strings.ReplaceAll(good, ".", ""),
		"garbage": "xyz",
		"empty":   "",
		"swapped": "MTox." + good[strings.Index(good, ".")+1:],
	} {
		e.cookie = &http.Cookie{Name: cookieName, Value: value}
		if rec := e.do("GET", "/admin/api/me", "", nil); rec.Code != 401 {
			t.Errorf("%s cookie was accepted", name)
		}
	}
}

func TestASessionExpires(t *testing.T) {
	e := newEnv(t)
	e.login()
	e.now = e.now.Add(sessionTime - time.Minute)
	status(t, e.do("GET", "/admin/api/me", "", nil), 200)
	e.now = e.now.Add(2 * time.Minute)
	status(t, e.do("GET", "/admin/api/me", "", nil), 401)
}

func TestTooManyWrongPasswordsLockTheAttemptOut(t *testing.T) {
	e := newEnv(t)
	wrong := `{"email":"` + adminEmail + `","password":"wrong"}`
	for i := 0; i < maxFailures; i++ {
		status(t, e.do("POST", "/admin/api/login", wrong, nil), 401)
	}
	status(t, e.do("POST", "/admin/api/login", wrong, nil), 429)
	// Even the right password waits, or the limit would not stop guessing.
	status(t, e.do("POST", "/admin/api/login", `{"email":"`+adminEmail+`","password":"`+adminPassword+`"}`, nil), 429)

	e.now = e.now.Add(failureWindow + time.Minute)
	status(t, e.do("POST", "/admin/api/login", `{"email":"`+adminEmail+`","password":"`+adminPassword+`"}`, nil), 200)
}

func TestChangesNeedTheRequestHeader(t *testing.T) {
	e := newEnv(t)
	e.login()
	status(t, e.do("POST", "/admin/api/apps", `{"name":"sayan"}`, nil), 403)
	status(t, e.call("POST", "/admin/api/apps", `{"name":"sayan"}`), 201)
	status(t, e.do("GET", "/admin/api/apps", "", nil), 200)
}

func TestSigningOut(t *testing.T) {
	e := newEnv(t)
	e.login()
	rec := e.do("POST", "/admin/api/logout", "", nil)
	status(t, rec, 200)
	if c := rec.Result().Cookies()[0]; c.MaxAge >= 0 {
		t.Fatal("the cookie was not cleared")
	}
}

func TestEverythingElseNeedsASession(t *testing.T) {
	e := newEnv(t)
	for _, c := range [][2]string{
		{"GET", "/admin/api/apps"}, {"POST", "/admin/api/apps"}, {"GET", "/admin/api/apps/1"},
		{"DELETE", "/admin/api/apps/1"}, {"PUT", "/admin/api/apps/1/credentials"}, {"PUT", "/admin/api/apps/1/settings"},
		{"GET", "/admin/api/apps/1/logs"}, {"POST", "/admin/api/apps/1/keys"}, {"DELETE", "/admin/api/apps/1/keys/1"},
		{"POST", "/admin/api/password"},
	} {
		if rec := e.do(c[0], c[1], "{}", csrf); rec.Code != 401 {
			t.Errorf("%s %s answered %d without a session", c[0], c[1], rec.Code)
		}
	}
	status(t, e.do("GET", "/admin/api/nothing", "", nil), 404)
}

func TestAppsAndTheirFirebaseKeys(t *testing.T) {
	e := newEnv(t)
	e.login()

	key, _ := json.Marshal(serviceAccount)
	rec := e.call("POST", "/admin/api/apps", `{"name":"sayan","serviceAccount":`+string(key)+`}`)
	status(t, rec, 201)
	app := decode(t, rec)
	if app["projectId"] != "my-project" || app["hasCredentials"] != true {
		t.Fatalf("unexpected app %v", app)
	}
	if p, _ := e.store.AppProvider(int64(app["id"].(float64)), store.ProviderFCM); p.Credentials == "" || strings.Contains(p.Credentials, "private_key") {
		t.Fatal("the Firebase key was stored unsealed")
	}
	if strings.Contains(rec.Body.String(), "BEGIN PRIVATE KEY") {
		t.Fatal("the response gave the key back")
	}

	status(t, e.call("POST", "/admin/api/apps", `{"name":"sayan"}`), 409)
	status(t, e.call("POST", "/admin/api/apps", `{"name":"bad name!"}`), 400)
	status(t, e.call("POST", "/admin/api/apps", `{"name":"other","serviceAccount":"{\"type\":\"nope\"}"}`), 400)

	bare := decode(t, e.call("POST", "/admin/api/apps", `{"name":"bare"}`))
	if bare["hasCredentials"] != false {
		t.Fatal("an app made without a key should say so")
	}
	id := strconv.Itoa(int(bare["id"].(float64)))
	status(t, e.call("PUT", "/admin/api/apps/"+id+"/credentials", `{"serviceAccount":"bad"}`), 400)
	rec = e.call("PUT", "/admin/api/apps/"+id+"/credentials", `{"serviceAccount":`+string(key)+`}`)
	status(t, rec, 200)
	if decode(t, rec)["hasCredentials"] != true {
		t.Fatal("the key was not saved")
	}

	rec = e.call("PUT", "/admin/api/apps/"+id+"/settings", `{"allowKeyRequests":true}`)
	if decode(t, rec)["allowKeyRequests"] != true {
		t.Fatal("the switch was not saved")
	}

	list := e.call("GET", "/admin/api/apps", "")
	var apps []map[string]any
	json.Unmarshal(list.Body.Bytes(), &apps)
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}

	status(t, e.call("DELETE", "/admin/api/apps/"+id, ""), 200)
	status(t, e.call("GET", "/admin/api/apps/"+id, ""), 404)
	status(t, e.call("GET", "/admin/api/apps/abc", ""), 404)
}

func TestAccessKeys(t *testing.T) {
	e := newEnv(t)
	e.login()
	app, _ := e.store.CreateApp("sayan")
	base := "/admin/api/apps/" + strconv.FormatInt(app.ID, 10)

	rec := e.call("POST", base+"/keys", `{"description":"School A","contact":"admin@a.example"}`)
	status(t, rec, 201)
	created := decode(t, rec)
	raw, _ := created["key"].(string)
	if len(raw) != 32 || len(created["permissions"].([]any)) != 3 {
		t.Fatalf("unexpected key %v", created)
	}

	// The key is stored only as a hash, so it cannot be read back.
	shown := e.call("GET", base, "")
	if strings.Contains(shown.Body.String(), raw) {
		t.Fatal("the key can be read again")
	}
	if _, err := e.store.KeyByHash(secret.Hash(raw)); err != nil {
		t.Fatalf("the key does not work: %v", err)
	}

	limited := decode(t, e.call("POST", base+"/keys", `{"permissions":["sendNotification"]}`))
	if got := limited["permissions"].([]any); len(got) != 1 || got[0] != "sendNotification" {
		t.Fatalf("unexpected permissions %v", got)
	}
	status(t, e.call("POST", base+"/keys", `{"permissions":["fly"]}`), 400)
	status(t, e.call("POST", base+"/keys", `{"permissions":[]}`), 400)
	status(t, e.call("POST", "/admin/api/apps/999/keys", `{}`), 404)

	keyID := strconv.Itoa(int(created["id"].(float64)))
	status(t, e.call("DELETE", base+"/keys/"+keyID, ""), 200)
	status(t, e.call("DELETE", base+"/keys/"+keyID, ""), 404)
	if _, err := e.store.KeyByHash(secret.Hash(raw)); err == nil {
		t.Fatal("a revoked key still works")
	}
}

func TestLogs(t *testing.T) {
	e := newEnv(t)
	e.login()
	app, _ := e.store.CreateApp("sayan")
	base := "/admin/api/apps/" + strconv.FormatInt(app.ID, 10) + "/logs"
	for i := 0; i < 3; i++ {
		e.store.AddLog(app.ID, "push", "line "+strconv.Itoa(i))
	}

	var entries []map[string]any
	json.Unmarshal(e.call("GET", base, "").Body.Bytes(), &entries)
	if len(entries) != 3 || entries[0]["info"] != "line 2" {
		t.Fatalf("unexpected log %v", entries)
	}
	json.Unmarshal(e.call("GET", base+"?limit=1", "").Body.Bytes(), &entries)
	if len(entries) != 1 {
		t.Fatal("the limit was ignored")
	}
	status(t, e.call("GET", base+"?limit=0", ""), 400)
	status(t, e.call("GET", base+"?limit=9999", ""), 400)
}

func TestChangingThePassword(t *testing.T) {
	e := newEnv(t)
	e.login()
	status(t, e.call("POST", "/admin/api/password", `{"current":"wrong","new":"another-long-password"}`), 403)
	status(t, e.call("POST", "/admin/api/password", `{"current":"`+adminPassword+`","new":"short"}`), 400)
	status(t, e.call("POST", "/admin/api/password", `{"current":"`+adminPassword+`","new":"another-long-password"}`), 200)

	e.cookie = nil
	status(t, e.do("POST", "/admin/api/login", `{"email":"`+adminEmail+`","password":"`+adminPassword+`"}`, nil), 401)
	status(t, e.do("POST", "/admin/api/login", `{"email":"`+adminEmail+`","password":"another-long-password"}`, nil), 200)
}
