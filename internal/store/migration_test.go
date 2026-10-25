package store

import (
	"database/sql"
	"testing"
	"time"
)

func TestTheProvidersMigrationKeepsTheFirebaseKeysOfExistingApps(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	// A database as the first version left it.
	first, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		string(first),
		`CREATE TABLE schema_migrations (name TEXT PRIMARY KEY)`,
		`INSERT INTO schema_migrations (name) VALUES ('001_init.sql')`,
		`INSERT INTO apps (name, project_id, credentials, created_at) VALUES ('keyed', 'proj', 'sealed-key', 1)`,
		`INSERT INTO apps (name, created_at) VALUES ('bare', 2)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	s := &Store{db: db, now: time.Now}
	if err := s.migrate(); err != nil {
		t.Fatal(err)
	}

	keyed, _ := s.AppByName("keyed")
	if !keyed.HasCredentials || keyed.ProjectID != "proj" {
		t.Fatalf("the key was lost: %+v", keyed)
	}
	if p, err := s.AppProvider(keyed.ID, ProviderFCM); err != nil || p.Credentials != "sealed-key" {
		t.Fatalf("unexpected provider %+v, %v", p, err)
	}
	if bare, _ := s.AppByName("bare"); bare.HasCredentials {
		t.Fatal("an app without a key gained one")
	}
}
