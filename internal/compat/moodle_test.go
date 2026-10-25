package compat

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Silify/letspush/internal/gateway"
	"github.com/Silify/letspush/internal/store"
)

// These tests send requests the way Moodle's message_airnotifier plugin builds them (version 2026100500,
// message_output_airnotifier.php and classes/manager.php): its headers, its form encoding, its JSON. A change
// that breaks one of them breaks real Moodle sites, so the expectations here should not be loosened to make
// a test pass.

// moodleRequest is a request as the plugin's curl sends it: it sets only these headers, and a body it
// posts as a string (the push JSON) goes out with the default form content type.
func (e *env) moodleRequest(method, path, body string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-AN-APP-NAME", "sayan")
	req.Header.Set("X-AN-APP-KEY", e.key)
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

// registered is how manager.php registers a device: POST /tokens/{token} with the platform in a form.
func (e *env) registered(token, platform string) {
	e.t.Helper()
	body := url.Values{"device": {platform}}.Encode()
	expect(e.t, e.moodleRequest("POST", "/tokens/"+token, body), 200, "status", "ok")
}

// The extra of a notification the way send_message builds it, before encryption.
const moodleExtra = `{
    "component": "mod_assign", "name": "assign_notification",
    "userfromid": 12, "userfromfullname": "Ms Krabappel", "usertoid": 7,
    "subject": "Essay graded", "smallmessage": "Your essay was graded",
    "contexturl": "https://school.example/mod/assign/view.php?id=5",
    "timecreated": 1760000000, "processor": "moodle", "site": "6b4c3b0a2f0e4f5d9d6f3b6f0c2d7a11",
    "date": 1760000000, "notification": 1, "sitefullname": "Springfield School", "siteshortname": "SPS",
    "wwwroot": "https://school.example"`

func TestAPlainMoodleNotificationAsTheCurrentPluginSendsIt(t *testing.T) {
	e := newEnv(t, store.PermAll)
	e.registered("fcm-token-1", "Android")

	body := `{"device":"Android","token":"fcm-token-1","extra":` + moodleExtra + `,"encrypted":false}}`
	expect(t, e.moodleRequest("POST", "/api/v2/push/", body), 202, "status", "accepted")

	if len(e.sender.sent) != 1 {
		t.Fatalf("expected one message, got %d", len(e.sender.sent))
	}
	got := e.sender.sent[0]
	if got.device.Token != "fcm-token-1" || got.device.Platform != "android" {
		t.Fatalf("unexpected device %+v", got.device)
	}
	if got.msg.Title != "Springfield School" || got.msg.Body != "Your essay was graded" {
		t.Fatalf("unexpected text %q / %q", got.msg.Title, got.msg.Body)
	}
	for k, want := range map[string]string{
		"component": "mod_assign", "userfromid": "12", "usertoid": "7", "notification": "1",
		"encrypted": "0", "timecreated": "1760000000", "wwwroot": "https://school.example", "processor": "moodle",
		"contexturl": "https://school.example/mod/assign/view.php?id=5",
	} {
		if got.msg.Data[k] != want {
			t.Errorf("data[%s] = %q, want %q", k, got.msg.Data[k], want)
		}
	}
}

func TestAnEncryptedMoodleNotificationIsDeliveredAsDataOnly(t *testing.T) {
	e := newEnv(t, store.PermAll)
	e.registered("fcm-token-2", "Android")

	// With encryption the plugin sets alert to null and seals these fields with the device's public key
	// (base64), so nothing readable may be turned into a visible notification.
	const sealed = "kOa3sF0iWwq2o6c9lB8y7x1m3n5p0r2t4v6x8z0b2d4f6h8j0lQ"
	extra := strings.Replace(moodleExtra, `"userfromfullname": "Ms Krabappel"`, `"userfromfullname": "`+sealed+`"`, 1)
	extra = strings.Replace(extra, `"smallmessage": "Your essay was graded"`, `"smallmessage": "`+sealed+`"`, 1)
	extra = strings.Replace(extra, `"sitefullname": "Springfield School"`, `"sitefullname": "`+sealed+`"`, 1)
	body := `{"device":"Android","token":"fcm-token-2","extra":` + extra + `,"encrypted":true},"alert":null}`

	expect(t, e.moodleRequest("POST", "/api/v2/push/", body), 202, "status", "accepted")

	got := e.sender.sent[0]
	if got.msg.Title != "" || got.msg.Body != "" {
		t.Fatalf("an encrypted notification was shown in the clear: %q / %q", got.msg.Title, got.msg.Body)
	}
	if got.msg.Data["encrypted"] != "1" {
		t.Fatalf("the app is not told the message is encrypted: %v", got.msg.Data)
	}
	for _, field := range []string{"userfromfullname", "smallmessage", "sitefullname"} {
		if got.msg.Data[field] != sealed {
			t.Errorf("data[%s] = %q, want the sealed value unchanged", field, got.msg.Data[field])
		}
	}
}

func TestTheDevicePlatformsMoodleRegistersAreAccepted(t *testing.T) {
	e := newEnv(t, store.PermAll)
	for platform, stored := range map[string]string{"Android": "android", "iOS": "ios", "iOS-fcm": "ios-fcm"} {
		token := "token-" + stored
		e.registered(token, platform)

		found, err := e.store.FindToken(e.keyRow.ID, token)
		if err != nil || found.Device != stored {
			t.Fatalf("%s: stored %+v, %v", platform, found, err)
		}
		body := `{"device":"` + platform + `","token":"` + token + `","extra":` + moodleExtra + `}}`
		expect(t, e.moodleRequest("POST", "/api/v2/push/", body), 202, "status", "accepted")
	}
	if len(e.sender.sent) != 3 {
		t.Fatalf("expected three messages, got %d", len(e.sender.sent))
	}
}

func TestTheUnregisteredTokenAnswerIsWhatMoodleLooksFor(t *testing.T) {
	// manager.php deletes the device only when the status is 404 and the body's "error" is exactly this text.
	e := newEnv(t, store.PermAll)
	e.registered("gone", "Android")
	e.sender.err = gateway.ErrUnregistered

	rec := e.moodleRequest("POST", "/api/v2/push/", `{"device":"Android","token":"gone","extra":`+moodleExtra+`}}`)
	expect(t, rec, 404, "error", "Unregistered token")
}
