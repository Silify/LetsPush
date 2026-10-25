// Package gateway is the heart of LetsPush: it registers devices and delivers notifications to them.
// It knows nothing about HTTP or about any push service; the APIs in front of it and the providers
// behind it are plugged in from outside.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Silify/letspush/internal/store"
)

var (
	// ErrNoPermission means the access key may not do this.
	ErrNoPermission = errors.New("no permission")
	// ErrInvalidToken means the device token is empty or too long.
	ErrInvalidToken = errors.New("invalid token")
	// ErrUnknownToken means the key has not registered this device and may not register it now.
	ErrUnknownToken = errors.New("unknown token and no permission to create")
	// ErrTokenNotFound means the key has no such device.
	ErrTokenNotFound = errors.New("token not found")
	// ErrNoProvider means the app has no working key for the push service this device needs.
	ErrNoProvider = errors.New("the app has no working key for this device")
	// ErrUnregistered is what a provider returns when the push service no longer knows the device.
	ErrUnregistered = errors.New("unregistered token")
	// ErrDelivery means the push service refused the notification or could not be reached.
	ErrDelivery = errors.New("delivery failed")
)

const (
	maxTokenLength = 4096
	defaultDevice  = "fcm"
	defaultChannel = "default"

	defaultWait    = 1500 * time.Millisecond
	defaultTimeout = 30 * time.Second
)

// Notification is what the person sees, plus the data the app receives.
// With no title and no body it is data only, and the app shows it.
type Notification struct {
	Title string
	Body  string
	Data  map[string]string
}

// Device is where a notification goes.
type Device struct {
	Token string
	// Platform is the device type the site gave: "android", "ios", "web"...
	Platform string
}

// Provider delivers a notification through one push service.
// It returns ErrUnregistered when the service no longer knows the device.
type Provider interface {
	Send(ctx context.Context, credentials []byte, device Device, n Notification) error
}

// Store is the part of the database the gateway uses.
type Store interface {
	AppProvider(appID int64, kind string) (store.Provider, error)
	FindToken(keyID int64, token string) (store.Token, error)
	UpsertToken(store.Token) (created bool, err error)
	DeleteToken(keyID int64, token string) error
	AddLog(appID int64, action, info string) error
}

// Opener opens the sealed provider keys.
type Opener interface {
	Open(sealed string) ([]byte, error)
}

// Config is what a Service is built from.
type Config struct {
	Store  Store
	Opener Opener
	// Providers are the push services, by kind.
	Providers map[string]Provider
	// Wait is how long Send waits for the provider before it answers Pending. Default 1.5 seconds.
	Wait time.Duration
	// Timeout is how long a delivery may take in all. Default 30 seconds.
	Timeout time.Duration
}

// Service registers devices and delivers notifications.
type Service struct {
	store     Store
	opener    Opener
	providers map[string]Provider
	wait      time.Duration
	timeout   time.Duration
	running   sync.WaitGroup
}

// New returns a Service.
func New(c Config) *Service {
	s := &Service{store: c.Store, opener: c.Opener, providers: c.Providers, wait: c.Wait, timeout: c.Timeout}
	if s.wait <= 0 {
		s.wait = defaultWait
	}
	if s.timeout <= 0 {
		s.timeout = defaultTimeout
	}
	return s
}

// Wait blocks until every delivery that is still running has finished. Call it when shutting down.
func (s *Service) Wait() { s.running.Wait() }

// Register adds a device for an access key, or updates it. It reports whether the device was new.
func (s *Service) Register(app store.App, key store.AccessKey, token, platform, channel string) (created bool, err error) {
	if !key.Can(store.PermCreateToken) {
		return false, ErrNoPermission
	}
	_, created, err = s.register(app, key, token, platform, channel)
	return created, err
}

// Remove deletes a device of an access key.
func (s *Service) Remove(app store.App, key store.AccessKey, token string) error {
	if !key.Can(store.PermDeleteToken) {
		return ErrNoPermission
	}
	if err := s.store.DeleteToken(key.ID, token); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrTokenNotFound
		}
		return err
	}
	_ = s.store.AddLog(app.ID, "token", "removed by key "+key.Prefix)
	return nil
}

func (s *Service) register(app store.App, key store.AccessKey, token, platform, channel string) (store.Token, bool, error) {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > maxTokenLength {
		return store.Token{}, false, ErrInvalidToken
	}
	t := store.Token{
		AppID:   app.ID,
		KeyID:   key.ID,
		Token:   token,
		Device:  orDefault(strings.ToLower(strings.TrimSpace(platform)), defaultDevice),
		Channel: orDefault(strings.TrimSpace(channel), defaultChannel),
	}
	created, err := s.store.UpsertToken(t)
	if err != nil {
		return store.Token{}, false, err
	}
	if created {
		_ = s.store.AddLog(app.ID, "token", "added by key "+key.Prefix)
	}
	return t, created, nil
}

