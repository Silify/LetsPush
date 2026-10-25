package fcm

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Silify/letspush/internal/gateway"
)

func newProvider(g *google) *Provider {
	p := NewProvider()
	p.client = g.client(nil)
	return p
}

func TestProviderSendsTheGatewaysNotification(t *testing.T) {
	g := newGoogle(t)
	p := newProvider(g)
	raw := keyFile(t, newKey(t))

	err := p.Send(context.Background(), raw, gateway.Device{Token: "device", Platform: "android"},
		gateway.Notification{Title: "Site", Body: "Hello", Data: map[string]string{"a": "1"}})
	if err != nil {
		t.Fatal(err)
	}
	if g.sends.Load() != 1 {
		t.Fatalf("expected one send, got %d", g.sends.Load())
	}
}

func TestProviderParsesAKeyOnce(t *testing.T) {
	p := newProvider(newGoogle(t))
	raw := keyFile(t, newKey(t))
	for range 3 {
		if _, err := p.credentials(raw); err != nil {
			t.Fatal(err)
		}
	}
	if len(p.parsed) != 1 {
		t.Fatalf("expected the key to be cached once, got %d", len(p.parsed))
	}
	if _, err := p.credentials([]byte("not a key")); err == nil {
		t.Fatal("a bad key should fail")
	}
	if len(p.parsed) != 1 {
		t.Fatal("a bad key was cached")
	}
}

func TestProviderReportsADeadDeviceInTheGatewaysWords(t *testing.T) {
	g := newGoogle(t)
	g.sendReply = func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"status":"NOT_FOUND","details":[{"errorCode":"UNREGISTERED"}]}}`))
	}
	p := newProvider(g)
	err := p.Send(context.Background(), keyFile(t, newKey(t)), gateway.Device{Token: "gone"}, gateway.Notification{Title: "t"})
	if !errors.Is(err, gateway.ErrUnregistered) {
		t.Fatalf("got %v", err)
	}
}
