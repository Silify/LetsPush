// Command letspush runs the push gateway.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Silify/letspush/internal/config"
	"github.com/Silify/letspush/internal/fcm"
	"github.com/Silify/letspush/internal/gateway"
	"github.com/Silify/letspush/internal/reqid"
	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/server"
	"github.com/Silify/letspush/internal/store"
)

// version is set at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("letspush", version)
		return
	}
	if err := run(); err != nil {
		slog.Error("letspush stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	logger := newLogger(cfg.LogFormat)
	slog.SetDefault(logger)

	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	box, err := secret.NewBox(cfg.Secret)
	if err != nil {
		return err
	}

	created, err := server.EnsureAdmin(db, cfg.AdminEmail, cfg.AdminPassword)
	if err != nil {
		return err
	}
	if created {
		slog.Info("created the first admin", "email", cfg.AdminEmail)
	}
	if n, _ := db.CountAdmins(); n == 0 {
		slog.Warn("there is no admin yet: set LETSPUSH_ADMIN_EMAIL and LETSPUSH_ADMIN_PASSWORD and restart")
	}

	gw := gateway.New(gateway.Config{
		Store:     db,
		Opener:    box,
		Providers: map[string]gateway.Provider{store.ProviderFCM: fcm.NewProvider()},
	})
	defer gw.Wait() // let deliveries that are still running finish

	handler, err := server.New(server.Options{Store: db, Box: box, Gateway: gw, Secret: cfg.Secret})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go server.PurgeLogs(ctx, db, cfg.LogRetentionDays, time.Hour)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	failed := make(chan error, 1)
	go func() {
		slog.Info("listening", "version", version, "addr", cfg.Addr)
		failed <- srv.ListenAndServe()
	}()

	select {
	case err := <-failed:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	}
	return nil
}

// newLogger writes the server's own log to standard error, as plain text or as JSON lines.
// Records logged with a request's context carry its request_id.
func newLogger(format string) *slog.Logger {
	var h slog.Handler = slog.NewTextHandler(os.Stderr, nil)
	if format == "json" {
		h = slog.NewJSONHandler(os.Stderr, nil)
	}
	return slog.New(reqid.Handler(h))
}
