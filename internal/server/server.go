// Package server puts the pieces together: the compatible API, the admin API and the web interface.
package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Silify/letspush/internal/admin"
	"github.com/Silify/letspush/internal/compat"
	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
	"github.com/Silify/letspush/internal/webui"
)

// Options are what the server is built from.
type Options struct {
	Store  *store.Store
	Box    *secret.Box
	Sender compat.Sender
	Secret string
	Now    func() time.Time
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
	compatAPI := compat.New(o.Store, o.Sender, o.Box)

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

	return chain(mux, recoverPanics, securityHeaders, accessLog), nil
}

type middleware func(http.Handler) http.Handler

func chain(h http.Handler, layers ...middleware) http.Handler {
	for i := len(layers) - 1; i >= 0; i-- {
		h = layers[i](h)
	}
	return h
}

// recoverPanics keeps one bad request from taking the server down.
func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic while serving %s: %v", logPath(r), recovered)
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
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, logPath(r), sw.status, time.Since(start).Round(time.Millisecond))
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
