package store

import (
	"testing"
	"time"
)

func TestAKeysLastUseIsWrittenAtMostOncePerMinute(t *testing.T) {
	s := open(t)
	key := newKey(t, s, newApp(t, s, "sayan").ID, "hash-touch")
	base := time.Unix(1_700_000_000, 0)
	at := func(d time.Duration) {
		s.now = func() time.Time { return base.Add(d) }
		if err := s.TouchKey(key.ID); err != nil {
			t.Fatal(err)
		}
	}
	lastUsed := func() time.Time {
		found, err := s.KeyByHash("hash-touch")
		if err != nil || found.LastUsedAt == nil {
			t.Fatalf("no last use: %v", err)
		}
		return *found.LastUsedAt
	}

	at(0)
	if got := lastUsed(); !got.Equal(base) {
		t.Fatalf("first use not recorded: %v", got)
	}
	at(30 * time.Second)
	if got := lastUsed(); !got.Equal(base) {
		t.Fatalf("written again within a minute: %v", got)
	}
	at(61 * time.Second)
	if got := lastUsed(); !got.Equal(base.Add(61 * time.Second)) {
		t.Fatalf("not written after a minute: %v", got)
	}
}
