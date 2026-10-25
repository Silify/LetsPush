package store

import (
	"database/sql"
	"time"
)

// Permissions an access key can carry.
const (
	PermCreateToken      = 1
	PermDeleteToken      = 2
	PermSendNotification = 4
	PermAll              = PermCreateToken | PermDeleteToken | PermSendNotification
)

// AccessKey lets one site talk to one app. The key itself is never stored, only its hash.
type AccessKey struct {
	ID          int64
	AppID       int64
	Prefix      string
	Description string
	Contact     string
	Permissions int
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
	Tokens      int
}

// Can reports whether the key has every bit of [permission].
func (k AccessKey) Can(permission int) bool { return k.Permissions&permission == permission }

const keyColumns = `k.id, k.app_id, k.prefix, k.description, k.contact, k.permissions, k.created_at, k.last_used_at, k.revoked_at,
	(SELECT COUNT(*) FROM tokens t WHERE t.key_id = k.id)`

func scanKey(row rowScanner) (AccessKey, error) {
	var k AccessKey
	var created int64
	var used, revoked sql.NullInt64
	if err := row.Scan(&k.ID, &k.AppID, &k.Prefix, &k.Description, &k.Contact, &k.Permissions, &created, &used, &revoked, &k.Tokens); err != nil {
		return AccessKey{}, wrap(err)
	}
	k.CreatedAt = fromUnix(created)
	k.LastUsedAt = fromNullUnix(used)
	k.RevokedAt = fromNullUnix(revoked)
	return k, nil
}

// NewKey describes a key to add.
type NewKey struct {
	AppID       int64
	Hash        string
	Prefix      string
	Description string
	Contact     string
	Permissions int
}

// CreateKey stores the hash of a new access key.
func (s *Store) CreateKey(n NewKey) (AccessKey, error) {
	res, err := s.db.Exec(`INSERT INTO access_keys (app_id, key_hash, prefix, description, contact, permissions, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, n.AppID, n.Hash, n.Prefix, n.Description, n.Contact, n.Permissions, s.unix())
	if err != nil {
		return AccessKey{}, wrap(err)
	}
	id, _ := res.LastInsertId()
	return scanKey(s.db.QueryRow(`SELECT `+keyColumns+` FROM access_keys k WHERE k.id = ?`, id))
}

// KeyByHash finds a key that has not been revoked.
func (s *Store) KeyByHash(hash string) (AccessKey, error) {
	return scanKey(s.db.QueryRow(`SELECT `+keyColumns+` FROM access_keys k WHERE k.key_hash = ? AND k.revoked_at IS NULL`, hash))
}

// ListKeys returns the keys of an app, newest first, including revoked ones.
func (s *Store) ListKeys(appID int64) ([]AccessKey, error) {
	rows, err := s.db.Query(`SELECT `+keyColumns+` FROM access_keys k WHERE k.app_id = ? ORDER BY k.id DESC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []AccessKey{}
	for rows.Next() {
		k, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// RevokeKey stops a key from working. Its tokens go with it.
func (s *Store) RevokeKey(appID, keyID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE access_keys SET revoked_at = ? WHERE id = ? AND app_id = ? AND revoked_at IS NULL`, s.unix(), keyID, appID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(`DELETE FROM tokens WHERE key_id = ?`, keyID); err != nil {
		return err
	}
	return tx.Commit()
}

// touchEvery is how often a key's last use is written. A busy site sends many requests a minute, and
// the date only has to be roughly right.
const touchEvery = time.Minute

// TouchKey records that a key was just used, unless that was already recorded within the last minute.
func (s *Store) TouchKey(id int64) error {
	now := s.unix()
	_, err := s.db.Exec(`UPDATE access_keys SET last_used_at = ? WHERE id = ? AND (last_used_at IS NULL OR last_used_at <= ?)`,
		now, id, now-int64(touchEvery.Seconds()))
	return err
}
