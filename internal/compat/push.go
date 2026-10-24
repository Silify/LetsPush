package compat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Silify/letspush/internal/fcm"
	"github.com/Silify/letspush/internal/store"
)

// push is POST /api/v2/push/: send one notification to one device.
func (h *Handler) push(w http.ResponseWriter, r *http.Request) {
	sess, ok := h.authenticate(w, r, true)
	if !ok {
		return
	}
	if !sess.key.Can(store.PermSendNotification) {
		writeError(w, http.StatusForbidden, "No permission to send notification")
		return
	}

	var req pushRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxBody))
	decoder.UseNumber()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" || len(token) > maxTokenLength {
		writeError(w, http.StatusBadRequest, "Invalid token")
		return
	}

	// A device this key has not registered yet is registered now, if the key may.
	if _, err := h.store.FindToken(sess.key.ID, token); errors.Is(err, store.ErrNotFound) {
		if !sess.key.Can(store.PermCreateToken) {
			writeError(w, http.StatusBadRequest, "Unknown token and you have no permission to create")
			return
		}
		if _, err := h.store.UpsertToken(store.Token{
			AppID:   sess.app.ID,
			KeyID:   sess.key.ID,
			Token:   token,
			Device:  orDefault(strings.ToLower(strings.TrimSpace(req.Device)), defaultDevice),
			Channel: orDefault(strings.TrimSpace(req.Channel), defaultChannel),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save the token")
			return
		}
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not look up the token")
		return
	}

	creds, err := h.credentials(sess.app)
	if err != nil {
		_ = h.store.AddLog(sess.app.ID, "error", err.Error())
		writeError(w, http.StatusServiceUnavailable, "This app has no working Firebase key")
		return
	}

	title, body, data := notificationFrom(req)
	err = h.sender.Send(r.Context(), creds, fcm.Message{Token: token, Title: title, Body: body, Data: data})
	switch {
	case err == nil:
		_ = h.store.AddLog(sess.app.ID, "push", fmt.Sprintf("sent to %s by key %s", shortToken(token), sess.key.Prefix))
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
	case errors.Is(err, fcm.ErrUnregistered):
		_ = h.store.DeleteToken(sess.key.ID, token)
		_ = h.store.AddLog(sess.app.ID, "push", fmt.Sprintf("%s is no longer registered; removed", shortToken(token)))
		writeError(w, http.StatusNotFound, "Unregistered token")
	default:
		_ = h.store.AddLog(sess.app.ID, "error", fmt.Sprintf("sending to %s failed: %s", shortToken(token), err))
		writeError(w, http.StatusBadGateway, "error response from fcm")
	}
}

// credentials opens the app's sealed Firebase key.
func (h *Handler) credentials(app store.App) (fcm.Credentials, error) {
	if !app.HasCredentials {
		return fcm.Credentials{}, errors.New("the app has no Firebase key")
	}
	sealed, err := h.store.AppCredentials(app.ID)
	if err != nil {
		return fcm.Credentials{}, err
	}
	raw, err := h.opener.Open(sealed)
	if err != nil {
		return fcm.Credentials{}, err
	}
	return fcm.ParseCredentials(bytes.TrimSpace(raw))
}

// shortToken is enough of a token to recognise it in the log without keeping the whole device address:
// the first characters, never more than half of it.
func shortToken(token string) string {
	runes := []rune(token)
	keep := min(8, len(runes)/2)
	return string(runes[:keep]) + "…"
}
