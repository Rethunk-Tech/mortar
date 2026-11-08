package profile

import (
	"testing"
	"time"
)

func TestParseSinceLayoutsAndLocalTimes(t *testing.T) {
	t.Parallel()
	utc := time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)
	offset := time.Date(2026, 3, 2, 9, 0, 0, 0, time.FixedZone("", 2*60*60))
	cases := []struct {
		name string
		in   string
		want time.Time
	}{
		{"empty", "", time.Time{}},
		{"blanks only", " \t ", time.Time{}},
		{"rfc3339", "2026-03-02T08:00:00Z", utc},
		{"trimmed", " 2026-03-02T08:00:00Z ", utc},
		{"rfc3339 offset", "2026-03-02T09:00:00+02:00", offset},
		// A time with no zone is read as UTC.
		{"no zone seconds", "2026-03-02 15:04:05", time.Date(2026, 3, 2, 15, 4, 5, 0, time.UTC)},
		{"no zone minutes", "2026-03-02 15:04", time.Date(2026, 3, 2, 15, 4, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got, err := ParseSince(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !got.Equal(c.want) {
			t.Fatalf("%s: ParseSince(%q) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
	if _, err := ParseSince("2 days ago"); err == nil {
		t.Fatal("a time nothing parses was accepted")
	}
}

// EventDiffs answers every listed event by its id and fails the whole call for one that is not in the history.
func TestEventDiffsMapsByEventID(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	svc, _ := e.withSettings(t)
	p := mustCreate(t, e, "Farm")
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	events, err := svc.History("stardew", p.ID)
	if err != nil || len(events) < 2 {
		t.Fatalf("history = %d events, %v", len(events), err)
	}
	ids := []string{events[len(events)-1].ID, events[len(events)-2].ID}
	got, err := svc.EventDiffs("stardew", p.ID, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(ids) {
		t.Fatalf("EventDiffs = %d entries, want %d", len(got), len(ids))
	}
	for _, id := range ids {
		d, ok := got[id]
		if !ok || d.B != id {
			t.Fatalf("diff for %s = %+v, %v", id, d, ok)
		}
	}
	if _, err := svc.EventDiffs("stardew", p.ID, []string{"no-such-event"}); err == nil {
		t.Fatal("an unknown event id was accepted")
	}
}
