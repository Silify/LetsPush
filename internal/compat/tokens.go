package compat

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Silify/letspush/internal/store"
)

const (
	maxTokenLength = 4096
	defaultDevice  = "fcm"
	defaultChannel = "default"

	// Moodle's settings check registers this token to see whether its access key works.
	probeToken = "testtoken"
)

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
	if !sess.key.Can(store.PermCreateToken) {
		writeError(w, http.StatusForbidden, "No permission to create token")
		return
	}
	token = strings.TrimSpace(token)
	if token == "" || len(token) > maxTokenLength {
		writeError(w, http.StatusBadRequest, "Invalid token")
		return
	}
	// A key that works answers like any other, without storing the probe.
	if token == probeToken {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	created, err := h.store.UpsertToken(store.Token{
		AppID:   sess.app.ID,
		KeyID:   sess.key.ID,
		Token:   token,
		Device:  orDefault(strings.ToLower(strings.TrimSpace(device)), defaultDevice),
		Channel: orDefault(strings.TrimSpace(channel), defaultChannel),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save the token")
		return
	}
	if created {
		_ = h.store.AddLog(sess.app.ID, "token", "added by key "+sess.key.Prefix)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "token exists"})
}

// deleteToken is DELETE /tokens/{token}.
func (h *Handler) deleteToken(w http.ResponseWriter, r *http.Request) {
	sess, ok := h.authenticate(w, r, true)
	if !ok {
		return
	}
	if !sess.key.Can(store.PermDeleteToken) {
		writeError(w, http.StatusForbidden, "No permission to delete token")
		return
	}
	if err := h.store.DeleteToken(sess.key.ID, r.PathValue("token")); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"status": "Token doesn't exist"})
		return
	}
	_ = h.store.AddLog(sess.app.ID, "token", "removed by key "+sess.key.Prefix)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
