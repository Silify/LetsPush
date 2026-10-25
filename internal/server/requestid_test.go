package server

import (
	"bytes"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/Silify/letspush/internal/reqid"
)

// captureLog sends the default logger to a buffer, as main sets it up, until the test ends.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(reqid.Handler(slog.NewTextHandler(&out, nil))))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &out
}

func TestEveryRequestGetsAnIDThatIsSentBackAndLogged(t *testing.T) {
	out := captureLog(t)
	h, _, _ := build(t, nil)

	generated := send(h, "GET", "/admin/api/me", "", nil, nil).Header().Get("X-Request-Id")
	if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(generated) {
		t.Fatalf("no generated ID: %q", generated)
	}

	kept := send(h, "GET", "/admin/api/me", "", map[string]string{"X-Request-Id": "site-42.a"}, nil)
	if got := kept.Header().Get("X-Request-Id"); got != "site-42.a" {
		t.Fatalf("the caller's ID was not kept: %q", got)
	}

	bad := send(h, "GET", "/admin/api/me", "", map[string]string{"X-Request-Id": "has spaces <b>"}, nil)
	if got := bad.Header().Get("X-Request-Id"); got == "has spaces <b>" || got == "" {
		t.Fatalf("a bad ID was echoed: %q", got)
	}

	for _, id := range []string{generated, "site-42.a"} {
		if !strings.Contains(out.String(), "request_id="+id) {
			t.Errorf("the access log has no line for request %s:\n%s", id, out.String())
		}
	}
	if !strings.Contains(out.String(), "status=401") || !strings.Contains(out.String(), "duration_ms=") {
		t.Fatalf("the access log lacks fields:\n%s", out.String())
	}
}
