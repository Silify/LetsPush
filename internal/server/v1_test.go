package server

import (
	"strings"
	"testing"
)

func TestTheNativeAPIIsMountedAndItsDeviceTokensStayOutOfTheLog(t *testing.T) {
	out := captureLog(t)
	h, _, _ := build(t, nil)

	rec := send(h, "DELETE", "/v1/devices/SECRET-DEVICE-TOKEN", "", nil, nil)
	if rec.Code != 401 || !strings.Contains(rec.Body.String(), `"code":"missing_key"`) {
		t.Fatalf("the native API did not answer: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(out.String(), "SECRET") {
		t.Fatalf("a device token reached the log: %s", out.String())
	}
	if !strings.Contains(out.String(), "/v1/devices/…") {
		t.Fatalf("the request was not logged: %s", out.String())
	}

	// An endpoint that does not exist is a JSON 404, not the web page.
	if rec := send(h, "GET", "/v1/nothing", "", nil, nil); rec.Code != 404 || !strings.Contains(rec.Body.String(), `"code":"not_found"`) {
		t.Fatalf("unexpected answer %d %s", rec.Code, rec.Body.String())
	}
}
