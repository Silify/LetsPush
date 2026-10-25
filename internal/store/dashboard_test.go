package store

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestLogsCanBeFilteredPagedAndNameTheirApp(t *testing.T) {
	s := open(t)
	one := newApp(t, s, "one")
	two := newApp(t, s, "two")
	for i := 0; i < 5; i++ {
		s.AddLog(one.ID, "push", "sent to a"+strconv.Itoa(i))
	}
	s.AddLog(one.ID, "error", "boom")
	s.AddLog(two.ID, "push", "sent to b")
	s.AddLog(0, "app", "no app")

	all, _ := s.ListLogs(LogFilter{Limit: 50})
	if len(all) != 8 || all[0].Info != "no app" || all[0].AppName != "" || all[1].AppName != "two" {
		t.Fatalf("unexpected log %+v", all)
	}
	if got, _ := s.ListLogs(LogFilter{AppID: one.ID, Limit: 50}); len(got) != 6 {
		t.Fatalf("app filter: %d entries", len(got))
	}
	if got, _ := s.ListLogs(LogFilter{AppID: one.ID, Action: "error", Limit: 50}); len(got) != 1 || got[0].Info != "boom" {
		t.Fatalf("action filter: %+v", got)
	}

	first, _ := s.ListLogs(LogFilter{AppID: one.ID, Limit: 4})
	older, _ := s.ListLogs(LogFilter{AppID: one.ID, Limit: 4, BeforeID: first[len(first)-1].ID})
	if len(first) != 4 || len(older) != 2 || older[0].ID >= first[3].ID {
		t.Fatalf("paging: %d then %d", len(first), len(older))
	}
}

func TestLogsAreCountedByActionTextAndTime(t *testing.T) {
	s := open(t)
	app := newApp(t, s, "one")
	other := newApp(t, s, "two")
	base := time.Unix(1_700_000_000, 0)

	s.now = func() time.Time { return base.Add(-48 * time.Hour) }
	s.AddLog(app.ID, "push", "sent to old")
	s.now = func() time.Time { return base }
	s.AddLog(app.ID, "push", "sent to a")
	s.AddLog(app.ID, "push", "sent to b")
	s.AddLog(app.ID, "push", "x is no longer registered; removed")
	s.AddLog(app.ID, "error", "boom")
	s.AddLog(other.ID, "push", "sent to c")

	since := base.Add(-24 * time.Hour)
	if n, _ := s.CountLogs(app.ID, "push", "sent to ", since); n != 2 {
		t.Fatalf("sent in the last day for the app: %d", n)
	}
	if n, _ := s.CountLogs(0, "push", "sent to ", since); n != 3 {
		t.Fatalf("sent in the last day overall: %d", n)
	}
	if n, _ := s.CountLogs(app.ID, "error", "", since); n != 1 {
		t.Fatalf("errors: %d", n)
	}
}

func TestTotals(t *testing.T) {
	s := open(t)
	keyed := newApp(t, s, "keyed")
	s.CreateApp("bare")
	k := newKey(t, s, keyed.ID, "hash-one")
	revoked := newKey(t, s, keyed.ID, "hash-two")
	s.RevokeKey(keyed.ID, revoked.ID)
	s.UpsertToken(Token{AppID: keyed.ID, KeyID: k.ID, Token: "T1", Device: "android", Channel: "default"})
	s.UpsertToken(Token{AppID: keyed.ID, KeyID: k.ID, Token: "T2", Device: "web", Channel: "default"})

	got, err := s.Totals()
	if err != nil || got != (Totals{Apps: 2, Keys: 1, Devices: 2, AppsWithoutKey: 1}) {
		t.Fatalf("unexpected totals %+v, %v", got, err)
	}
}

func TestDevicesAreListedWithTheirKeyPagedAndRemoved(t *testing.T) {
	s := open(t)
	app := newApp(t, s, "one")
	other := newApp(t, s, "two")
	k := newKey(t, s, app.ID, "hash-one")
	otherKey := newKey(t, s, other.ID, "hash-two")
	for i := 0; i < 5; i++ {
		s.UpsertToken(Token{AppID: app.ID, KeyID: k.ID, Token: "T" + strconv.Itoa(i), Device: "android", Channel: "default"})
	}
	s.UpsertToken(Token{AppID: other.ID, KeyID: otherKey.ID, Token: "X", Device: "ios", Channel: "default"})

	first, err := s.ListDevices(app.ID, 0, 3)
	if err != nil || len(first) != 3 || first[0].Token != "T4" || first[0].KeyPrefix != "hash" || first[0].Platform != "android" {
		t.Fatalf("unexpected first page %+v, %v", first, err)
	}
	rest, _ := s.ListDevices(app.ID, first[2].ID, 3)
	if len(rest) != 2 || rest[1].Token != "T0" {
		t.Fatalf("unexpected second page %+v", rest)
	}

	if err := s.DeleteDevice(other.ID, first[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another app's device was removed: %v", err)
	}
	if err := s.DeleteDevice(app.ID, first[0].ID); err != nil {
		t.Fatal(err)
	}
	if left, _ := s.ListDevices(app.ID, 0, 10); len(left) != 4 {
		t.Fatalf("%d devices left", len(left))
	}
}
