package reqid

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestAcceptKeepsAGoodIDAndReplacesABadOne(t *testing.T) {
	if got := Accept("abc-123_X.y"); got != "abc-123_X.y" {
		t.Fatalf("a good ID was changed: %q", got)
	}
	for _, bad := range []string{"", "has space", "new\nline", strings.Repeat("a", 65), "<script>"} {
		got := Accept(bad)
		if got == bad || len(got) != 16 {
			t.Errorf("%q should have been replaced by a fresh ID, got %q", bad, got)
		}
	}
	if New() == New() {
		t.Fatal("IDs repeat")
	}
}

func TestTheLogGetsTheRequestIDOfItsContext(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(Handler(slog.NewTextHandler(&out, nil))).With("app", "sayan")

	logger.InfoContext(With(context.Background(), "req-1"), "with")
	logger.Info("without")

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "request_id=req-1") || !strings.Contains(lines[0], "app=sayan") {
		t.Fatalf("unexpected first line %q", lines)
	}
	if strings.Contains(lines[1], "request_id") {
		t.Fatalf("a record with no ID got one: %q", lines[1])
	}
}
