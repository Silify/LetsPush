package native

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/Silify/letspush/internal/gateway"
	"github.com/Silify/letspush/internal/store"
)

const (
	// maxPayload is Firebase's limit on a message, with a little room for what LetsPush adds.
	maxPayload = 4000
	maxImage   = 2048
	maxSound   = 100
	maxTTL     = 28 * 24 * time.Hour // the longest Firebase keeps a message
)

// sendRequest is the body of POST /v1/notifications.
type sendRequest struct {
	To struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	} `json:"to"`
	Title   string            `json:"title"`
	Body    string            `json:"body"`
	Image   string            `json:"image"`
	Data    map[string]string `json:"data"`
	Options struct {
		Priority string `json:"priority"`
		TTL      int64  `json:"ttl"`
		Sound    string `json:"sound"`
	} `json:"options"`
}

// sendNotification is POST /v1/notifications. It answers 202 with the delivery's ID and how far it got:
// "sent" when the push service accepted it, "pending" when it had not answered yet.
func (h *Handler) sendNotification(w http.ResponseWriter, r *http.Request, sess session) {
	var req sendRequest
	if !decode(w, r, &req) {
		return
	}
	notification, problem := req.notification()
	if problem != "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", problem)
		return
	}
	req.To.Platform = strings.ToLower(strings.TrimSpace(req.To.Platform))
	if req.To.Platform != "" && !platform.MatchString(req.To.Platform) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "to.platform may use lowercase letters, digits, - or _, up to 32 characters")
		return
	}

	receipt, err := h.gateway.SendTracked(r.Context(), sess.app, sess.key, gateway.Message{
		Token:        req.To.Token,
		Platform:     req.To.Platform,
		Notification: notification,
	})
	if err != nil {
		h.writeGatewayError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": receipt.ID, "status": string(receipt.Status)})
}

// notification checks the request and turns it into the gateway's notification. The second result is
// what is wrong with the request, or "".
func (req sendRequest) notification() (gateway.Notification, string) {
	n := gateway.Notification{
		Title: req.Title,
		Body:  req.Body,
		Image: req.Image,
		Data:  req.Data,
		Sound: req.Options.Sound,
	}
	if req.To.Token == "" {
		return n, "to.token is required"
	}
	if req.Title == "" && req.Body == "" && len(req.Data) == 0 {
		return n, "send a title, a body or data"
	}

	if req.Image != "" {
		if req.Title == "" && req.Body == "" {
			return n, "an image needs a title or a body to go with it"
		}
		if u, err := url.Parse(req.Image); err != nil || u.Scheme != "https" || u.Host == "" || len(req.Image) > maxImage {
			return n, "image must be an https address of at most 2048 characters"
		}
	}

	size := len(req.Title) + len(req.Body) + len(req.Image)
	for key, value := range req.Data {
		if key == "" {
			return n, "data keys must not be empty"
		}
		if reservedDataKey(key) {
			return n, `data key "` + key + `" is reserved by Firebase (not "from", "notification", "message_type", or starting with "google" or "gcm")`
		}
		size += len(key) + len(value)
	}
	if size > maxPayload {
		return n, "the notification is too large: title, body, image and data together may be at most 4000 bytes"
	}

	switch req.Options.Priority {
	case "":
	case gateway.PriorityNormal, gateway.PriorityHigh:
		n.Priority = req.Options.Priority
	default:
		return n, `options.priority must be "normal" or "high"`
	}
	n.TTL = time.Duration(req.Options.TTL) * time.Second
	if req.Options.TTL < 0 || n.TTL > maxTTL {
		return n, "options.ttl is a number of seconds from 1 to 2419200 (28 days)"
	}
	if len(req.Options.Sound) > maxSound || strings.IndexFunc(req.Options.Sound, unicode.IsControl) >= 0 {
		return n, "options.sound is a name of at most 100 characters"
	}
	return n, ""
}

// reservedDataKey says whether Firebase keeps this key for itself.
func reservedDataKey(key string) bool {
	switch key {
	case "from", "notification", "message_type":
		return true
	}
	lower := strings.ToLower(key)
	return strings.HasPrefix(lower, "google") || strings.HasPrefix(lower, "gcm")
}

// showNotification is GET /v1/notifications/{id}: how a delivery that this key started ended.
func (h *Handler) showNotification(w http.ResponseWriter, r *http.Request, sess session) {
	d, err := h.store.DeliveryByID(sess.key.ID, r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "not_found", "no such notification for this access key (they are kept for the log retention period)")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal", "something went wrong")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        d.ID,
		"status":    d.Status,
		"requestId": d.RequestID,
		"createdAt": d.CreatedAt.Format(time.RFC3339),
		"updatedAt": d.UpdatedAt.Format(time.RFC3339),
	})
}
