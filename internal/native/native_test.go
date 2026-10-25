package native

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Silify/letspush/internal/gateway"
	"github.com/Silify/letspush/internal/reqid"
	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
)

type sent struct {
	device gateway.Device
	n      gateway.Notification
}

type fakeProvider struct {
	mu   sync.Mutex
	sent []sent
	err  error
}

func (f *fakeProvider) Send(_ context.Context, _ []byte, device gateway.Device, n gateway.Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, sent{device, n})
	return nil
}

type env struct {
	t        *testing.T
	store    *store.Store
	provider *fakeProvider
	service  *gateway.Service
	handler  http.Handler
	app      store.App
	key      string
	keyRow   store.AccessKey
}

func newEnv(t *testing.T, permissions int) *env {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	box, _ := secret.NewBox(strings.Repeat("k", 32))
	sealed, _ := box.Seal([]byte("the key"))
	app, err := db.CreateApp("sayan", store.Provider{Kind: store.ProviderFCM, Label: "p", Credentials: sealed})
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, store: db, provider: &fakeProvider{}, app: app}
	e.service = gateway.New(gateway.Config{
		Store:     db,
		Opener:    box,
		Providers: map[string]gateway.Provider{store.ProviderFCM: e.provider},
		Wait:      time.Second,
		Timeout:   5 * time.Second,
	})
	t.Cleanup(e.service.Wait)
	e.handler = New(db, e.service)
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

// call makes a request with the environment's key; pass key "" to send none.
func (e *env) call(method, path, body, key string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = req.WithContext(reqid.With(req.Context(), "req-test"))
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func (e *env) as(method, path, body string) *httptest.ResponseRecorder {
	return e.call(method, path, body, e.key)
}

func answerOf(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("not JSON: %q", rec.Body.String())
		}
	}
	return out
}

// wantError checks the status and the error code, and that the one error shape is used.
func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d (%s)", rec.Code, status, rec.Body.String())
	}
	e, _ := answerOf(t, rec)["error"].(map[string]any)
	if e["code"] != code || e["message"] == "" || e["requestId"] != "req-test" {
		t.Fatalf("unexpected error %v, want code %q", e, code)
	}
}

func TestEveryRouteNeedsAValidBearerKey(t *testing.T) {
	e := newEnv(t, store.PermAll)
	revoked, row := e.makeKey(e.app.ID, store.PermAll)
	e.store.RevokeKey(e.app.ID, row.ID)

	for _, route := range [][2]string{
		{"POST", "/v1/devices"}, {"DELETE", "/v1/devices/T"}, {"POST", "/v1/notifications"}, {"GET", "/v1/notifications/ntf_1"},
	} {
		wantError(t, e.call(route[0], route[1], `{}`, ""), 401, "missing_key")
		wantError(t, e.call(route[0], route[1], `{}`, "wrong"), 401, "invalid_key")
		wantError(t, e.call(route[0], route[1], `{}`, revoked), 401, "invalid_key")
	}

	req := httptest.NewRequest("POST", "/v1/devices", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Basic "+e.key)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("a non-Bearer scheme was accepted: %d", rec.Code)
	}

	wantError(t, e.as("GET", "/v1/nothing", ""), 404, "not_found")
}

func TestRegisteringAndRemovingADevice(t *testing.T) {
	e := newEnv(t, store.PermAll)

	rec := e.as("POST", "/v1/devices", `{"token":"T1","platform":"android"}`)
	if rec.Code != 201 || answerOf(t, rec)["created"] != true {
		t.Fatalf("unexpected answer %d %s", rec.Code, rec.Body.String())
	}
	rec = e.as("POST", "/v1/devices", `{"token":"T1","platform":"android"}`)
	if rec.Code != 200 || answerOf(t, rec)["created"] != false {
		t.Fatalf("registering again: %d %s", rec.Code, rec.Body.String())
	}
	if got, err := e.store.FindToken(e.keyRow.ID, "T1"); err != nil || got.Device != "android" {
		t.Fatalf("stored wrongly: %v %+v", err, got)
	}

	if rec := e.as("DELETE", "/v1/devices/T1", ""); rec.Code != 204 || rec.Body.Len() != 0 {
		t.Fatalf("removing: %d %s", rec.Code, rec.Body.String())
	}
	wantError(t, e.as("DELETE", "/v1/devices/T1", ""), 404, "not_found")
}

