package fcm

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	defaultTokenURL = "https://oauth2.googleapis.com/token"
	defaultSendBase = "https://fcm.googleapis.com"
	scope           = "https://www.googleapis.com/auth/firebase.messaging"
	refreshMargin   = time.Minute
)

// ErrUnregistered means Firebase no longer knows the device token.
var ErrUnregistered = errors.New("unregistered token")

// Error is a refusal from Google. It carries no Google text, only the status.
type Error struct {
	Status int
	Stage  string // "sign-in" or "send"
}

func (e *Error) Error() string {
	return fmt.Sprintf("firebase %s failed with status %d", e.Stage, e.Status)
}

// Message is one notification for one device.
type Message struct {
	Token string
	Title string
	Body  string
	Data  map[string]string
}

// Client sends messages. It keeps one access token per Firebase key and renews it before it expires.
type Client struct {
	HTTP     *http.Client
	TokenURL string
	SendBase string
	Now      func() time.Time

	mu      sync.Mutex
	cache   map[string]*cachedToken
	pending map[string]*pendingToken
}

type cachedToken struct {
	value     string
	expiresAt time.Time
}

type pendingToken struct {
	done  chan struct{}
	value string
	err   error
}

// NewClient returns a client for Google's real endpoints.
func NewClient() *Client {
	return &Client{
		HTTP:     &http.Client{Timeout: 15 * time.Second},
		TokenURL: defaultTokenURL,
		SendBase: defaultSendBase,
		Now:      time.Now,
		cache:    map[string]*cachedToken{},
		pending:  map[string]*pendingToken{},
	}
}

// Send delivers a message. It returns ErrUnregistered for a dead device, and *Error for other refusals.
func (c *Client) Send(ctx context.Context, creds Credentials, msg Message) error {
	token, err := c.accessToken(ctx, creds)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(map[string]any{"message": buildMessage(msg)})
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/v1/projects/%s/messages:send", c.SendBase, url.PathEscape(creds.ProjectID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		c.forget(creds)
	}
	if isUnregistered(resp.StatusCode, body) {
		return ErrUnregistered
	}
	return &Error{Status: resp.StatusCode, Stage: "send"}
}

func isUnregistered(status int, body []byte) bool {
	if status != http.StatusNotFound && status != http.StatusBadRequest {
		return false
	}
	var reply struct {
		Error struct {
			Status  string `json:"status"`
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &reply) != nil {
		return false
	}
	for _, d := range reply.Error.Details {
		if d.ErrorCode == "UNREGISTERED" {
			return true
		}
	}
	return status == http.StatusNotFound && reply.Error.Status == "NOT_FOUND"
}

// buildMessage is the FCM v1 message: a notification Android shows by itself, plus the data for the app.
func buildMessage(m Message) map[string]any {
	out := map[string]any{
		"token":        m.Token,
		"notification": map[string]string{"title": m.Title, "body": m.Body},
		"android":      map[string]any{"priority": "high"},
		"apns":         map[string]any{"headers": map[string]string{"apns-priority": "10"}},
	}
	if len(m.Data) > 0 {
		out["data"] = m.Data
	}
	return out
}

func (c *Client) cacheKey(creds Credentials) string { return creds.ClientEmail + "|" + creds.ProjectID }

func (c *Client) forget(creds Credentials) {
	c.mu.Lock()
	delete(c.cache, c.cacheKey(creds))
	c.mu.Unlock()
}

// accessToken returns a valid token, asking Google once even when many sends start together.
func (c *Client) accessToken(ctx context.Context, creds Credentials) (string, error) {
	key := c.cacheKey(creds)

	c.mu.Lock()
	if t := c.cache[key]; t != nil && t.expiresAt.Sub(c.Now()) > refreshMargin {
		c.mu.Unlock()
		return t.value, nil
	}
	if p := c.pending[key]; p != nil {
		c.mu.Unlock()
		select {
		case <-p.done:
			return p.value, p.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	p := &pendingToken{done: make(chan struct{})}
	c.pending[key] = p
	c.mu.Unlock()

	value, expires, err := c.requestToken(ctx, creds)

	c.mu.Lock()
	if err == nil {
		c.cache[key] = &cachedToken{value: value, expiresAt: expires}
	}
	delete(c.pending, key)
	c.mu.Unlock()

	p.value, p.err = value, err
	close(p.done)
	return value, err
}

func (c *Client) requestToken(ctx context.Context, creds Credentials) (string, time.Time, error) {
	assertion, err := c.signAssertion(creds)
	if err != nil {
		return "", time.Time{}, err
	}
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, &Error{Status: resp.StatusCode, Stage: "sign-in"}
	}

	var reply struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&reply); err != nil || reply.AccessToken == "" {
		return "", time.Time{}, &Error{Status: resp.StatusCode, Stage: "sign-in"}
	}
	return reply.AccessToken, c.Now().Add(time.Duration(reply.ExpiresIn) * time.Second), nil
}

// signAssertion builds the signed request Google exchanges for an access token.
func (c *Client) signAssertion(creds Credentials) (string, error) {
	issued := c.Now().Unix()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	claims, _ := json.Marshal(map[string]any{
		"iss":   creds.ClientEmail,
		"scope": scope,
		"aud":   defaultTokenURL,
		"iat":   issued,
		"exp":   issued + 3600,
	})
	enc := base64.RawURLEncoding
	signingInput := enc.EncodeToString(header) + "." + enc.EncodeToString(claims)

	sum := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, creds.privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + enc.EncodeToString(signature), nil
}
