package store

import "time"

// LogEntry is one line of the push log.
type LogEntry struct {
	ID        int64
	AppID     *int64
	AppName   string // empty for an event that belongs to no app
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

// LogFilter narrows ListLogs. Zero values mean "any".
type LogFilter struct {
	AppID    int64
	Action   string
	BeforeID int64 // only entries older than this one, to page back
	Limit    int
}

// ListLogs returns the newest entries first.
func (s *Store) ListLogs(f LogFilter) ([]LogEntry, error) {
	query := `SELECT l.id, l.app_id, COALESCE(a.name, ''), l.action, l.info, l.created_at
		FROM logs l LEFT JOIN apps a ON a.id = l.app_id WHERE 1 = 1`
	args := []any{}
	if f.AppID != 0 {
		query += ` AND l.app_id = ?`
		args = append(args, f.AppID)
	}
	if f.Action != "" {
		query += ` AND l.action = ?`
		args = append(args, f.Action)
	}
	if f.BeforeID != 0 {
		query += ` AND l.id < ?`
		args = append(args, f.BeforeID)
	}
	query += ` ORDER BY l.id DESC LIMIT ?`
	args = append(args, f.Limit)

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
		if err := rows.Scan(&e.ID, &app, &e.AppName, &e.Action, &e.Info, &created); err != nil {
			return nil, err
		}
		e.AppID = app
		e.CreatedAt = fromUnix(created)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// RecentLogs returns the newest entries of an app, or of every app when [appID] is 0.
func (s *Store) RecentLogs(appID int64, limit int) ([]LogEntry, error) {
	return s.ListLogs(LogFilter{AppID: appID, Limit: limit})
}

// CountLogs counts the entries of an app (or of every app when [appID] is 0) with this action, since a time.
// A non-empty [infoPrefix] keeps only entries whose text starts with it.
func (s *Store) CountLogs(appID int64, action, infoPrefix string, since time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM logs WHERE action = ? AND created_at >= ?`
	args := []any{action, since.Unix()}
	if appID != 0 {
		query += ` AND app_id = ?`
		args = append(args, appID)
	}
	if infoPrefix != "" {
		query += ` AND substr(info, 1, ?) = ?`
		args = append(args, len([]rune(infoPrefix)), infoPrefix)
	}
	var n int
	err := s.db.QueryRow(query, args...).Scan(&n)
	return n, err
}

// PurgeLogs deletes entries older than [olderThan] and returns how many went.
func (s *Store) PurgeLogs(olderThan time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM logs WHERE created_at < ?`, olderThan.Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
