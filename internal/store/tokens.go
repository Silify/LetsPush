package store

import (
	"database/sql"
	"errors"
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
