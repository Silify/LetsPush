package store

import (
	"database/sql"
	"errors"
	"time"
)

// Token is one device, registered by one access key.
type Token struct {
	ID      int64
	AppID   int64
	KeyID   int64
	Token   string
	Device  string
	Channel string
}

// UpsertToken registers a device for a key. It reports whether the token was new.
func (s *Store) UpsertToken(t Token) (created bool, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	now := s.unix()
	var id int64
	err = tx.QueryRow(`SELECT id FROM tokens WHERE key_id = ? AND token = ?`, t.KeyID, t.Token).Scan(&id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.Exec(`INSERT INTO tokens (app_id, key_id, token, device, channel, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, t.AppID, t.KeyID, t.Token, t.Device, t.Channel, now, now)
		created = true
	case err == nil:
		_, err = tx.Exec(`UPDATE tokens SET device = ?, channel = ?, updated_at = ? WHERE id = ?`, t.Device, t.Channel, now, id)
	}
	if err != nil {
		return false, wrap(err)
	}
	return created, tx.Commit()
}

// DeviceRow is a device with the access key that registered it, for the admin's list.
type DeviceRow struct {
	ID             int64
	Token          string
	Platform       string
	Channel        string
	KeyPrefix      string
	KeyDescription string
	KeyRevoked     bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ListDevices returns an app's devices, newest first. [beforeID] pages back from a device id (0 for the
// first page).
func (s *Store) ListDevices(appID, beforeID int64, limit int) ([]DeviceRow, error) {
	rows, err := s.db.Query(`SELECT t.id, t.token, t.device, t.channel, k.prefix, k.description, k.revoked_at IS NOT NULL,
			t.created_at, t.updated_at
		FROM tokens t JOIN access_keys k ON k.id = t.key_id
		WHERE t.app_id = ? AND (? = 0 OR t.id < ?)
		ORDER BY t.id DESC LIMIT ?`, appID, beforeID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	devices := []DeviceRow{}
	for rows.Next() {
		var d DeviceRow
		var created, updated int64
		if err := rows.Scan(&d.ID, &d.Token, &d.Platform, &d.Channel, &d.KeyPrefix, &d.KeyDescription, &d.KeyRevoked, &created, &updated); err != nil {
			return nil, err
		}
		d.CreatedAt, d.UpdatedAt = fromUnix(created), fromUnix(updated)
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// DeleteDevice removes one device of an app.
func (s *Store) DeleteDevice(appID, id int64) error {
	return s.exec1(`DELETE FROM tokens WHERE app_id = ? AND id = ?`, appID, id)
}

// FindToken returns a device registered by this key.
func (s *Store) FindToken(keyID int64, token string) (Token, error) {
	var t Token
	err := s.db.QueryRow(`SELECT id, app_id, key_id, token, device, channel FROM tokens WHERE key_id = ? AND token = ?`, keyID, token).
		Scan(&t.ID, &t.AppID, &t.KeyID, &t.Token, &t.Device, &t.Channel)
	return t, wrap(err)
}

// DeleteToken removes a device registered by this key.
func (s *Store) DeleteToken(keyID int64, token string) error {
	return s.exec1(`DELETE FROM tokens WHERE key_id = ? AND token = ?`, keyID, token)
}
