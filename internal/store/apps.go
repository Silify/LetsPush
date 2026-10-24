package store

import "time"

// App is one mobile app (one Firebase project) that notifications are sent to.
type App struct {
	ID               int64
	Name             string
	ProjectID        string
	HasCredentials   bool
	AllowKeyRequests bool
	CreatedAt        time.Time
	Keys             int
	Tokens           int
}

const appColumns = `a.id, a.name, a.project_id, a.credentials <> '', a.allow_key_requests, a.created_at,
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

// CreateApp adds an app. [credentials] is the already sealed Firebase key, or empty.
func (s *Store) CreateApp(name, projectID, credentials string) (App, error) {
	res, err := s.db.Exec(`INSERT INTO apps (name, project_id, credentials, created_at) VALUES (?, ?, ?, ?)`,
		name, projectID, credentials, s.unix())
	if err != nil {
		return App{}, wrap(err)
	}
	id, _ := res.LastInsertId()
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

// AppCredentials returns the sealed Firebase key of an app.
func (s *Store) AppCredentials(id int64) (string, error) {
	var sealed string
	err := s.db.QueryRow(`SELECT credentials FROM apps WHERE id = ?`, id).Scan(&sealed)
	return sealed, wrap(err)
}

// UpdateAppCredentials replaces the Firebase key and project of an app.
func (s *Store) UpdateAppCredentials(id int64, projectID, credentials string) error {
	return s.exec1(`UPDATE apps SET project_id = ?, credentials = ? WHERE id = ?`, projectID, credentials, id)
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
