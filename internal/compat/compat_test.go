package compat

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

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

type sentMessage struct {
	creds fcm.Credentials
	msg   fcm.Message
}

type fakeSender struct {
	sent []sentMessage
	err  error
}

func (f *fakeSender) Send(_ context.Context, creds fcm.Credentials, msg fcm.Message) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, sentMessage{creds, msg})
	return nil
}

type env struct {
	t       *testing.T
	store   *store.Store
	sender  *fakeSender
	handler http.Handler
	app     store.App
	key     string // the raw access key
	keyRow  store.AccessKey
}

func newEnv(t *testing.T, permissions int) *env {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	box, _ := secret.NewBox(strings.Repeat("k", 32))
	sealed, _ := box.Seal([]byte(serviceAccount))
	app, err := s.CreateApp("sayan", "p", sealed)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, store: s, sender: &fakeSender{}, app: app}
	e.handler = New(s, e.sender, box)
	e.key, e.keyRow = e.makeKey(app.ID, permissions)
	return e
}

func (e *env) makeKey(appID int64, permissions int) (string, store.AccessKey) {
	key, prefix, hash, _ := secret.NewAccessKey()
	row, err := e.store.CreateKey(store.NewKey{AppID: appID, Hash: hash, Prefix: prefix, Permissions: permissions})
	if err != nil {
		e.t.Fatal(err)
	}
	return key, row
}

type request struct {
	method, path, body string
	app, key           string
	form               bool
}

func (e *env) do(r request) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(r.method, r.path, strings.NewReader(r.body))
	if r.form {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.app != "" {
		req.Header.Set("X-AN-APP-NAME", r.app)
	}
	if r.key != "" {
		req.Header.Set("X-AN-APP-KEY", r.key)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func (e *env) authed(method, path, body string) request {
	return request{method: method, path: path, body: body, app: "sayan", key: e.key}
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON: %q", rec.Body.String())
	}
	return out
}

func expect(t *testing.T, rec *httptest.ResponseRecorder, status int, field, want string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d (%s)", rec.Code, status, rec.Body.String())
	}
	if field != "" {
		if got := decode(t, rec)[field]; got != want {
			t.Fatalf("%s = %v, want %q", field, got, want)
		}
	}
}

func TestRequestsAreChecked(t *testing.T) {
	e := newEnv(t, store.PermAll)
	other, _ := e.store.CreateApp("other", "", "")
	foreignKey, _ := e.makeKey(other.ID, store.PermAll)
	revoked, row := e.makeKey(e.app.ID, store.PermAll)
	e.store.RevokeKey(e.app.ID, row.ID)

	cases := []struct {
		name string
		req  request
		want string
	}{
		{"no app name", request{method: "POST", path: "/tokens/T", key: e.key, form: true}, "app name is required"},
		{"no key", request{method: "POST", path: "/tokens/T", app: "sayan", form: true}, "app key is required"},
		{"unknown app", request{method: "POST", path: "/tokens/T", app: "nope", key: e.key, form: true}, "Invalid application name"},
		{"wrong key", request{method: "POST", path: "/tokens/T", app: "sayan", key: "wrong", form: true}, "Invalid access key"},
		{"another app's key", request{method: "POST", path: "/tokens/T", app: "sayan", key: foreignKey, form: true}, "Invalid access key"},
		{"revoked key", request{method: "POST", path: "/tokens/T", app: "sayan", key: revoked, form: true}, "Invalid access key"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { expect(t, e.do(c.req), 400, "error", c.want) })
	}
}

func TestMoodlesKeyCheckReceivesTheAnswersItLooksFor(t *testing.T) {
	e := newEnv(t, store.PermAll)
	bad := e.do(request{method: "POST", path: "/tokens/testtoken", app: "sayan", key: "wrong"})
	expect(t, bad, 400, "error", "Invalid access key")

	good := e.do(e.authed("POST", "/tokens/testtoken", ""))
	expect(t, good, 200, "status", "ok")
	if _, err := e.store.FindToken(e.keyRow.ID, "testtoken"); err == nil {
		t.Fatal("the probe token must not be stored")
	}
}

