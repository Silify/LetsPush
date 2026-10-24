package admin

import (
	"sync"
	"time"
)

const (
	maxFailures    = 5
	failureWindow  = 15 * time.Minute
	maxTrackedKeys = 10_000
)

// throttle slows down password guessing: after a few failures for one address and email,
// further attempts are refused for a while.
type throttle struct {
	mu       sync.Mutex
	now      func() time.Time
	failures map[string][]time.Time
}

func newThrottle(now func() time.Time) *throttle {
	return &throttle{now: now, failures: map[string][]time.Time{}}
}

// allowed says whether [key] may try now.
func (t *throttle) allowed(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.recent(key)) < maxFailures
}

// fail records a failed attempt.
func (t *throttle) fail(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.failures) >= maxTrackedKeys {
		t.failures = map[string][]time.Time{} // under attack from many sources: start over rather than grow
	}
	t.failures[key] = append(t.recent(key), t.now())
}

// succeed forgets earlier failures.
func (t *throttle) succeed(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.failures, key)
}

// recent returns the failures inside the window; the caller holds the lock.
func (t *throttle) recent(key string) []time.Time {
	cutoff := t.now().Add(-failureWindow)
	kept := t.failures[key][:0]
	for _, at := range t.failures[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	t.failures[key] = kept
	return kept
}
