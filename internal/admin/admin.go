package admin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/Silify/letspush/internal/store"
)

// Store is the part of the database the admin API uses.
type Store interface {
	AdminByEmail(email string) (store.Admin, error)
	AdminByID(id int64) (store.Admin, error)
	SetAdminPassword(id int64, passwordHash string) error

	ListApps() ([]store.App, error)
	AppByID(id int64) (store.App, error)
	CreateApp(name string, providers ...store.Provider) (store.App, error)
	SetProvider(appID int64, p store.Provider) error
	SetAllowKeyRequests(id int64, allow bool) error
	DeleteApp(id int64) error

	CreateKey(store.NewKey) (store.AccessKey, error)
	ListKeys(appID int64) ([]store.AccessKey, error)
	RevokeKey(appID, keyID int64) error

	ListLogs(store.LogFilter) ([]store.LogEntry, error)
	CountLogs(appID int64, action, infoPrefix string, since time.Time) (int, error)
	AddLog(appID int64, action, info string) error

	Totals() (store.Totals, error)
	ListDevices(appID, beforeID int64, limit int) ([]store.DeviceRow, error)
	DeleteDevice(appID, id int64) error
}

// Sealer encrypts the Firebase keys before they are stored.
type Sealer interface {
	Seal(plain []byte) (string, error)
}

// Handler is the admin API.
type Handler struct {
	store    Store
	sealer   Sealer
	sessions *sessions
	throttle *throttle
	now      func() time.Time
	// dummyHash is checked when an email is unknown, so timing does not reveal which emails exist.
	dummyHash string
}

// New returns the API's routes, all under /admin/api/.
func New(s Store, sealer Sealer, serverSecret string, now func() time.Time) (http.Handler, error) {
	dummy, err := hashForTiming()
	if err != nil {
		return nil, err
	}
	h := &Handler{
		store:     s,
		sealer:    sealer,
		sessions:  newSessions(serverSecret, now),
		throttle:  newThrottle(now),
		now:       now,
		dummyHash: dummy,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/api/login", h.login)
	mux.HandleFunc("POST /admin/api/logout", h.logout)

	mux.Handle("GET /admin/api/me", h.protect(h.me))
	mux.Handle("POST /admin/api/password", h.protect(h.changePassword))

	mux.Handle("GET /admin/api/apps", h.protect(h.listApps))
	mux.Handle("POST /admin/api/apps", h.protect(h.createApp))
	mux.Handle("GET /admin/api/apps/{id}", h.protect(h.showApp))
	mux.Handle("DELETE /admin/api/apps/{id}", h.protect(h.deleteApp))
	mux.Handle("PUT /admin/api/apps/{id}/credentials", h.protect(h.setCredentials))
	mux.Handle("PUT /admin/api/apps/{id}/settings", h.protect(h.setSettings))
	mux.Handle("GET /admin/api/apps/{id}/logs", h.protect(h.logs))
	mux.Handle("GET /admin/api/apps/{id}/devices", h.protect(h.devices))
	mux.Handle("DELETE /admin/api/apps/{id}/devices/{deviceId}", h.protect(h.removeDevice))

	mux.Handle("GET /admin/api/overview", h.protect(h.overview))
	mux.Handle("GET /admin/api/logs", h.protect(h.logs))

	mux.Handle("POST /admin/api/apps/{id}/keys", h.protect(h.createKey))
	mux.Handle("DELETE /admin/api/apps/{id}/keys/{keyId}", h.protect(h.revokeKey))

	// Anything else under the prefix is a JSON 404, not the web page.
	mux.HandleFunc("/admin/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	return mux, nil
}

type adminKey struct{}

// protect requires a signed-in admin. Requests that change something must also carry the header
// a page on another site cannot add, on top of the cookie's SameSite=Strict.
func (h *Handler) protect(next func(http.ResponseWriter, *http.Request, store.Admin)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := h.sessions.adminID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "sign in first")
			return
		}
		admin, err := h.store.AdminByID(id)
		if err != nil {
			h.sessions.clear(w, r)
			writeError(w, http.StatusUnauthorized, "sign in first")
			return
		}
		if r.Method != http.MethodGet && r.Header.Get("X-Requested-With") != "letspush" {
			writeError(w, http.StatusForbidden, "missing X-Requested-With header")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), adminKey{}, admin)), admin)
	})
}

const maxJSON = 1 << 20

var appName = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

func readJSON(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, maxJSON)).Decode(into); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// storeError answers for a failed database call.
func storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, "that already exists")
	default:
		writeError(w, http.StatusInternalServerError, "something went wrong")
	}
}
