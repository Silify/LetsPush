package store

import "time"

// Admin is a person who can sign in to the web interface.
type Admin struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// CountAdmins returns how many admins exist.
func (s *Store) CountAdmins() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n)
	return n, err
}

// CreateAdmin adds an admin.
func (s *Store) CreateAdmin(email, passwordHash string) (Admin, error) {
	now := s.unix()
	res, err := s.db.Exec(`INSERT INTO admins (email, password_hash, created_at) VALUES (?, ?, ?)`, email, passwordHash, now)
	if err != nil {
		return Admin{}, wrap(err)
	}
	id, _ := res.LastInsertId()
	return Admin{ID: id, Email: email, PasswordHash: passwordHash, CreatedAt: fromUnix(now)}, nil
}

// AdminByEmail finds an admin by email.
func (s *Store) AdminByEmail(email string) (Admin, error) {
	return s.scanAdmin(s.db.QueryRow(`SELECT id, email, password_hash, created_at FROM admins WHERE email = ?`, email))
}

// AdminByID finds an admin by id.
func (s *Store) AdminByID(id int64) (Admin, error) {
	return s.scanAdmin(s.db.QueryRow(`SELECT id, email, password_hash, created_at FROM admins WHERE id = ?`, id))
}

// SetAdminPassword replaces an admin's password hash.
func (s *Store) SetAdminPassword(id int64, passwordHash string) error {
	res, err := s.db.Exec(`UPDATE admins SET password_hash = ? WHERE id = ?`, passwordHash, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(dest ...any) error }

func (s *Store) scanAdmin(row rowScanner) (Admin, error) {
	var a Admin
	var created int64
	if err := row.Scan(&a.ID, &a.Email, &a.PasswordHash, &created); err != nil {
		return Admin{}, wrap(err)
	}
	a.CreatedAt = fromUnix(created)
	return a, nil
}
