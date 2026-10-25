package store

// Totals are the counts the dashboard's overview shows.
type Totals struct {
	Apps           int
	Keys           int // not revoked
	Devices        int
	AppsWithoutKey int // apps that cannot send yet: no Firebase key
}

// Totals counts apps, active access keys and devices.
func (s *Store) Totals() (Totals, error) {
	var t Totals
	err := s.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM apps),
		(SELECT COUNT(*) FROM access_keys WHERE revoked_at IS NULL),
		(SELECT COUNT(*) FROM tokens),
		(SELECT COUNT(*) FROM apps a WHERE NOT EXISTS
			(SELECT 1 FROM app_providers p WHERE p.app_id = a.id AND p.kind = ?))`, ProviderFCM).
		Scan(&t.Apps, &t.Keys, &t.Devices, &t.AppsWithoutKey)
	return t, err
}