func TestRegisteringADeviceIsChecked(t *testing.T) {
	e := newEnv(t, store.PermAll)
	wantError(t, e.as("POST", "/v1/devices", `{"token":"T"}`), 400, "invalid_request")
	wantError(t, e.as("POST", "/v1/devices", `{"token":"T","platform":"Not Valid!"}`), 400, "invalid_request")
	wantError(t, e.as("POST", "/v1/devices", `{"token":"","platform":"web"}`), 400, "invalid_request")
	wantError(t, e.as("POST", "/v1/devices", `{"token":"T","platform":"web","colour":"red"}`), 400, "invalid_request")
	wantError(t, e.as("POST", "/v1/devices", `nope`), 400, "invalid_request")

	limited := newEnv(t, store.PermSendNotification)
	wantError(t, limited.as("POST", "/v1/devices", `{"token":"T","platform":"web"}`), 403, "forbidden")
	wantError(t, limited.as("DELETE", "/v1/devices/T", ""), 403, "forbidden")
}

const fullRequest = `{
  "to": {"token": "DEVICE-1", "platform": "Android"},
  "title": "Assignment graded",
  "body": "Your essay was graded",
  "image": "https://school.example/a.png",
  "data": {"course": "12", "deepLink": "/mod/assign/5"},
  "options": {"priority": "normal", "ttl": 3600, "sound": "default"}
}`

func TestSendingANotificationWithEverythingTheAPIOffers(t *testing.T) {
	e := newEnv(t, store.PermAll)
	rec := e.as("POST", "/v1/notifications", fullRequest)
	if rec.Code != 202 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	answer := answerOf(t, rec)
	id, _ := answer["id"].(string)
	if !strings.HasPrefix(id, "ntf_") || answer["status"] != "sent" {
		t.Fatalf("unexpected answer %v", answer)
	}

	if len(e.provider.sent) != 1 {
		t.Fatalf("expected one message, got %d", len(e.provider.sent))
	}
	got := e.provider.sent[0]
	if got.device.Token != "DEVICE-1" || got.device.Platform != "android" {
		t.Fatalf("unexpected device %+v", got.device)
	}
	want := gateway.Notification{
		Title: "Assignment graded", Body: "Your essay was graded", Image: "https://school.example/a.png",
		Data:     map[string]string{"course": "12", "deepLink": "/mod/assign/5"},
		Priority: "normal", TTL: time.Hour, Sound: "default",
	}
	if got.n.Title != want.Title || got.n.Body != want.Body || got.n.Image != want.Image ||
		got.n.Priority != want.Priority || got.n.TTL != want.TTL || got.n.Sound != want.Sound ||
		len(got.n.Data) != 2 || got.n.Data["course"] != "12" || got.n.Data["deepLink"] != "/mod/assign/5" {
		t.Fatalf("unexpected notification %+v", got.n)
	}

	status := e.as("GET", "/v1/notifications/"+id, "")
	body := answerOf(t, status)
	if status.Code != 200 || body["status"] != "sent" || body["requestId"] != "req-test" || body["createdAt"] == "" {
		t.Fatalf("status lookup: %d %v", status.Code, body)
	}
}

func TestAMinimalNotificationAndADataOnlyOne(t *testing.T) {
	e := newEnv(t, store.PermAll)
	if rec := e.as("POST", "/v1/notifications", `{"to":{"token":"D"},"title":"Hi"}`); rec.Code != 202 {
		t.Fatalf("title only: %d %s", rec.Code, rec.Body.String())
	}
	if rec := e.as("POST", "/v1/notifications", `{"to":{"token":"D"},"data":{"a":"1"}}`); rec.Code != 202 {
		t.Fatalf("data only: %d %s", rec.Code, rec.Body.String())
	}
	if n := e.provider.sent[1].n; n.Title != "" || n.Body != "" || n.Data["a"] != "1" {
		t.Fatalf("a data-only notification was not data only: %+v", n)
	}
}

