package compat

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/Silify/letspush/internal/gateway"
)

// push is POST /api/v2/push/: send one notification to one device.
func (h *Handler) push(w http.ResponseWriter, r *http.Request) {
	sess, ok := h.authenticate(w, r, true)
	if !ok {
		return
	}

	var req pushRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxBody))
	decoder.UseNumber()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	title, body, data := notificationFrom(req)
	_, err := h.gateway.Send(r.Context(), sess.app, sess.key, gateway.Message{
		Token:        req.Token,
		Platform:     req.Device,
		Channel:      req.Channel,
		Notification: gateway.Notification{Title: title, Body: body, Data: data},
	})
	switch {
	case err == nil:
		// Sent, or still on its way: Moodle only needs to know that it was taken.
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
	case errors.Is(err, gateway.ErrNoPermission):
		writeError(w, http.StatusForbidden, "No permission to send notification")
	case errors.Is(err, gateway.ErrInvalidToken):
		writeError(w, http.StatusBadRequest, "Invalid token")
	case errors.Is(err, gateway.ErrUnknownToken):
		writeError(w, http.StatusBadRequest, "Unknown token and you have no permission to create")
	case errors.Is(err, gateway.ErrNoProvider):
		writeError(w, http.StatusServiceUnavailable, "This app has no working Firebase key")
	case errors.Is(err, gateway.ErrUnregistered):
		writeError(w, http.StatusNotFound, "Unregistered token")
	case errors.Is(err, gateway.ErrDelivery):
		writeError(w, http.StatusBadGateway, "error response from fcm")
	default:
		writeError(w, http.StatusInternalServerError, "something went wrong")
	}
}
