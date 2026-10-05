package syncsvc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// machine is one machine's profiles: id to content.
type machine struct {
	profiles map[string]string
	updated  map[string]time.Time
	names    map[string]string
	clock    *time.Time
}

func (m *machine) set(id, content string) {
	*m.clock = m.clock.Add(time.Minute)
	m.profiles[id], m.updated[id], m.names[id] = content, *m.clock, "Main"
}

func (m *machine) Games() []string { return []string{"stardew"} }
func (m *machine) Profiles(string) ([]Ref, error) {
	var out []Ref
	for id := range m.profiles {
		out = append(out, Ref{Game: "stardew", ID: id, Name: m.names[id], Updated: m.updated[id]})
	}
	return out, nil
}
func (m *machine) Export(_, id string) ([]byte, error) { return []byte(m.profiles[id]), nil }
func (m *machine) Create(_, name string) (string, error) {
	id := "local-" + name
	m.set(id, "")
	return id, nil
}

func (m *machine) Preview(_ context.Context, _, id string, payload []byte) (Diff, error) {
	return Diff{Add: []string{string(payload)}, Remove: []string{m.profiles[id]}}, nil
}

func (m *machine) Apply(_ context.Context, _, id string, payload []byte) error {
	m.set(id, string(payload))
	return nil
}

func newMachine(t *testing.T, folder, name string, clock *time.Time) (*Service, *machine) {
	t.Helper()
	m := &machine{profiles: map[string]string{}, updated: map[string]time.Time{}, names: map[string]string{}, clock: clock}
	s, err := New(Deps{Source: m, Folder: func() string { return folder }, Dir: t.TempDir(), Machine: name})
	if err != nil {
		t.Fatal(err)
	}
	return s, m
}

func scan(t *testing.T, s *Service) []Offer {
	t.Helper()
	offers, err := s.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return offers
}

func shared1(t *testing.T, folder string) string {
	t.Helper()
	b, err := fsx.ReadFile(filepath.Join(folder, "stardew", "main.mortar"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTwoMachinesOfferApplyAndConflict(t *testing.T) {
	ctx := context.Background()
	folder := t.TempDir()
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a, ma := newMachine(t, folder, "Desktop", &clock)
	b, mb := newMachine(t, folder, "Laptop", &clock)

	ma.set("main", "v1")
	if offers := scan(t, a); len(offers) != 0 || shared1(t, folder) != "v1" {
		t.Fatalf("first write: offers %v, payload %q", offers, shared1(t, folder))
	}
	offers := scan(t, b)
	if len(offers) != 1 || !offers[0].New || offers[0].Machine != "Desktop" {
		t.Fatalf("new profile offer = %+v", offers)
	}
	if err := b.Resolve(ctx, "stardew", "main", offers[0].Revision, Theirs); err != nil {
		t.Fatal(err)
	}
	if mb.profiles["local-Main"] != "v1" || len(scan(t, b)) != 0 {
		t.Fatalf("after apply: %v", mb.profiles)
	}

	ma.set("main", "v2")
	scan(t, a)
	offers = scan(t, b)
	if len(offers) != 1 || offers[0].Conflict || offers[0].Profile != "local-Main" {
		t.Fatalf("newer revision offer = %+v", offers)
	}
	if d, err := b.Diff(ctx, "stardew", "main"); err != nil || d.Add[0] != "v2" || d.Remove[0] != "v1" {
		t.Fatalf("diff = %+v, %v", d, err)
	}
	if err := b.Resolve(ctx, "stardew", "main", offers[0].Revision, Theirs); err != nil || mb.profiles["local-Main"] != "v2" {
		t.Fatalf("apply v2: %v %v", mb.profiles, err)
	}

	ma.set("main", "v3")
	mb.set("local-Main", "v3b")
	scan(t, a)
	offers = scan(t, b)
	if len(offers) != 1 || !offers[0].Conflict {
		t.Fatalf("both changed must conflict: %+v", offers)
	}
	if mb.profiles["local-Main"] != "v3b" || shared1(t, folder) != "v3" {
		t.Fatal("a conflict must not merge or overwrite")
	}
	if err := b.Resolve(ctx, "stardew", "main", offers[0].Revision, Mine); err != nil {
		t.Fatal(err)
	}
	if shared1(t, folder) != "v3b" {
		t.Fatalf("keep mine must write it out, got %q", shared1(t, folder))
	}
	offers = scan(t, a)
	if len(offers) != 1 || offers[0].Conflict || !strings.Contains(offers[0].Machine, "Laptop") {
		t.Fatalf("the other side now sees a plain offer: %+v", offers)
	}
}

func TestSyncIsOffWithoutAFolder(t *testing.T) {
	clock := time.Now()
	s, m := newMachine(t, "", "Desktop", &clock)
	m.set("main", "v1")
	if offers := scan(t, s); len(offers) != 0 {
		t.Fatalf("offers = %v", offers)
	}
}

func TestResolveRefusesARevisionOtherThanTheOneShown(t *testing.T) {
	ctx := context.Background()
	folder := t.TempDir()
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a, ma := newMachine(t, folder, "Desktop", &clock)
	b, mb := newMachine(t, folder, "Laptop", &clock)
	ma.set("main", "v1")
	scan(t, a)
	shown := scan(t, b)[0].Revision
	ma.set("main", "v2")
	scan(t, a)
	if err := b.Resolve(ctx, "stardew", "main", shown, Theirs); err == nil {
		t.Fatal("a stale revision must not apply")
	}
	if len(mb.profiles) != 0 {
		t.Fatalf("nothing may be applied: %v", mb.profiles)
	}
	if err := b.Resolve(ctx, "stardew", "main", scan(t, b)[0].Revision, Theirs); err != nil {
		t.Fatal(err)
	}
}
