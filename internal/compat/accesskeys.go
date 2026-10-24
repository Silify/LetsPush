package compat

import (
	"net/http"
	"strings"

	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
)

// requestAccessKey is Moodle's "Request access key" button. It only works for an app whose admin
// has allowed it; otherwise keys are made in the web interface.
func (h *Handler) requestAccessKey(w http.ResponseWriter, r *http.Request) {
	sess, ok := h.authenticate(w, r, false)
	if !ok {
		return
	}
	if !sess.app.AllowKeyRequests {
		writeError(w, http.StatusForbidden, "This server does not give out access keys on request")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	siteURL := strings.TrimSpace(r.FormValue("url"))
	description := strings.TrimSpace(r.FormValue("description"))
	if description == "" {
		description = siteURL
	}

	key, prefix, hash, err := secret.NewAccessKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not make a key")
		return
	}
	if _, err := h.store.CreateKey(store.NewKey{
		AppID:       sess.app.ID,
		Hash:        hash,
		Prefix:      prefix,
		Description: truncate(description, 200),
		Contact:     truncate(siteURL, 200),
		Permissions: store.PermAll,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save the key")
		return
	}
	_ = h.store.AddLog(sess.app.ID, "accesskey", "requested by "+truncate(siteURL, 100))
	writeJSON(w, http.StatusOK, map[string]string{"accesskey": key})
}

func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}
