// Package reqid gives every request an ID that follows it into the logs.
package reqid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"regexp"
)

type ctxKey struct{}

// valid is what an ID sent in by a caller may look like. Anything else is replaced.
var valid = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// New returns a fresh ID.
func New() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Accept returns [id] when it is fit to be logged and echoed back, and a fresh ID otherwise.
func Accept(id string) string {
	if valid.MatchString(id) {
		return id
	}
	return New()
}

// With returns a context that carries the ID.
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// From returns the ID in the context, or "".
func From(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// Handler wraps a log handler so that every record logged with a context that carries an ID
// (slog.InfoContext and friends) gets a request_id field.
func Handler(next slog.Handler) slog.Handler { return handler{next} }

type handler struct{ slog.Handler }

func (h handler) Handle(ctx context.Context, r slog.Record) error {
	if id := From(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return handler{h.Handler.WithAttrs(attrs)}
}
func (h handler) WithGroup(name string) slog.Handler { return handler{h.Handler.WithGroup(name)} }
