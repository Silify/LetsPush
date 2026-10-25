// Package server puts the pieces together: the compatible API, the admin API and the web interface.
package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Silify/letspush/internal/admin"
	"github.com/Silify/letspush/internal/compat"
	"github.com/Silify/letspush/internal/gateway"
	"github.com/Silify/letspush/internal/reqid"
	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
	"github.com/Silify/letspush/internal/webui"
)

// Options are what the server is built from.
type Options struct {
	Store   *store.Store
	Box     *secret.Box
	Gateway *gateway.Service
	Secret  string
	Now     func() time.Time
	// UI serves everything that is not an API call. It defaults to the built-in interface.
	UI http.Handler
}

// New returns the whole HTTP handler.
func New(o Options) (http.Handler, error) {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.UI == nil {
		o.UI = webui.Handler()
	}

	adminAPI, err := admin.New(o.Store, o.Box, o.Secret, o.Now)
	if err != nil {
		return nil, err
	}
	compatAPI := compat.New(o.Store, o.Gateway)

	mux := http.NewServeMux()
	// Longer, more specific patterns win, so the interface only gets what no API claims.
	mux.Handle("/admin/api/", adminAPI)
	for _, route := range []string{
		"POST /accesskeys/",
		"/tokens/",
		"/api/v2/",
	} {
		mux.Handle(route, compatAPI)
	}
	mux.Handle("/", o.UI)

	return chain(mux, requestID, recoverPanics, securityHeaders, accessLog), nil
}

type middleware func(http.Handler) http.Handler

func chain(h http.Handler, layers ...middleware) http.Handler {
	for i := len(layers) - 1; i >= 0; i-- {
		h = layers[i](h)
	}
	return h
}

// requestID gives the request an ID, taken from the caller's X-Request-Id header when it is fit to be
// logged, and sends it back in the answer. Every log line about the request carries it.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := reqid.Accept(r.Header.Get("X-Request-Id"))
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(reqid.With(r.Context(), id)))
	})
}

// recoverPanics keeps one bad request from taking the server down.
func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(r.Context(), "panic while serving a request", "path", logPath(r), "panic", recovered)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"something went wrong"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// securityHeaders applies to every answer.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

// statusWriter remembers the status for the access log.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// accessLog writes one line per request: method, path, status and time. No headers, no bodies.
// Server errors are logged as errors.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		level := slog.LevelInfo
		if sw.status >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		slog.LogAttrs(r.Context(), level, "request",
			slog.String("method", r.Method),
			slog.String("path", logPath(r)),
			slog.Int("status", sw.status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()))
	})
}

// logPath hides device tokens, which are part of the path of some requests.
func logPath(r *http.Request) string {
	path := r.URL.Path
	for _, prefix := range []string{"/tokens/", "/api/v2/tokens/"} {
		if strings.HasPrefix(path, prefix) && len(path) > len(prefix) {
			return prefix + "…"
		}
	}
	return path
}
