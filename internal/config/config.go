// Package config reads Let's Push's settings from the environment.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Config holds every setting the server needs.
type Config struct {
	// Addr is where the HTTP server listens. Put a reverse proxy with HTTPS in front of it.
	Addr string
	// DatabasePath is the SQLite file.
	DatabasePath string
	// Secret protects stored Firebase keys and signs login sessions. At least 32 characters.
	Secret string
	// AdminEmail and AdminPassword create the first admin when there is none.
	AdminEmail    string
	AdminPassword string
	// LogRetentionDays is how long the push log is kept.
	LogRetentionDays int
	// LogFormat is how the server writes its own log: "text" or "json".
	LogFormat string
}

const minSecretLength = 32

// Load reads the settings from [getenv], which is os.Getenv in the real program.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Addr:             valueOr(getenv("LETSPUSH_ADDR"), "127.0.0.1:8080"),
		DatabasePath:     valueOr(getenv("LETSPUSH_DB"), "./data/letspush.db"),
		Secret:           getenv("LETSPUSH_SECRET"),
		AdminEmail:       strings.TrimSpace(getenv("LETSPUSH_ADMIN_EMAIL")),
		AdminPassword:    getenv("LETSPUSH_ADMIN_PASSWORD"),
		LogRetentionDays: 30,
		LogFormat:        valueOr(strings.ToLower(strings.TrimSpace(getenv("LETSPUSH_LOG_FORMAT"))), "text"),
	}

	if len(cfg.Secret) < minSecretLength {
		return Config{}, fmt.Errorf("LETSPUSH_SECRET must be at least %d characters (try: openssl rand -hex 32)", minSecretLength)
	}

	if raw := getenv("LETSPUSH_LOG_RETENTION_DAYS"); raw != "" {
		days, err := strconv.Atoi(raw)
		if err != nil || days < 1 {
			return Config{}, errors.New("LETSPUSH_LOG_RETENTION_DAYS must be a whole number of days, 1 or more")
		}
		cfg.LogRetentionDays = days
	}

	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		return Config{}, errors.New(`LETSPUSH_LOG_FORMAT must be "text" or "json"`)
	}

	if (cfg.AdminEmail == "") != (cfg.AdminPassword == "") {
		return Config{}, errors.New("set both LETSPUSH_ADMIN_EMAIL and LETSPUSH_ADMIN_PASSWORD, or neither")
	}
	if cfg.AdminPassword != "" && len(cfg.AdminPassword) < 10 {
		return Config{}, errors.New("LETSPUSH_ADMIN_PASSWORD must be at least 10 characters")
	}

	return cfg, nil
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
