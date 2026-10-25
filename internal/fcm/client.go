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
	mrand "math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultTokenURL = "https://oauth2.googleapis.com/token"
	defaultSendBase = "https://fcm.googleapis.com"
	scope           = "https://www.googleapis.com/auth/firebase.messaging"
	refreshMargin   = time.Minute

	defaultAttempts = 3
	retryBase       = 300 * time.Millisecond
	retryCap        = 10 * time.Second
	reasonLimit     = 300
)

// ErrUnregistered means Firebase no longer knows the device token.
var ErrUnregistered = errors.New("unregistered token")

// Error is a refusal from Google.
type Error struct {
	Status int
	Stage  string // "sign-in" or "send"
	// Reason is Google's own explanation, shortened, such as "PERMISSION_DENIED: Cloud Messaging API has not
	// been used in project…". It is for the log; it never goes back to the site that sent the push.
	Reason string

	retryAfter time.Duration
}

func (e *Error) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("firebase %s failed with status %d", e.Stage, e.Status)
	}
	return fmt.Sprintf("firebase %s failed with status %d: %s", e.Stage, e.Status, e.Reason)
}

// Message is one notification for one device.
type Message struct {
	Token string
	Title string
	Body  string
	Image string
	Data  map[string]string

	Priority string        // "normal" or "high"; empty means high
	TTL      time.Duration // zero leaves it to Firebase
	Sound    string
}

// Client sends messages. It keeps one access token per Firebase key and renews it before it expires.
type Client struct {
	HTTP     *http.Client
	TokenURL string
	SendBase string
	Now      func() time.Time
	// Attempts is how many times a send is tried when Google is busy or unreachable. Default 3.
	Attempts int
	// Sleep waits between attempts. It returns early with an error when ctx ends. Default: a timer.
	Sleep func(ctx context.Context, d time.Duration) error

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
		Attempts: defaultAttempts,
		Sleep:    sleep,
		cache:    map[string]*cachedToken{},
		pending:  map[string]*pendingToken{},
	}
}

// Send delivers a message. It returns ErrUnregistered for a dead device, and *Error for other refusals.
// A busy Google (429, 5xx), a network failure or an expired access token is retried with a growing pause.
func (c *Client) Send(ctx context.Context, creds Credentials, msg Message) error {
	payload, err := json.Marshal(map[string]any{"message": buildMessage(msg, c.Now())})
	if err != nil {
		return err
	}
	for attempt := 1; ; attempt++ {
		err = c.sendOnce(ctx, creds, payload)
		if err == nil || attempt >= c.Attempts || !retryable(ctx, err) {
			return err
		}
		if c.Sleep(ctx, backoff(attempt, err)) != nil {
			return err
		}
	}
}

func (c *Client) sendOnce(ctx context.Context, creds Credentials, payload []byte) error {
	token, err := c.accessToken(ctx, creds)
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
	return &Error{Status: resp.StatusCode, Stage: "send", Reason: reasonOf(body), retryAfter: retryAfter(resp)}
}

// retryable says whether another attempt could succeed: Google is busy or failing, the network dropped
// the request, or the access token was refused (the next attempt asks for a fresh one).
func retryable(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		s := apiErr.Status
		return s == http.StatusTooManyRequests || s >= 500 || (s == http.StatusUnauthorized && apiErr.Stage == "send")
	}
	var netErr *url.Error
	return errors.As(err, &netErr)
}

// backoff is the pause before attempt+1: doubling from 300 ms with up to half again as jitter, or what
// Google asked for in Retry-After if that is longer.
func backoff(attempt int, err error) time.Duration {
	d := retryBase << (attempt - 1)
	d = d/2 + mrand.N(d)
	var apiErr *Error
	if errors.As(err, &apiErr) && apiErr.retryAfter > d {
		d = apiErr.retryAfter
	}
	return min(d, retryCap)
}

func retryAfter(resp *http.Response) time.Duration {
	seconds, err := strconv.Atoi(resp.Header.Get("Retry-After"))
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// reasonOf pulls Google's explanation out of an error reply, for the log. Google's replies carry no
// secrets, only the status, a message and sometimes an error code.
func reasonOf(body []byte) string {
	var reply struct {
		// FCM: {"error": {"status", "message", "details": [{"errorCode"}]}}
		// Sign-in: {"error": "invalid_grant", "error_description": "..."}
		Error json.RawMessage `json:"error"`
		Desc  string          `json:"error_description"`
	}
	if json.Unmarshal(body, &reply) != nil || len(reply.Error) == 0 {
		return ""
	}

	var code, message string
	var plain string
	if json.Unmarshal(reply.Error, &plain) == nil {
		code, message = plain, reply.Desc
	} else {
		var object struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		}
		if json.Unmarshal(reply.Error, &object) != nil {
			return ""
		}
		code, message = object.Status, object.Message
		for _, d := range object.Details {
			if d.ErrorCode != "" && d.ErrorCode != code {
				code += "/" + d.ErrorCode
				break
			}
		}
	}

	reason := strings.TrimSpace(code + ": " + strings.Join(strings.Fields(message), " "))
	reason = strings.Trim(reason, ": ")
	runes := []rune(reason)
	if len(runes) > reasonLimit {
		reason = string(runes[:reasonLimit]) + "…"
	}
	return reason
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
// With no title and no body the message is data only (an image needs a title or a body), and the app shows it. High priority is the
// default, because a notification that arrives late is of little use.
//
// Delivery options are set for each platform's own fields: Android (priority, ttl, sound), Apple
// (priority, expiration, sound) and the web (urgency, TTL). The web ones are only sent when the caller
// chose a priority or a TTL.
func buildMessage(m Message, now time.Time) map[string]any {
	high := m.Priority != "normal"

	android := map[string]any{"priority": "normal"}
	apnsHeaders := map[string]string{"apns-priority": "5"}
	if high {
		android["priority"] = "high"
		apnsHeaders["apns-priority"] = "10"
	}
	if m.TTL > 0 {
		android["ttl"] = fmt.Sprintf("%ds", int64(m.TTL.Seconds()))
		apnsHeaders["apns-expiration"] = strconv.FormatInt(now.Add(m.TTL).Unix(), 10)
	}
	apns := map[string]any{"headers": apnsHeaders}
	if m.Sound != "" {
		android["notification"] = map[string]string{"sound": m.Sound}
		apns["payload"] = map[string]any{"aps": map[string]string{"sound": m.Sound}}
	}

	out := map[string]any{"token": m.Token, "android": android, "apns": apns}
	if m.Priority != "" || m.TTL > 0 {
		webHeaders := map[string]string{"Urgency": "normal"}
		if high {
			webHeaders["Urgency"] = "high"
		}
		if m.TTL > 0 {
			webHeaders["TTL"] = strconv.FormatInt(int64(m.TTL.Seconds()), 10)
		}
		out["webpush"] = map[string]any{"headers": webHeaders}
	}
	if m.Title != "" || m.Body != "" {
		notification := map[string]string{"title": m.Title, "body": m.Body}
		if m.Image != "" {
			notification["image"] = m.Image
		}
		out["notification"] = notification
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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return "", time.Time{}, &Error{Status: resp.StatusCode, Stage: "sign-in", Reason: reasonOf(body), retryAfter: retryAfter(resp)}
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