func TestARequestThatIsNotValidIsRefusedBeforeAnythingIsSent(t *testing.T) {
	e := newEnv(t, store.PermAll)
	big := strings.Repeat("x", 4001)
	cases := map[string]string{
		"no token":            `{"title":"Hi"}`,
		"nothing to show":     `{"to":{"token":"D"}}`,
		"empty data":          `{"to":{"token":"D"},"data":{}}`,
		"image alone":         `{"to":{"token":"D"},"image":"https://a.example/i.png"}`,
		"image over http":     `{"to":{"token":"D"},"title":"t","image":"http://a.example/i.png"}`,
		"image not a url":     `{"to":{"token":"D"},"title":"t","image":"nope"}`,
		"data not strings":    `{"to":{"token":"D"},"data":{"a":1}}`,
		"empty data key":      `{"to":{"token":"D"},"data":{"":"x"}}`,
		"reserved data key":   `{"to":{"token":"D"},"data":{"from":"x"}}`,
		"reserved google key": `{"to":{"token":"D"},"data":{"google.sent_time":"x"}}`,
		"reserved gcm key":    `{"to":{"token":"D"},"data":{"GCM.x":"x"}}`,
		"too large":           `{"to":{"token":"D"},"body":"` + big + `"}`,
		"bad priority":        `{"to":{"token":"D"},"title":"t","options":{"priority":"urgent"}}`,
		"negative ttl":        `{"to":{"token":"D"},"title":"t","options":{"ttl":-1}}`,
		"ttl too long":        `{"to":{"token":"D"},"title":"t","options":{"ttl":2419201}}`,
		"sound with control":  `{"to":{"token":"D"},"title":"t","options":{"sound":"a\u0000b"}}`,
		"bad platform":        `{"to":{"token":"D","platform":"No Good"},"title":"t"}`,
		"unknown field":       `{"to":{"token":"D"},"title":"t","colour":"red"}`,
		"unknown option":      `{"to":{"token":"D"},"title":"t","options":{"vibrate":true}}`,
		"not json":            `nope`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) { wantError(t, e.as("POST", "/v1/notifications", body), 400, "invalid_request") })
	}
	if len(e.provider.sent) != 0 {
		t.Fatal("something was sent for an invalid request")
	}
}

func TestSendingFailsWithTheCodeThatSaysWhy(t *testing.T) {
	t.Run("a key that may not send", func(t *testing.T) {
		e := newEnv(t, store.PermCreateToken)
		wantError(t, e.as("POST", "/v1/notifications", `{"to":{"token":"D"},"title":"t"}`), 403, "forbidden")
	})
	t.Run("an unknown device and a key that may not register it", func(t *testing.T) {
		e := newEnv(t, store.PermSendNotification)
		wantError(t, e.as("POST", "/v1/notifications", `{"to":{"token":"D"},"title":"t"}`), 404, "unknown_device")
	})
	t.Run("a dead device", func(t *testing.T) {
		e := newEnv(t, store.PermAll)
		e.provider.err = gateway.ErrUnregistered
		wantError(t, e.as("POST", "/v1/notifications", `{"to":{"token":"D"},"title":"t"}`), 404, "device_unregistered")
		if _, err := e.store.FindToken(e.keyRow.ID, "D"); !errors.Is(err, store.ErrNotFound) {
			t.Fatal("the dead device was kept")
		}
	})
	t.Run("a refusal", func(t *testing.T) {
		e := newEnv(t, store.PermAll)
		e.provider.err = errors.New("firebase send failed with status 403: PERMISSION_DENIED: no")
		rec := e.as("POST", "/v1/notifications", `{"to":{"token":"D"},"title":"t"}`)
		wantError(t, rec, 502, "delivery_failed")
		if strings.Contains(rec.Body.String(), "PERMISSION_DENIED") {
			t.Fatal("the push service's text was given to the caller")
		}
	})
	t.Run("an app without a key", func(t *testing.T) {
		e := newEnv(t, store.PermAll)
		bare, _ := e.store.CreateApp("bare")
		key, _ := e.makeKey(bare.ID, store.PermAll)
		wantError(t, e.call("POST", "/v1/notifications", `{"to":{"token":"D"},"title":"t"}`, key), 503, "no_provider")
	})
}

func TestAnUnknownDeviceIsRegisteredWhenTheKeyMayAndKeepsItsPlatform(t *testing.T) {
	e := newEnv(t, store.PermAll)
	e.as("POST", "/v1/notifications", `{"to":{"token":"NEW","platform":"web"},"title":"t"}`)
	if got, err := e.store.FindToken(e.keyRow.ID, "NEW"); err != nil || got.Device != "web" {
		t.Fatalf("stored wrongly: %v %+v", err, got)
	}
}

func TestADeliveryCanOnlyBeAskedAboutByTheKeyThatStartedIt(t *testing.T) {
	e := newEnv(t, store.PermAll)
	id := answerOf(t, e.as("POST", "/v1/notifications", `{"to":{"token":"D"},"title":"t"}`))["id"].(string)

	otherKey, _ := e.makeKey(e.app.ID, store.PermAll)
	wantError(t, e.call("GET", "/v1/notifications/"+id, "", otherKey), 404, "not_found")
	wantError(t, e.as("GET", "/v1/notifications/ntf_nope", ""), 404, "not_found")
}
