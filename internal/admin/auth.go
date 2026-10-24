package admin

import (
	"net"
	"net/http"
	"strings"

	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
)

const minPasswordLength = 10

func hashForTiming() (string, error) { return secret.HashPassword("not-a-real-password") }

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if !readJSON(w, r, &body) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	attempt := clientAddress(r) + "|" + email

	if !h.throttle.allowed(attempt) {
		writeError(w, http.StatusTooManyRequests, "too many attempts; try again in a few minutes")
		return
	}

	admin, err := h.store.AdminByEmail(email)
	hash := h.dummyHash
	if err == nil {
		hash = admin.PasswordHash
	}
	// Always check a hash, so a wrong email takes as long as a wrong password.
	valid := secret.CheckPassword(body.Password, hash)
	if err != nil || !valid {
		h.throttle.fail(attempt)
		writeError(w, http.StatusUnauthorized, "wrong email or password")
		return
	}

	h.throttle.succeed(attempt)
	h.sessions.issue(w, r, admin.ID)
	writeJSON(w, http.StatusOK, map[string]string{"email": admin.Email})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.sessions.clear(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) me(w http.ResponseWriter, _ *http.Request, admin store.Admin) {
	writeJSON(w, http.StatusOK, map[string]string{"email": admin.Email})
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request, admin store.Admin) {
	var body struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if !secret.CheckPassword(body.Current, admin.PasswordHash) {
		writeError(w, http.StatusForbidden, "the current password is wrong")
		return
	}
	if len(body.New) < minPasswordLength {
		writeError(w, http.StatusBadRequest, "the new password must be at least 10 characters")
		return
	}
	hash, err := secret.HashPassword(body.New)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if err := h.store.SetAdminPassword(admin.ID, hash); err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// clientAddress is the address the request came from. Behind a proxy that is the proxy's address,
// which still limits guessing; trusting forwarded headers here would let an attacker avoid the limit.
func clientAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
