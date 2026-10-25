// Package store keeps Let's Push's data in SQLite.
package store

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite" // the database/sql driver, registered as "sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

var (
	// ErrNotFound means no row matches.
	ErrNotFound = errors.New("not found")
	// ErrConflict means the row would duplicate a unique value.
	ErrConflict = errors.New("already exists")
)

// Store is the database.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

// Open opens (and creates) the database file, or ":memory:", and brings the schema up to date.
func Open(path string) (*Store, error) {
	dsn := path
	if path == ":memory:" {
		dsn = "file::memory:"
	} else {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		dsn = "file:" + filepath.ToSlash(path)
	}
	dsn += "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// One connection: SQLite writes are serial anyway, and an in-memory database lives per connection.
	db.SetMaxOpenConns(1)

	s := &Store{db: db, now: time.Now}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	if path != ":memory:" {
		// The file holds sealed keys and hashes: only the server's user may read it.
		_ = os.Chmod(path, 0o600)
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var done string
		err := s.db.QueryRow(`SELECT name FROM schema_migrations WHERE name = ?`, name).Scan(&done)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		script, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(script)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) unix() int64 { return s.now().Unix() }

func fromUnix(seconds int64) time.Time { return time.Unix(seconds, 0).UTC() }

func fromNullUnix(seconds sql.NullInt64) *time.Time {
	if !seconds.Valid {
		return nil
	}
	t := fromUnix(seconds.Int64)
	return &t
}

// wrap turns the driver's errors into the package's own.
func wrap(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, sql.ErrNoRows):
		return ErrNotFound
	case strings.Contains(err.Error(), "UNIQUE constraint failed"):
		return ErrConflict
	case strings.Contains(err.Error(), "FOREIGN KEY constraint failed"):
		return ErrNotFound
	}
	return err
}