func TestCreatingAndDeletingTokens(t *testing.T) {
	e := newEnv(t, store.PermAll)
	create := e.authed("POST", "/tokens/DEVICE-1", url.Values{"device": {"Android"}}.Encode())
	create.form = true

	expect(t, e.do(create), 200, "status", "ok")
	expect(t, e.do(create), 200, "status", "token exists")
	got, err := e.store.FindToken(e.keyRow.ID, "DEVICE-1")
	if err != nil || got.Device != "android" || got.Channel != "default" {
		t.Fatalf("stored wrongly: %v %+v", err, got)
	}

	expect(t, e.do(e.authed("DELETE", "/tokens/DEVICE-1", "")), 200, "status", "deleted")
	expect(t, e.do(e.authed("DELETE", "/tokens/DEVICE-1", "")), 404, "status", "Token doesn't exist")
}

func TestTokenPermissionsAndLimits(t *testing.T) {
	e := newEnv(t, store.PermSendNotification)
	expect(t, e.do(e.authed("POST", "/tokens/T", "")), 403, "error", "No permission to create token")
	expect(t, e.do(e.authed("DELETE", "/tokens/T", "")), 403, "error", "No permission to delete token")

	e = newEnv(t, store.PermAll)
	expect(t, e.do(e.authed("POST", "/tokens/"+strings.Repeat("x", maxTokenLength+1), "")), 400, "error", "Invalid token")
}

func TestTokensThroughTheJSONInterface(t *testing.T) {
	e := newEnv(t, store.PermAll)
	expect(t, e.do(e.authed("POST", "/api/v2/tokens", `{"token":"T2","device":"ios"}`)), 200, "status", "ok")
	expect(t, e.do(e.authed("POST", "/api/v2/tokens", `{"token":"T2"}`)), 200, "status", "token exists")
	expect(t, e.do(e.authed("POST", "/api/v2/tokens", `{"token":""}`)), 400, "error", "Invalid token")
	expect(t, e.do(e.authed("POST", "/api/v2/tokens", `not json`)), 400, "error", "invalid JSON")
	expect(t, e.do(e.authed("DELETE", "/api/v2/tokens/T2", "")), 200, "status", "deleted")
}

const moodlePush = `{
  "device": "android",
  "token": "DEVICE-1",
  "extra": {
    "processor": "moodle", "site": "abc", "sitefullname": "Springfield School",
    "smallmessage": "Your essay was graded", "userfromfullname": "Ms Krabappel",
    "notification": 1, "usertoid": 7, "encrypted": false, "customdata": {"a": [1, 2]}, "empty": null
  }
}`

func TestAMoodlePushBecomesANotificationWithAllItsDataAsStrings(t *testing.T) {
	e := newEnv(t, store.PermAll)
	e.do(e.authed("POST", "/tokens/DEVICE-1", ""))

	expect(t, e.do(e.authed("POST", "/api/v2/push/", moodlePush)), 202, "", "")
	if len(e.sender.sent) != 1 {
		t.Fatalf("expected one message, got %d", len(e.sender.sent))
	}
	got := e.sender.sent[0]
	if got.creds.ProjectID != "p" || got.msg.Token != "DEVICE-1" ||
		got.msg.Title != "Springfield School" || got.msg.Body != "Your essay was graded" {
		t.Fatalf("unexpected message %+v", got)
	}
	want := map[string]string{
		"processor": "moodle", "site": "abc", "sitefullname": "Springfield School", "smallmessage": "Your essay was graded",
		"userfromfullname": "Ms Krabappel", "notification": "1", "usertoid": "7", "encrypted": "0", "customdata": `{"a":[1,2]}`,
	}
	if len(got.msg.Data) != len(want) {
		t.Fatalf("data has %d entries, want %d: %v", len(got.msg.Data), len(want), got.msg.Data)
	}
	for k, v := range want {
		if got.msg.Data[k] != v {
			t.Errorf("data[%s] = %q, want %q", k, got.msg.Data[k], v)
		}
	}
}

