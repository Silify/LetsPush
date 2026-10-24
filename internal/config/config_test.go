package config

import (
	"strings"
	"testing"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

var secret = strings.Repeat("s", 32)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(map[string]string{"LETSPUSH_SECRET": secret}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "127.0.0.1:8080" || cfg.DatabasePath != "./data/letspush.db" || cfg.LogRetentionDays != 30 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRefusesBadSettings(t *testing.T) {
	cases := map[string]map[string]string{
		"no secret":          {},
		"short secret":       {"LETSPUSH_SECRET": "short"},
		"email without pass": {"LETSPUSH_SECRET": secret, "LETSPUSH_ADMIN_EMAIL": "a@b.c"},
		"pass without email": {"LETSPUSH_SECRET": secret, "LETSPUSH_ADMIN_PASSWORD": "longenoughpass"},
		"short password":     {"LETSPUSH_SECRET": secret, "LETSPUSH_ADMIN_EMAIL": "a@b.c", "LETSPUSH_ADMIN_PASSWORD": "short"},
		"bad retention":      {"LETSPUSH_SECRET": secret, "LETSPUSH_LOG_RETENTION_DAYS": "0"},
		"text retention":     {"LETSPUSH_SECRET": secret, "LETSPUSH_LOG_RETENTION_DAYS": "soon"},
	}
	for name, values := range cases {
		if _, err := Load(env(values)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestLoadAdmin(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"LETSPUSH_SECRET":         secret,
		"LETSPUSH_ADMIN_EMAIL":    " admin@example.com ",
		"LETSPUSH_ADMIN_PASSWORD": "a-long-password",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AdminEmail != "admin@example.com" {
		t.Fatalf("email not trimmed: %q", cfg.AdminEmail)
	}
}
