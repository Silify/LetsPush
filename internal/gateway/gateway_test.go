package gateway

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
)

// fakeProvider answers like a push service: at once, when released, or with an error.
type fakeProvider struct {
	mu      sync.Mutex
	sent    []Device
	err     error
	ctxErr  error         // the context's state when the send was let go
	release chan struct{} // when set, Send blocks until it is closed
}

func (f *fakeProvider) Send(ctx context.Context, _ []byte, device Device, _ Notification) error {
	if f.release != nil {
		<-f.release
	}
	f.mu.Lock()
	f.ctxErr = ctx.Err()
	f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	f.sent = append(f.sent, device)
	f.mu.Unlock()
	return nil
}

func (f *fakeProvider) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

type env struct {
	t        *testing.T
	store    *store.Store
	provider *fakeProvider
	service  *Service
	app      store.App
	key      store.AccessKey
}

func newEnv(t *testing.T, permissions int, wait time.Duration) *env {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	box, _ := secret.NewBox(strings.Repeat("k", 32))
	sealed, _ := box.Seal([]byte("the key"))
	app, err := db.CreateApp("sayan", store.Provider{Kind: store.ProviderFCM, Label: "p", Credentials: sealed})
	if err != nil {
		t.Fatal(err)
	}
	_, prefix, hash, _ := secret.NewAccessKey()
	key, err := db.CreateKey(store.NewKey{AppID: app.ID, Hash: hash, Prefix: prefix, Permissions: permissions})
	if err != nil {
		t.Fatal(err)
	}
	provider := &fakeProvider{}
	service := New(Config{
		Store:     db,
		Opener:    box,
		Providers: map[string]Provider{store.ProviderFCM: provider},
		Wait:      wait,
		Timeout:   5 * time.Second,
	})
	t.Cleanup(service.Wait)
	return &env{t, db, provider, service, app, key}
}

func (e *env) send(token string) (Status, error) {
	return e.service.Send(context.Background(), e.app, e.key, Message{Token: token, Notification: Notification{Title: "t"}})
}

func (e *env) hasToken(token string) bool {
	_, err := e.store.FindToken(e.key.ID, token)
	return err == nil
}

func (e *env) lastLog() store.LogEntry {
	e.t.Helper()
	logs, err := e.store.RecentLogs(e.app.ID, 1)
	if err != nil || len(logs) == 0 {
		e.t.Fatalf("no log entry: %v", err)
	}
	return logs[0]
}

func TestSendDeliversAndRegistersTheDevice(t *testing.T) {
	e := newEnv(t, store.PermAll, time.Second)
	status, err := e.send("DEVICE-1")
	if err != nil || status != Sent {
		t.Fatalf("got %q, %v", status, err)
	}
	if !e.hasToken("DEVICE-1") {
		t.Fatal("an unknown device should be registered on first use")
	}
	if e.provider.count() != 1 {
		t.Fatalf("expected one delivery, got %d", e.provider.count())
	}
	if log := e.lastLog(); log.Action != "push" {
		t.Fatalf("unexpected log %+v", log)
	}
}