func TestAMoodlePushFallsBackToTheLongMessageAndAGenericTitle(t *testing.T) {
	e := newEnv(t, store.PermAll)
	body := `{"token":"D","extra":{"processor":"moodle","fullmessage":"Long text"}}`
	expect(t, e.do(e.authed("POST", "/api/v2/push", body)), 202, "", "")
	if m := e.sender.sent[0].msg; m.Title != "Notification" || m.Body != "Long text" {
		t.Fatalf("unexpected message %+v", m)
	}
}

func TestHowTheAlertShapesTheMessage(t *testing.T) {
	e := newEnv(t, store.PermAll)
	cases := map[string]struct{ title, body string }{
		`{"token":"D","alert":null,"extra":{"processor":"moodle","smallmessage":"secret"}}`: {"", ""},
		`{"token":"D","alert":"Hello"}`:                  {"Hello", "Hello"},
		`{"token":"D","alert":{"title":"T","body":"B"}}`: {"T", "B"},
		`{"token":"D","extra":{"a":"b"}}`:                {"", ""},
	}
	for body, want := range cases {
		e.sender.sent = nil
		expect(t, e.do(e.authed("POST", "/api/v2/push/", body)), 202, "", "")
		if m := e.sender.sent[0].msg; m.Title != want.title || m.Body != want.body {
			t.Errorf("%s -> %+v, want %+v", body, m, want)
		}
	}
}

func TestExplicitFCMDataOverridesExtra(t *testing.T) {
	e := newEnv(t, store.PermAll)
	e.do(e.authed("POST", "/api/v2/push/", `{"token":"D","extra":{"a":"1","b":"2"},"fcm":{"data":{"b":"3","c":4}}}`))
	d := e.sender.sent[0].msg.Data
	if d["a"] != "1" || d["b"] != "3" || d["c"] != "4" {
		t.Fatalf("unexpected data %v", d)
	}
}

func TestPushRules(t *testing.T) {
	t.Run("needs permission to send", func(t *testing.T) {
		e := newEnv(t, store.PermCreateToken)
		expect(t, e.do(e.authed("POST", "/api/v2/push/", `{"token":"D"}`)), 403, "error", "No permission to send notification")
	})
	t.Run("registers an unknown token when allowed", func(t *testing.T) {
		e := newEnv(t, store.PermAll)
		expect(t, e.do(e.authed("POST", "/api/v2/push/", `{"token":"NEW","device":"ios"}`)), 202, "", "")
		if _, err := e.store.FindToken(e.keyRow.ID, "NEW"); err != nil {
			t.Fatalf("not registered: %v", err)
		}
	})
	t.Run("refuses an unknown token when not allowed", func(t *testing.T) {
		e := newEnv(t, store.PermSendNotification)
		expect(t, e.do(e.authed("POST", "/api/v2/push/", `{"token":"NEW"}`)), 400, "", "")
		if len(e.sender.sent) != 0 {
			t.Fatal("sent to an unregistered token")
		}
	})
	t.Run("bad bodies", func(t *testing.T) {
		e := newEnv(t, store.PermAll)
		expect(t, e.do(e.authed("POST", "/api/v2/push/", `nope`)), 400, "error", "invalid JSON")
		expect(t, e.do(e.authed("POST", "/api/v2/push/", `{}`)), 400, "error", "Invalid token")
	})
}

