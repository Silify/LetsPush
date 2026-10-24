// Command letspush runs the push gateway.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Silify/letspush/internal/config"
	"github.com/Silify/letspush/internal/fcm"
	"github.com/Silify/letspush/internal/secret"
	"github.com/Silify/letspush/internal/server"
	"github.com/Silify/letspush/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("letspush: %v", err)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

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
		log.Printf("created the first admin: %s", cfg.AdminEmail)
	}
	if n, _ := db.CountAdmins(); n == 0 {
		log.Printf("there is no admin yet: set LETSPUSH_ADMIN_EMAIL and LETSPUSH_ADMIN_PASSWORD and restart")
	}

	handler, err := server.New(server.Options{Store: db, Box: box, Sender: fcm.NewClient(), Secret: cfg.Secret})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go server.PurgeLogs(ctx, db, cfg.LogRetentionDays, time.Hour)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	failed := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", cfg.Addr)
		failed <- srv.ListenAndServe()
	}()

	select {
	case err := <-failed:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Printf("shutting down")
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	}
	return nil
}
