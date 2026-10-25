package compat

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Silify/letspush/internal/gateway"
	"github.com/Silify/letspush/internal/store"
)

// Moodle's settings check registers this token to see whether its access key works.
const probeToken = "testtoken"

// createToken is POST /tokens/{token}, with the device type in the form.
func (h *Handler) createToken(w http.ResponseWriter, r *http.Request) {
	sess, ok := h.authenticate(w, r, true)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	h.register(w, sess, r.PathValue("token"), r.FormValue("device"), r.FormValue("channel"))
}

// createTokenV2 is POST /api/v2/tokens, with the token in a JSON body.
func (h *Handler) createTokenV2(w http.ResponseWriter, r *http.Request) {
	sess, ok := h.authenticate(w, r, true)
	if !ok {
		return
	}
	var body struct {
		Token   string `json:"token"`
		Device  string `json:"device"`
		Channel string `json:"channel"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxBody)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	h.register(w, sess, body.Token, body.Device, body.Channel)
}

func (h *Handler) register(w http.ResponseWriter, sess session, token, device, channel string) {
	// A key that works answers like any other, without storing the probe.
	if strings.TrimSpace(token) == probeToken && sess.key.Can(store.PermCreateToken) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	created, err := h.gateway.Register(sess.app, sess.key, token, device, channel)
	switch {
	case errors.Is(err, gateway.ErrNoPermission):
		writeError(w, http.StatusForbidden, "No permission to create token")
	case errors.Is(err, gateway.ErrInvalidToken):
		writeError(w, http.StatusBadRequest, "Invalid token")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "could not save the token")
	case created:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "token exists"})
	}
}

// deleteToken is DELETE /tokens/{token}.
func (h *Handler) deleteToken(w http.ResponseWriter, r *http.Request) {
	sess, ok := h.authenticate(w, r, true)
	if !ok {
		return
	}
	switch err := h.gateway.Remove(sess.app, sess.key, r.PathValue("token")); {
	case errors.Is(err, gateway.ErrNoPermission):
		writeError(w, http.StatusForbidden, "No permission to delete token")
	case errors.Is(err, gateway.ErrTokenNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"status": "Token doesn't exist"})
	case err != nil:
		writeError(w, http.StatusInternalServerError, "something went wrong")
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
