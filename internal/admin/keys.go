package admin

import (
	"net/http"
	"strings"
	"time"

	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
)

type keyView struct {
	ID          int64      `json:"id"`
	Prefix      string     `json:"prefix"`
	Description string     `json:"description"`
	Contact     string     `json:"contact"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt"`
	RevokedAt   *time.Time `json:"revokedAt"`
	Tokens      int        `json:"tokens"`
}

// permissionNames maps the stored bits to the names the interface uses.
var permissionNames = []struct {
	name string
	bit  int
}{
	{"createToken", store.PermCreateToken},
	{"deleteToken", store.PermDeleteToken},
	{"sendNotification", store.PermSendNotification},
}

func keyViewOf(k store.AccessKey) keyView {
	names := []string{}
	for _, p := range permissionNames {
		if k.Can(p.bit) {
			names = append(names, p.name)
		}
	}
	return keyView{k.ID, k.Prefix, k.Description, k.Contact, names, k.CreatedAt, k.LastUsedAt, k.RevokedAt, k.Tokens}
}

func keyViews(keys []store.AccessKey) []keyView {
	views := make([]keyView, 0, len(keys))
	for _, k := range keys {
		views = append(views, keyViewOf(k))
	}
	return views
}

func (h *Handler) createKey(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	appID, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Description string   `json:"description"`
		Contact     string   `json:"contact"`
		Permissions []string `json:"permissions"`
	}
	if !readJSON(w, r, &body) {
		return
	}

	permissions := store.PermAll
	if body.Permissions != nil {
		permissions = 0
		for _, name := range body.Permissions {
			bit := 0
			for _, p := range permissionNames {
				if p.name == name {
					bit = p.bit
				}
			}
			if bit == 0 {
				writeError(w, http.StatusBadRequest, "unknown permission: "+name)
				return
			}
			permissions |= bit
		}
		if permissions == 0 {
			writeError(w, http.StatusBadRequest, "a key needs at least one permission")
			return
		}
	}

	if _, err := h.store.AppByID(appID); err != nil {
		storeError(w, err)
		return
	}
	key, prefix, hash, err := secret.NewAccessKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	created, err := h.store.CreateKey(store.NewKey{
		AppID:       appID,
		Hash:        hash,
		Prefix:      prefix,
		Description: limit(strings.TrimSpace(body.Description), 200),
		Contact:     limit(strings.TrimSpace(body.Contact), 200),
		Permissions: permissions,
	})
	if err != nil {
		storeError(w, err)
		return
	}
	_ = h.store.AddLog(appID, "accesskey", "created "+prefix)

	// The key is shown now and never again: only its hash is kept.
	writeJSON(w, http.StatusCreated, struct {
		keyView
		Key string `json:"key"`
	}{keyViewOf(created), key})
}

func (h *Handler) revokeKey(w http.ResponseWriter, r *http.Request, _ store.Admin) {
	appID, ok := pathID(r, "id")
	keyID, ok2 := pathID(r, "keyId")
	if !ok || !ok2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err := h.store.RevokeKey(appID, keyID); err != nil {
		storeError(w, err)
		return
	}
	_ = h.store.AddLog(appID, "accesskey", "revoked")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func limit(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}
