// Package native serves LetsPush's own API, under /v1: register devices, send notifications with a title,
// body, image and data, and ask how a delivery ended. It is the API for new integrations; the Moodle-compatible
// one is in package compat.
package native

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Silify/letspush/internal/gateway"
	"github.com/Silify/letspush/internal/reqid"
	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
)

// Store is the part of the database this API uses itself; devices and sending go through the gateway.
type Store interface {
	KeyByHash(hash string) (store.AccessKey, error)
	TouchKey(id int64) error
	AppByID(id int64) (store.App, error)
	DeliveryByID(keyID int64, id string) (store.Delivery, error)
}

// Handler is the native API.
type Handler struct {
	store   Store
	gateway *gateway.Service
}

// New returns the API's routes, all under /v1/.
func New(s Store, gw *gateway.Service) http.Handler {
	h := &Handler{store: s, gateway: gw}
	mux := http.NewServeMux()

	mux.Handle("POST /v1/devices", h.authenticated(h.registerDevice))
	mux.Handle("DELETE /v1/devices/{token}", h.authenticated(h.removeDevice))
	mux.Handle("POST /v1/notifications", h.authenticated(h.sendNotification))
	mux.Handle("GET /v1/notifications/{id}", h.authenticated(h.showNotification))

	// Anything else under the prefix is a JSON 404.
	mux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, "not_found", "no such endpoint")
	})
	return mux
}

const maxBody = 64 << 10

// session is a request that carried a valid access key.
type session struct {
	app store.App
	key store.AccessKey
}

// authenticated checks the Bearer access key and hands the handler the key's app.
func (h *Handler) authenticated(next func(http.ResponseWriter, *http.Request, session)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, raw, _ := strings.Cut(r.Header.Get("Authorization"), " ")
		raw = strings.TrimSpace(raw)
		if !strings.EqualFold(scheme, "Bearer") || raw == "" {
			writeError(w, r, http.StatusUnauthorized, "missing_key", "send the access key as: Authorization: Bearer <key>")
			return
		}
		key, err := h.store.KeyByHash(secret.Hash(raw))
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "invalid_key", "the access key is unknown or revoked")
			return
		}
		app, err := h.store.AppByID(key.AppID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "invalid_key", "the access key is unknown or revoked")
			return
		}
		_ = h.store.TouchKey(key.ID)
		next(w, r, session{app: app, key: key})
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeError sends the one error shape of this API:
//
//	{"error": {"code": "invalid_request", "message": "…", "requestId": "…"}}
//
// The code is stable and meant for programs; the message is for people.
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{
		"code": code, "message": message, "requestId": reqid.From(r.Context()),
	}})
}

// decode reads a JSON body. Fields the API does not know are an error, so a misspelt option is not
// silently ignored.
func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "the body is not valid JSON for this endpoint: "+err.Error())
		return false
	}
	return true
}
