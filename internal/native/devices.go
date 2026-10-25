package native

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/Silify/letspush/internal/gateway"
)

// platform is what a device reports about itself: android, ios, web, or a name of the sender's own.
var platform = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// registerDevice is POST /v1/devices: {"token": "…", "platform": "android", "channel": "…"}.
// It answers 201 for a new device and 200 for one the key already registered.
func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request, sess session) {
	var body struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
		Channel  string `json:"channel"`
	}
	if !decode(w, r, &body) {
		return
	}
	body.Platform = strings.ToLower(strings.TrimSpace(body.Platform))
	if !platform.MatchString(body.Platform) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "platform is required: lowercase letters, digits, - or _, up to 32 characters, such as android, ios or web")
		return
	}

	created, err := h.gateway.Register(sess.app, sess.key, body.Token, body.Platform, body.Channel)
	switch {
	case err == nil && created:
		writeJSON(w, http.StatusCreated, map[string]any{"platform": body.Platform, "created": true})
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]any{"platform": body.Platform, "created": false})
	default:
		h.writeGatewayError(w, r, err)
	}
}

// removeDevice is DELETE /v1/devices/{token}. It answers 204.
func (h *Handler) removeDevice(w http.ResponseWriter, r *http.Request, sess session) {
	err := h.gateway.Remove(sess.app, sess.key, r.PathValue("token"))
	if err != nil {
		h.writeGatewayError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeGatewayError maps what the gateway returns to this API's codes.
func (h *Handler) writeGatewayError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, gateway.ErrNoPermission):
		writeError(w, r, http.StatusForbidden, "forbidden", "this access key may not do that")
	case errors.Is(err, gateway.ErrInvalidToken):
		writeError(w, r, http.StatusBadRequest, "invalid_request", "the device token is empty or too long")
	case errors.Is(err, gateway.ErrUnknownToken):
		writeError(w, r, http.StatusNotFound, "unknown_device", "this access key has not registered that device; register it first, or give the key permission to create devices")
	case errors.Is(err, gateway.ErrTokenNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "this access key has no such device")
	case errors.Is(err, gateway.ErrUnregistered):
		writeError(w, r, http.StatusNotFound, "device_unregistered", "the push service no longer knows this device; it was removed")
	case errors.Is(err, gateway.ErrNoProvider):
		writeError(w, r, http.StatusServiceUnavailable, "no_provider", "the app has no working key for this kind of device")
	case errors.Is(err, gateway.ErrDelivery):
		writeError(w, r, http.StatusBadGateway, "delivery_failed", "the push service refused the notification or could not be reached; the app's activity log has the reason")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