func TestFirebaseOutcomes(t *testing.T) {
	t.Run("a dead device is removed and reported as Moodle expects", func(t *testing.T) {
		e := newEnv(t, store.PermAll)
		e.do(e.authed("POST", "/tokens/DEAD", ""))
		e.sender.err = fcm.ErrUnregistered
		expect(t, e.do(e.authed("POST", "/api/v2/push/", `{"token":"DEAD"}`)), 404, "error", "Unregistered token")
		if _, err := e.store.FindToken(e.keyRow.ID, "DEAD"); !errors.Is(err, store.ErrNotFound) {
			t.Fatal("the dead token was kept")
		}
	})
	t.Run("another failure is a bad gateway and the token stays", func(t *testing.T) {
		e := newEnv(t, store.PermAll)
		e.do(e.authed("POST", "/tokens/OK", ""))
		e.sender.err = &fcm.Error{Status: 503, Stage: "send"}
		expect(t, e.do(e.authed("POST", "/api/v2/push/", `{"token":"OK"}`)), 502, "error", "error response from fcm")
		if _, err := e.store.FindToken(e.keyRow.ID, "OK"); err != nil {
			t.Fatal("the token was removed on a temporary failure")
		}
	})
	t.Run("an app without a Firebase key is unavailable", func(t *testing.T) {
		e := newEnv(t, store.PermAll)
		bare, _ := e.store.CreateApp("bare", "", "")
		key, _ := e.makeKey(bare.ID, store.PermAll)
		rec := e.do(request{method: "POST", path: "/api/v2/push/", body: `{"token":"D"}`, app: "bare", key: key})
		expect(t, rec, 503, "", "")
	})
	t.Run("a sealed key that cannot be opened is unavailable, not a crash", func(t *testing.T) {
		e := newEnv(t, store.PermAll)
		other, _ := secret.NewBox(strings.Repeat("z", 32))
		sealed, _ := other.Seal([]byte(serviceAccount))
		e.store.UpdateAppCredentials(e.app.ID, "p", sealed)
		expect(t, e.do(e.authed("POST", "/api/v2/push/", `{"token":"D"}`)), 503, "", "")
	})
}

func TestAccessKeyRequests(t *testing.T) {
	e := newEnv(t, store.PermAll)
	req := request{method: "POST", path: "/accesskeys/", app: "sayan", key: "none", form: true,
		body: url.Values{"url": {"https://school.example"}, "siteid": {"abc"}, "description": {"School"}}.Encode()}

	expect(t, e.do(req), 403, "", "")

	e.store.SetAllowKeyRequests(e.app.ID, true)
	rec := e.do(req)
	expect(t, rec, 200, "", "")
	issued, _ := decode(t, rec)["accesskey"].(string)
	if len(issued) != 32 {
		t.Fatalf("unexpected key %q", issued)
	}
	// The new key works at once.
	expect(t, e.do(request{method: "POST", path: "/tokens/T", app: "sayan", key: issued}), 200, "status", "ok")

	keys, _ := e.store.ListKeys(e.app.ID)
	if keys[0].Description != "School" || keys[0].Contact != "https://school.example" {
		t.Fatalf("the request details were not kept: %+v", keys[0])
	}

	expect(t, e.do(request{method: "POST", path: "/accesskeys/", app: "nope", form: true}), 400, "error", "Invalid application name")
	expect(t, e.do(request{method: "POST", path: "/accesskeys/", form: true}), 400, "error", "app name is required")
}

func TestNothingSensitiveReachesTheLog(t *testing.T) {
	e := newEnv(t, store.PermAll)
	e.do(e.authed("POST", "/api/v2/push/", `{"token":"A-VERY-LONG-DEVICE-TOKEN-1234567890","extra":{"processor":"moodle","smallmessage":"private words"}}`))

	logs, _ := e.store.RecentLogs(e.app.ID, 20)
	for _, l := range logs {
		if strings.Contains(l.Info, "private words") || strings.Contains(l.Info, "A-VERY-LONG-DEVICE-TOKEN-1234567890") || strings.Contains(l.Info, e.key) {
			t.Fatalf("the log leaked: %q", l.Info)
		}
	}
	if len(logs) == 0 {
		t.Fatal("the push was not logged")
	}
}
