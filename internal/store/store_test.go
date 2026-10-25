package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func newApp(t *testing.T, s *Store, name string) App {
	t.Helper()
	app, err := s.CreateApp(name, Provider{Kind: ProviderFCM, Label: "project", Credentials: "sealed"})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func newKey(t *testing.T, s *Store, appID int64, hash string) AccessKey {
	t.Helper()
	key, err := s.CreateKey(NewKey{AppID: appID, Hash: hash, Prefix: hash[:4], Permissions: PermAll})
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestOpenCreatesAFileOnlyTheOwnerCanReadAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "letspush.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	newApp(t, s, "one")
	s.Close()

	again, err := Open(path)
	if err != nil {
		t.Fatalf("reopening failed (migrations must be repeatable): %v", err)
	}
	defer again.Close()
	if _, err := again.AppByName("one"); err != nil {
		t.Fatalf("data was lost: %v", err)
	}
}

func TestAdmins(t *testing.T) {
	s := open(t)
	if n, _ := s.CountAdmins(); n != 0 {
		t.Fatalf("expected no admins, got %d", n)
	}
	admin, err := s.CreateAdmin("a@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAdmin("a@example.com", "other"); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate email should conflict, got %v", err)
	}
	got, err := s.AdminByEmail("a@example.com")
	if err != nil || got.ID != admin.ID {
		t.Fatalf("lookup failed: %v", err)
	}
	if _, err := s.AdminByEmail("nobody@example.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if err := s.SetAdminPassword(admin.ID, "new"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.AdminByID(admin.ID); got.PasswordHash != "new" {
		t.Fatal("password was not updated")
	}
	if err := s.SetAdminPassword(999, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestApps(t *testing.T) {
	s := open(t)
	app := newApp(t, s, "sayan")
	if !app.HasCredentials || app.AllowKeyRequests {
		t.Fatalf("unexpected app %+v", app)
	}
	if _, err := s.CreateApp("sayan"); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate name should conflict, got %v", err)
	}
	bare, _ := s.CreateApp("bare")
	if bare.HasCredentials {
		t.Fatal("an app without a key should say so")
	}

	if err := s.SetAllowKeyRequests(app.ID, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.AppByName("sayan"); !got.AllowKeyRequests {
		t.Fatal("the switch was not saved")
	}
	if err := s.SetProvider(app.ID, Provider{Kind: ProviderFCM, Label: "other", Credentials: "resealed"}); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.AppProvider(app.ID, ProviderFCM); p.Credentials != "resealed" || p.Label != "other" {
		t.Fatalf("provider not updated: %+v", p)
	}
	if got, _ := s.AppByID(app.ID); got.ProjectID != "other" {
		t.Fatalf("project id not updated: %q", got.ProjectID)
	}
	if _, err := s.AppProvider(bare.ID, ProviderFCM); !errors.Is(err, ErrNotFound) {
		t.Fatalf("an app without a key should have no provider, got %v", err)
	}
	if err := s.SetProvider(9999, Provider{Kind: ProviderFCM, Credentials: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a missing app should not be found, got %v", err)
	}
	if apps, _ := s.ListApps(); len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}
	if err := s.DeleteApp(app.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppByID(app.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("app should be gone, got %v", err)
	}
}

func TestKeys(t *testing.T) {
	s := open(t)
	app := newApp(t, s, "sayan")
	key := newKey(t, s, app.ID, "hash-one")

	if !key.Can(PermSendNotification) || !key.Can(PermAll) {
		t.Fatal("a key with all permissions should be able to do everything")
	}
	limited, _ := s.CreateKey(NewKey{AppID: app.ID, Hash: "hash-two", Prefix: "two", Permissions: PermCreateToken})
	if limited.Can(PermSendNotification) {
		t.Fatal("a limited key can send")
	}
	if _, err := s.CreateKey(NewKey{AppID: app.ID, Hash: "hash-one", Prefix: "x", Permissions: 1}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate hash should conflict, got %v", err)
	}

	found, err := s.KeyByHash("hash-one")
	if err != nil || found.ID != key.ID || found.LastUsedAt != nil {
		t.Fatalf("lookup failed: %v %+v", err, found)
	}
	if err := s.TouchKey(key.ID); err != nil {
		t.Fatal(err)
	}
	if found, _ := s.KeyByHash("hash-one"); found.LastUsedAt == nil {
		t.Fatal("use was not recorded")
	}

	if err := s.RevokeKey(app.ID, key.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.KeyByHash("hash-one"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a revoked key must not be found, got %v", err)
	}
	if err := s.RevokeKey(app.ID, key.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoking twice should say not found, got %v", err)
	}
	keys, _ := s.ListKeys(app.ID)
	if len(keys) != 2 || keys[1].RevokedAt == nil {
		t.Fatalf("the list should keep revoked keys: %+v", keys)
	}
}

func TestTokensBelongToTheKeyThatRegisteredThem(t *testing.T) {
	s := open(t)
	app := newApp(t, s, "sayan")
	keyA := newKey(t, s, app.ID, "hash-a")
	keyB := newKey(t, s, app.ID, "hash-b")

	created, err := s.UpsertToken(Token{AppID: app.ID, KeyID: keyA.ID, Token: "T", Device: "android", Channel: "default"})
	if err != nil || !created {
		t.Fatalf("first registration should be new: %v %v", created, err)
	}
	created, _ = s.UpsertToken(Token{AppID: app.ID, KeyID: keyA.ID, Token: "T", Device: "android", Channel: "default"})
	if created {
		t.Fatal("registering again should not be new")
	}

	if _, err := s.FindToken(keyA.ID, "T"); err != nil {
		t.Fatalf("the owner should find it: %v", err)
	}
	if _, err := s.FindToken(keyB.ID, "T"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another key must not find it, got %v", err)
	}

	// The same phone on a second site is a second registration.
	if created, _ := s.UpsertToken(Token{AppID: app.ID, KeyID: keyB.ID, Token: "T", Device: "android", Channel: "default"}); !created {
		t.Fatal("a second key registering the same token should be new")
	}

	if err := s.DeleteToken(keyA.ID, "T"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.FindToken(keyB.ID, "T"); err != nil {
		t.Fatal("deleting one key's token removed the other's")
	}
	if err := s.DeleteToken(keyA.ID, "T"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRevokingAKeyRemovesItsTokensAndCountsFollow(t *testing.T) {
	s := open(t)
	app := newApp(t, s, "sayan")
	key := newKey(t, s, app.ID, "hash-a")
	s.UpsertToken(Token{AppID: app.ID, KeyID: key.ID, Token: "T1", Device: "android", Channel: "default"})
	s.UpsertToken(Token{AppID: app.ID, KeyID: key.ID, Token: "T2", Device: "android", Channel: "default"})

	if got, _ := s.AppByID(app.ID); got.Tokens != 2 || got.Keys != 1 {
		t.Fatalf("counts wrong: %+v", got)
	}
	s.RevokeKey(app.ID, key.ID)
	if got, _ := s.AppByID(app.ID); got.Tokens != 0 || got.Keys != 0 {
		t.Fatalf("revoking should clear tokens and the key count: %+v", got)
	}
}

func TestLogs(t *testing.T) {
	s := open(t)
	app := newApp(t, s, "sayan")
	other := newApp(t, s, "other")

	base := time.Unix(1_700_000_000, 0)
	s.now = func() time.Time { return base.Add(-48 * time.Hour) }
	s.AddLog(app.ID, "push", "old")
	s.now = func() time.Time { return base }
	s.AddLog(app.ID, "push", "new")
	s.AddLog(other.ID, "push", "elsewhere")
	s.AddLog(0, "start", "")

	if logs, _ := s.RecentLogs(app.ID, 10); len(logs) != 2 || logs[0].Info != "new" {
		t.Fatalf("newest first and only this app: %+v", logs)
	}
	if logs, _ := s.RecentLogs(0, 2); len(logs) != 2 {
		t.Fatalf("the limit was ignored: %d", len(logs))
	}
	if n, _ := s.PurgeLogs(base.Add(-24 * time.Hour)); n != 1 {
		t.Fatalf("expected to purge 1, purged %d", n)
	}
}
