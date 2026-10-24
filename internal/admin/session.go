// Package admin serves the JSON API behind the web interface: signing in, apps, access keys and the log.
package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Silify/letspush/internal/secret"
)

const (
	cookieName  = "letspush_session"
	sessionTime = 12 * time.Hour
)

// sessions issue and check the signed cookie that keeps an admin signed in.
type sessions struct {
	key []byte
	now func() time.Time
}

func newSessions(serverSecret string, now func() time.Time) *sessions {
	mac := hmac.New(sha256.New, []byte(serverSecret))
	mac.Write([]byte("letspush-session-v1"))
	return &sessions{key: mac.Sum(nil), now: now}
}

func (s *sessions) sign(payload string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// issue sets the cookie for an admin.
func (s *sessions) issue(w http.ResponseWriter, r *http.Request, adminID int64) {
	expires := s.now().Add(sessionTime)
	payload := fmt.Sprintf("%d:%d", adminID, expires.Unix())
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + s.sign(payload)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   isSecure(r),
		SameSite: http.SameSiteStrictMode,
	})
}

// clear removes the cookie.
func (s *sessions) clear(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: isSecure(r), SameSite: http.SameSiteStrictMode,
	})
}

// adminID returns the admin the request's cookie belongs to.
func (s *sessions) adminID(r *http.Request) (int64, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return 0, false
	}
	encoded, signature, found := strings.Cut(cookie.Value, ".")
	if !found {
		return 0, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, false
	}
	payload := string(raw)
	if !secret.Equal(s.sign(payload), signature) {
		return 0, false
	}
	idText, expiryText, found := strings.Cut(payload, ":")
	if !found {
		return 0, false
	}
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil {
		return 0, false
	}
	expiry, err := strconv.ParseInt(expiryText, 10, 64)
	if err != nil || s.now().Unix() >= expiry {
		return 0, false
	}
	return id, true
}

// isSecure is true when the browser reached us over HTTPS, directly or through a proxy.
func isSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
