package store

import "time"

// Delivery is what became of one notification sent through the native API. It keeps no message text.
type Delivery struct {
	ID        string
	AppID     int64
	KeyID     int64
	Status    string // pending, sent, failed or unregistered
	RequestID string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AddDelivery records a delivery that has just started.
func (s *Store) AddDelivery(d Delivery) error {
	now := s.unix()
	_, err := s.db.Exec(`INSERT INTO deliveries (id, app_id, key_id, status, request_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, d.ID, d.AppID, d.KeyID, d.Status, d.RequestID, now, now)
	return wrap(err)
}

// SetDeliveryStatus records how a delivery ended.
func (s *Store) SetDeliveryStatus(id, status string) error {
	return s.exec1(`UPDATE deliveries SET status = ?, updated_at = ? WHERE id = ?`, status, s.unix(), id)
}

// DeliveryByID finds a delivery started by this access key. Another key's deliveries are not found.
func (s *Store) DeliveryByID(keyID int64, id string) (Delivery, error) {
	var d Delivery
	var created, updated int64
	err := s.db.QueryRow(`SELECT id, app_id, key_id, status, request_id, created_at, updated_at
		FROM deliveries WHERE id = ? AND key_id = ?`, id, keyID).
		Scan(&d.ID, &d.AppID, &d.KeyID, &d.Status, &d.RequestID, &created, &updated)
	if err != nil {
		return Delivery{}, wrap(err)
	}
	d.CreatedAt, d.UpdatedAt = fromUnix(created), fromUnix(updated)
	return d, nil
}

// PurgeDeliveries deletes deliveries older than [olderThan] and returns how many went.
func (s *Store) PurgeDeliveries(olderThan time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM deliveries WHERE created_at < ?`, olderThan.Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
