package fcm

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"

	"github.com/Silify/letspush/internal/gateway"
)

// maxParsed bounds how many parsed keys are kept; far more than there are apps.
const maxParsed = 64

// Provider delivers the gateway's notifications through Firebase. It parses each service-account key
// once instead of on every send.
type Provider struct {
	client *Client

	mu     sync.Mutex
	parsed map[[sha256.Size]byte]Credentials
}

// NewProvider returns a provider for Google's real endpoints.
func NewProvider() *Provider {
	return &Provider{client: NewClient(), parsed: map[[sha256.Size]byte]Credentials{}}
}

// Send delivers a notification. [credentials] is the service-account key file.
func (p *Provider) Send(ctx context.Context, credentials []byte, device gateway.Device, n gateway.Notification) error {
	creds, err := p.credentials(credentials)
	if err != nil {
		return err
	}
	err = p.client.Send(ctx, creds, Message{Token: device.Token, Title: n.Title, Body: n.Body, Data: n.Data})
	if errors.Is(err, ErrUnregistered) {
		return gateway.ErrUnregistered
	}
	return err
}

func (p *Provider) credentials(raw []byte) (Credentials, error) {
	id := sha256.Sum256(raw)

	p.mu.Lock()
	defer p.mu.Unlock()
	if creds, ok := p.parsed[id]; ok {
		return creds, nil
	}
	creds, err := ParseCredentials(raw)
	if err != nil {
		return Credentials{}, err
	}
	if len(p.parsed) >= maxParsed {
		clear(p.parsed)
	}
	p.parsed[id] = creds
	return creds, nil
}
