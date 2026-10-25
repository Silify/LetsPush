package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/store"
)

// EnsureAdmin creates the first admin from the settings, when there is none yet.
// Once any admin exists the settings are ignored, so changing the environment cannot
// quietly add or reset an account.
func EnsureAdmin(s *store.Store, email, password string) (created bool, err error) {
	if email == "" {
		return false, nil
	}
	count, err := s.CountAdmins()
	if err != nil || count > 0 {
		return false, err
	}
	hash, err := secret.HashPassword(password)
	if err != nil {
		return false, err
	}
	if _, err := s.CreateAdmin(email, hash); err != nil {
		return false, err
	}
	return true, nil
}

// PurgeLogs deletes old log entries now and then until [ctx] ends.
func PurgeLogs(ctx context.Context, s *store.Store, retentionDays int, every time.Duration) {
	purge := func() {
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		if n, err := s.PurgeLogs(cutoff); err != nil {
			slog.Error("could not purge the log", "error", err)
		} else if n > 0 {
			slog.Info("purged old log entries", "count", n)
		}
		if n, err := s.PurgeDeliveries(cutoff); err != nil {
			slog.Error("could not purge old deliveries", "error", err)
		} else if n > 0 {
			slog.Info("purged old deliveries", "count", n)
		}
	}
	purge()
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			purge()
		}
	}
}