func TestSendChecksPermissionsAndTokens(t *testing.T) {
	t.Run("a key that may not send", func(t *testing.T) {
		e := newEnv(t, store.PermCreateToken, time.Second)
		if _, err := e.send("T"); !errors.Is(err, ErrNoPermission) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("an unknown device and a key that may not register it", func(t *testing.T) {
		e := newEnv(t, store.PermSendNotification, time.Second)
		if _, err := e.send("T"); !errors.Is(err, ErrUnknownToken) {
			t.Fatalf("got %v", err)
		}
		if e.provider.count() != 0 {
			t.Fatal("sent to an unregistered device")
		}
	})
	t.Run("a token that is empty or too long", func(t *testing.T) {
		e := newEnv(t, store.PermAll, time.Second)
		for _, token := range []string{"", "  ", strings.Repeat("x", 4097)} {
			if _, err := e.send(token); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("token %q: got %v", token, err)
			}
		}
	})
	t.Run("an app without a key", func(t *testing.T) {
		e := newEnv(t, store.PermAll, time.Second)
		bare, _ := e.store.CreateApp("bare")
		e.app = bare
		_, prefix, hash, _ := secret.NewAccessKey()
		e.key, _ = e.store.CreateKey(store.NewKey{AppID: bare.ID, Hash: hash, Prefix: prefix, Permissions: store.PermAll})
		if _, err := e.send("T"); !errors.Is(err, ErrNoProvider) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestSendOutcomes(t *testing.T) {
	t.Run("a dead device is removed", func(t *testing.T) {
		e := newEnv(t, store.PermAll, time.Second)
		e.provider.err = ErrUnregistered
		if _, err := e.send("DEAD"); !errors.Is(err, ErrUnregistered) {
			t.Fatalf("got %v", err)
		}
		if e.hasToken("DEAD") {
			t.Fatal("the dead token was kept")
		}
	})
	t.Run("another failure keeps the device and is logged", func(t *testing.T) {
		e := newEnv(t, store.PermAll, time.Second)
		e.provider.err = errors.New("firebase send failed with status 403")
		_, err := e.send("OK")
		if !errors.Is(err, ErrDelivery) {
			t.Fatalf("got %v", err)
		}
		if !e.hasToken("OK") {
			t.Fatal("the token was removed on a failure")
		}
		if log := e.lastLog(); log.Action != "error" || !strings.Contains(log.Info, "status 403") {
			t.Fatalf("unexpected log %+v", log)
		}
	})
}

func TestASlowProviderMakesSendAnswerPendingAndTheDeliveryFinishesAlone(t *testing.T) {
	e := newEnv(t, store.PermAll, 20*time.Millisecond)
	e.provider.release = make(chan struct{})

	status, err := e.send("SLOW")
	if err != nil || status != Pending {
		t.Fatalf("got %q, %v; want pending", status, err)
	}
	close(e.provider.release)
	e.service.Wait()

	if e.provider.count() != 1 {
		t.Fatal("the delivery did not finish after Send answered")
	}
	if log := e.lastLog(); log.Action != "push" {
		t.Fatalf("the outcome was not logged: %+v", log)
	}
}

func TestADeadDeviceFoundAfterSendAnsweredIsStillRemoved(t *testing.T) {
	e := newEnv(t, store.PermAll, 20*time.Millisecond)
	e.provider.release = make(chan struct{})
	e.provider.err = ErrUnregistered

	if status, _ := e.send("DEAD"); status != Pending {
		t.Fatalf("got %q; want pending", status)
	}
	close(e.provider.release)
	e.service.Wait()
	if e.hasToken("DEAD") {
		t.Fatal("the dead token was kept")
	}
}

func TestACallerThatHangsUpDoesNotCancelTheDelivery(t *testing.T) {
	e := newEnv(t, store.PermAll, time.Second)
	e.provider.release = make(chan struct{})

	ctx, hangUp := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		hangUp()
	}()
	status, err := e.service.Send(ctx, e.app, e.key, Message{Token: "T"})
	if err != nil || status != Pending {
		t.Fatalf("got %q, %v; want pending", status, err)
	}

	close(e.provider.release)
	e.service.Wait()
	if e.provider.count() != 1 {
		t.Fatal("hanging up cancelled the delivery")
	}
	e.provider.mu.Lock()
	err = e.provider.ctxErr
	e.provider.mu.Unlock()
	if err != nil {
		t.Fatal("the provider was given the caller's cancelled context")
	}
}

func TestRegisterAndRemove(t *testing.T) {
	e := newEnv(t, store.PermAll, time.Second)

	created, err := e.service.Register(e.app, e.key, " T1 ", "Android", "")
	if err != nil || !created {
		t.Fatalf("got %v, %v", created, err)
	}
	if created, _ := e.service.Register(e.app, e.key, "T1", "android", ""); created {
		t.Fatal("registering again should not create")
	}
	got, err := e.store.FindToken(e.key.ID, "T1")
	if err != nil || got.Device != "android" || got.Channel != "default" {
		t.Fatalf("stored wrongly: %v %+v", err, got)
	}

	if err := e.service.Remove(e.app, e.key, "T1"); err != nil {
		t.Fatal(err)
	}
	if err := e.service.Remove(e.app, e.key, "T1"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("got %v", err)
	}

	limited := newEnv(t, store.PermSendNotification, time.Second)
	if _, err := limited.service.Register(limited.app, limited.key, "T", "", ""); !errors.Is(err, ErrNoPermission) {
		t.Fatalf("got %v", err)
	}
	if err := limited.service.Remove(limited.app, limited.key, "T"); !errors.Is(err, ErrNoPermission) {
		t.Fatalf("got %v", err)
	}
}
