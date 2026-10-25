package store

import (
	"errors"
	"testing"
	"time"
)

func TestDeliveriesAreRecordedUpdatedAndKeptPerKey(t *testing.T) {
	s := open(t)
	app := newApp(t, s, "sayan")
	mine := newKey(t, s, app.ID, "hash-mine")
	other := newKey(t, s, app.ID, "hash-other")

	if err := s.AddDelivery(Delivery{ID: "ntf_1", AppID: app.ID, KeyID: mine.ID, Status: "pending", RequestID: "req-1"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.DeliveryByID(mine.ID, "ntf_1")
	if err != nil || got.Status != "pending" || got.RequestID != "req-1" || got.CreatedAt.IsZero() {
		t.Fatalf("unexpected delivery %+v, %v", got, err)
	}

	if err := s.SetDeliveryStatus("ntf_1", "sent"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.DeliveryByID(mine.ID, "ntf_1"); got.Status != "sent" {
		t.Fatalf("status not updated: %+v", got)
	}
	if err := s.SetDeliveryStatus("nope", "sent"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a missing delivery should not be found, got %v", err)
	}

	if _, err := s.DeliveryByID(other.ID, "ntf_1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another key found a delivery that is not its own: %v", err)
	}
	if err := s.AddDelivery(Delivery{ID: "ntf_1", AppID: app.ID, KeyID: mine.ID, Status: "pending"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("a duplicate ID should conflict, got %v", err)
	}
}

func TestOldDeliveriesArePurgedAndGoWithTheirKey(t *testing.T) {
	s := open(t)
	app := newApp(t, s, "sayan")
	key := newKey(t, s, app.ID, "hash-one")
	base := time.Unix(1_700_000_000, 0)

	s.now = func() time.Time { return base.Add(-48 * time.Hour) }
	s.AddDelivery(Delivery{ID: "old", AppID: app.ID, KeyID: key.ID, Status: "sent"})
	s.now = func() time.Time { return base }
	s.AddDelivery(Delivery{ID: "new", AppID: app.ID, KeyID: key.ID, Status: "sent"})

	if n, err := s.PurgeDeliveries(base.Add(-24 * time.Hour)); err != nil || n != 1 {
		t.Fatalf("purged %d, %v; want 1", n, err)
	}
	if _, err := s.DeliveryByID(key.ID, "old"); !errors.Is(err, ErrNotFound) {
		t.Fatal("the old delivery was kept")
	}
	if _, err := s.DeliveryByID(key.ID, "new"); err != nil {
		t.Fatal("the new delivery was purged")
	}

	if err := s.RevokeKey(app.ID, key.ID); err != nil {
		t.Fatal(err)
	}
}
