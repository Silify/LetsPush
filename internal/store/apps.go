package store

import (
	"database/sql"
	"time"
)

// ProviderFCM is the kind of the Firebase Cloud Messaging provider, which delivers to Android, iOS and the web.
const ProviderFCM = "fcm"

// Provider is the key an app uses to reach one push service.
type Provider struct {
	Kind        string
	Label       string // shown in lists; for Firebase, the project id
	Credentials string // sealed
}

// App is one app that notifications are sent to.
type App struct {
	ID   int64
	Name string
	// ProjectID and HasCredentials describe the app's Firebase provider, the only one there is for now.
	ProjectID        string
	HasCredentials   bool
	AllowKeyRequests bool
	CreatedAt        time.Time
	Keys             int
	Tokens           int
}

const appColumns = `a.id, a.name,
	COALESCE((SELECT p.label FROM app_providers p WHERE p.app_id = a.id AND p.kind = 'fcm'), ''),
	EXISTS (SELECT 1 FROM app_providers p WHERE p.app_id = a.id AND p.kind = 'fcm'),
	a.allow_key_requests, a.created_at,
	(SELECT COUNT(*) FROM access_keys k WHERE k.app_id = a.id AND k.revoked_at IS NULL),
	(SELECT COUNT(*) FROM tokens t WHERE t.app_id = a.id)`

func scanApp(row rowScanner) (App, error) {
	var a App
	var created int64
	if err := row.Scan(&a.ID, &a.Name, &a.ProjectID, &a.HasCredentials, &a.AllowKeyRequests, &created, &a.Keys, &a.Tokens); err != nil {
		return App{}, wrap(err)
	}
	a.CreatedAt = fromUnix(created)
	return a, nil
}

// CreateApp adds an app, with its providers if it has any yet.
func (s *Store) CreateApp(name string, providers ...Provider) (App, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return App{}, err
	}
	defer tx.Rollback()

	now := s.unix()
	res, err := tx.Exec(`INSERT INTO apps (name, created_at) VALUES (?, ?)`, name, now)
	if err != nil {
		return App{}, wrap(err)
	}
	id, _ := res.LastInsertId()
	for _, p := range providers {
		if err := putProvider(tx, id, p, now); err != nil {
			return App{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return App{}, err
	}
	return s.AppByID(id)
}

// AppByID finds an app.
func (s *Store) AppByID(id int64) (App, error) {
	return scanApp(s.db.QueryRow(`SELECT `+appColumns+` FROM apps a WHERE a.id = ?`, id))
}

// AppByName finds an app by the name sites send in their requests.
func (s *Store) AppByName(name string) (App, error) {
	return scanApp(s.db.QueryRow(`SELECT `+appColumns+` FROM apps a WHERE a.name = ?`, name))
}

// ListApps returns every app, oldest first.
func (s *Store) ListApps() ([]App, error) {
	rows, err := s.db.Query(`SELECT ` + appColumns + ` FROM apps a ORDER BY a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	apps := []App{}
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// AppProvider returns the app's key for a provider, still sealed.
func (s *Store) AppProvider(appID int64, kind string) (Provider, error) {
	p := Provider{Kind: kind}
	err := s.db.QueryRow(`SELECT label, credentials FROM app_providers WHERE app_id = ? AND kind = ?`, appID, kind).
		Scan(&p.Label, &p.Credentials)
	return p, wrap(err)
}

// SetProvider adds an app's key for a provider, or replaces it.
func (s *Store) SetProvider(appID int64, p Provider) error {
	return putProvider(s.db, appID, p, s.unix())
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func putProvider(db execer, appID int64, p Provider, now int64) error {
	_, err := db.Exec(`INSERT INTO app_providers (app_id, kind, label, credentials, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (app_id, kind) DO UPDATE SET label = excluded.label, credentials = excluded.credentials, updated_at = excluded.updated_at`,
		appID, p.Kind, p.Label, p.Credentials, now)
	return wrap(err)
}

// SetAllowKeyRequests switches whether sites may request their own access key.
func (s *Store) SetAllowKeyRequests(id int64, allow bool) error {
	return s.exec1(`UPDATE apps SET allow_key_requests = ? WHERE id = ?`, allow, id)
}

// DeleteApp removes an app with its keys, tokens and log.
func (s *Store) DeleteApp(id int64) error {
	return s.exec1(`DELETE FROM apps WHERE id = ?`, id)
}

// exec1 runs a statement that must change exactly one row.
func (s *Store) exec1(query string, args ...any) error {
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return wrap(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
