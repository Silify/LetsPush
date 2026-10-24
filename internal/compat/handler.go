// Package compat serves the push API that Moodle's mobile notifications speak:
// access-key requests, device tokens and sending.
package compat

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Silify/letspush/internal/fcm"
	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
)

// Store is the part of the database this API uses.
type Store interface {
	AppByName(name string) (store.App, error)
	AppCredentials(appID int64) (string, error)
	KeyByHash(hash string) (store.AccessKey, error)
	TouchKey(id int64) error
	CreateKey(store.NewKey) (store.AccessKey, error)
	UpsertToken(store.Token) (bool, error)
	FindToken(keyID int64, token string) (store.Token, error)
	DeleteToken(keyID int64, token string) error
	AddLog(appID int64, action, info string) error
}

// Sender delivers a message to Firebase.
type Sender interface {
	Send(ctx context.Context, creds fcm.Credentials, msg fcm.Message) error
}

// Opener opens the sealed Firebase keys.
type Opener interface {
	Open(sealed string) ([]byte, error)
}

// Handler is the compatible API.
type Handler struct {
	store  Store
	sender Sender
	opener Opener
}

// New returns the API's routes.
func New(s Store, sender Sender, opener Opener) http.Handler {
	h := &Handler{store: s, sender: sender, opener: opener}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /accesskeys/", h.requestAccessKey)

	mux.HandleFunc("POST /tokens/{token}", h.createToken)
	mux.HandleFunc("DELETE /tokens/{token}", h.deleteToken)
	mux.HandleFunc("POST /api/v2/tokens", h.createTokenV2)
	mux.HandleFunc("POST /api/v2/tokens/", h.createTokenV2)
	mux.HandleFunc("DELETE /api/v2/tokens/{token}", h.deleteToken)

	mux.HandleFunc("POST /api/v2/push", h.push)
	mux.HandleFunc("POST /api/v2/push/", h.push)
	return mux
}

const (
	headerApp = "X-An-App-Name"
	headerKey = "X-An-App-Key"

	maxBody = 64 << 10
)

// session is a request that passed the app name and access key checks.
type session struct {
	app store.App
	key store.AccessKey
}

// authenticate checks the headers every request carries. It writes the error itself.
// The access key is only required when [needKey] is set.
func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request, needKey bool) (session, bool) {
	name := strings.TrimSpace(r.Header.Get(headerApp))
	if name == "" {
		writeError(w, http.StatusBadRequest, "app name is required")
		return session{}, false
	}
	rawKey := r.Header.Get(headerKey)
	if needKey && rawKey == "" {
		writeError(w, http.StatusBadRequest, "app key is required")
		return session{}, false
	}

	app, err := h.store.AppByName(name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application name")
		return session{}, false
	}
	if !needKey {
		return session{app: app}, true
	}

	key, err := h.store.KeyByHash(secret.Hash(rawKey))
	if err != nil || key.AppID != app.ID {
		writeError(w, http.StatusBadRequest, "Invalid access key")
		return session{}, false
	}
	_ = h.store.TouchKey(key.ID)
	return session{app: app, key: key}, true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