// Message is one notification for one device.
type Message struct {
	Token string
	// Platform and Channel register the device when this key has not done so yet.
	Platform string
	Channel  string
	Notification
}

// Status says how far a delivery got.
type Status string

const (
	// Sent means the push service accepted the notification.
	Sent Status = "sent"
	// Pending means the push service had not answered in time. The delivery carries on in the
	// background, and its outcome goes to the log.
	Pending Status = "pending"
)

// Send delivers a notification to a device. A device the key has not registered yet is registered
// first, if the key may.
//
// Send waits a short while for the push service, so that callers get the real answer (a dead device,
// a refusal) when it is quick. When it is slow, Send answers Pending and the delivery finishes on its
// own: a caller that gives up and hangs up does not cancel it.
func (s *Service) Send(ctx context.Context, app store.App, key store.AccessKey, msg Message) (Status, error) {
	if !key.Can(store.PermSendNotification) {
		return "", ErrNoPermission
	}
	device, err := s.deviceOf(app, key, msg)
	if err != nil {
		return "", err
	}
	provider, credentials, err := s.providerFor(app, device.Device)
	if err != nil {
		return "", err
	}

	done := make(chan error, 1)
	s.running.Add(1)
	go func() {
		defer s.running.Done()
		sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.timeout)
		defer cancel()
		done <- s.deliver(sendCtx, app, key, device, provider, credentials, msg.Notification)
	}()

	timer := time.NewTimer(s.wait)
	defer timer.Stop()
	select {
	case err := <-done:
		if err != nil {
			return "", err
		}
		return Sent, nil
	case <-timer.C:
	case <-ctx.Done():
	}
	return Pending, nil
}

// deviceOf finds the device a message is for, registering it when the key may.
func (s *Service) deviceOf(app store.App, key store.AccessKey, msg Message) (store.Token, error) {
	token := strings.TrimSpace(msg.Token)
	if token == "" || len(token) > maxTokenLength {
		return store.Token{}, ErrInvalidToken
	}
	device, err := s.store.FindToken(key.ID, token)
	if err == nil {
		return device, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.Token{}, err
	}
	if !key.Can(store.PermCreateToken) {
		return store.Token{}, ErrUnknownToken
	}
	device, _, err = s.register(app, key, token, msg.Platform, msg.Channel)
	return device, err
}

// providerFor picks the push service for a device and opens the app's key for it.
func (s *Service) providerFor(app store.App, device string) (Provider, []byte, error) {
	kind := kindFor(device)
	provider := s.providers[kind]
	if provider == nil {
		return nil, nil, ErrNoProvider
	}
	stored, err := s.store.AppProvider(app.ID, kind)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, ErrNoProvider
		}
		return nil, nil, err
	}
	credentials, err := s.opener.Open(stored.Credentials)
	if err != nil {
		_ = s.store.AddLog(app.ID, "error", "the app's key could not be opened: "+err.Error())
		return nil, nil, ErrNoProvider
	}
	return provider, credentials, nil
}

// kindFor says which push service delivers to a device. Firebase reaches Android, iOS and the web,
// so every device goes through it for now.
func kindFor(string) string { return store.ProviderFCM }

// deliver sends one notification and records what came of it.
func (s *Service) deliver(ctx context.Context, app store.App, key store.AccessKey, device store.Token, provider Provider, credentials []byte, n Notification) error {
	err := provider.Send(ctx, credentials, Device{Token: device.Token, Platform: device.Device}, n)
	switch {
	case err == nil:
		_ = s.store.AddLog(app.ID, "push", fmt.Sprintf("sent to %s by key %s", shortToken(device.Token), key.Prefix))
		return nil
	case errors.Is(err, ErrUnregistered):
		_ = s.store.DeleteToken(key.ID, device.Token)
		_ = s.store.AddLog(app.ID, "push", fmt.Sprintf("%s is no longer registered; removed", shortToken(device.Token)))
		return ErrUnregistered
	default:
		_ = s.store.AddLog(app.ID, "error", fmt.Sprintf("sending to %s failed: %s", shortToken(device.Token), err))
		return fmt.Errorf("%w: %v", ErrDelivery, err)
	}
}

// shortToken is enough of a token to recognise it in the log without keeping the whole device address:
// the first characters, never more than half of it.
func shortToken(token string) string {
	runes := []rune(token)
	keep := min(8, len(runes)/2)
	return string(runes[:keep]) + "…"
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
