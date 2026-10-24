package store

import "time"

// LogEntry is one line of the push log.
type LogEntry struct {
	ID        int64
	AppID     *int64
	Action    string
	Info      string
	CreatedAt time.Time
}

// AddLog records an event. [appID] is 0 when the event belongs to no app.
func (s *Store) AddLog(appID int64, action, info string) error {
	var app any
	if appID != 0 {
		app = appID
	}
	_, err := s.db.Exec(`INSERT INTO logs (app_id, action, info, created_at) VALUES (?, ?, ?, ?)`, app, action, info, s.unix())
	return err
}

// RecentLogs returns the newest entries of an app, or of every app when [appID] is 0.
func (s *Store) RecentLogs(appID int64, limit int) ([]LogEntry, error) {
	query := `SELECT id, app_id, action, info, created_at FROM logs `
	args := []any{}
	if appID != 0 {
		query += `WHERE app_id = ? `
		args = append(args, appID)
	}
	query += `ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []LogEntry{}
	for rows.Next() {
		var e LogEntry
		var app *int64
		var created int64
		if err := rows.Scan(&e.ID, &app, &e.Action, &e.Info, &created); err != nil {
			return nil, err
		}
		e.AppID = app
		e.CreatedAt = fromUnix(created)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// PurgeLogs deletes entries older than [olderThan] and returns how many went.
func (s *Store) PurgeLogs(olderThan time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM logs WHERE created_at < ?`, olderThan.Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
