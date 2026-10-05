package syncsvc

import (
	"bytes"
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

// modMachine holds real .mortar payloads: a profile is a list of Nexus mod ids, and a mod in fail never downloads.
type modMachine struct {
	*machine
	mods map[string][]int
	fail map[int]bool
	// note is the note on every entry.
	note string
}

func (m *modMachine) Export(_, id string) ([]byte, error) {
	p := profile.Profile{Name: "Main"}
	for _, n := range m.mods[id] {
		p.Entries = append(p.Entries, profile.Entry{Key: "n", Source: profile.Source{Kind: profile.KindNexus, ModID: n, FileID: n}, Note: m.note})
	}
	var buf bytes.Buffer
	_, err := share.Write(&buf, "stardew", p, "")
	return buf.Bytes(), err
}

func (m *modMachine) Apply(_ context.Context, _, id string, payload []byte) error {
	pv, err := share.ReadBytes(payload)
	if err != nil {
		return err
	}
	m.mods[id] = nil
	m.set(id, "")
	for _, r := range pv.Entries {
		if !m.fail[r.ModID] {
			m.mods[id] = append(m.mods[id], r.ModID)
		}
	}
	return nil
}

func (m *modMachine) edit(id string, mods ...int) {
	m.mods[id] = mods
	m.set(id, "")
}

func newModMachine(t *testing.T, folder, name string, clock *time.Time) (*Service, *modMachine) {
	t.Helper()
	m := &modMachine{machine: &machine{profiles: map[string]string{}, updated: map[string]time.Time{}, names: map[string]string{}, clock: clock}, mods: map[string][]int{}, fail: map[int]bool{}}
	s, err := New(Deps{Source: m, Folder: func() string { return folder }, Dir: t.TempDir(), Machine: name})
	if err != nil {
		t.Fatal(err)
	}
	return s, m
}

func payloadMods(t *testing.T, folder string) []int {
	t.Helper()
	raw, err := fsx.ReadFile(filepath.Join(folder, "stardew", "main.mortar"))
	if err != nil {
		t.Fatal(err)
	}
	pv, err := share.ReadBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int
	for _, r := range pv.Entries {
		ids = append(ids, r.ModID)
	}
	return ids
}

func TestTheirsApplyEchoIsNotPushedAndFailedDownloadsAreKept(t *testing.T) {
	folder := t.TempDir()
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a, ma := newModMachine(t, folder, "Desktop", &clock)
	b, mb := newModMachine(t, folder, "Laptop", &clock)
	mb.fail[3] = true
	ma.edit("main", 1, 2, 3)
	scan(t, a)
	offers := scan(t, b)
	if err := b.Resolve(context.Background(), "stardew", "main", offers[0].Revision, Theirs); err != nil {
		t.Fatal(err)
	}
	// The downloads finish after the apply, which bumps the profile; mod 3 failed.
	mb.edit("local-Main", 1, 2)
	scan(t, b)
	if got := payloadMods(t, folder); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("an echo rewrote the shared payload: %v", got)
	}
	if offers := scan(t, a); len(offers) != 0 {
		t.Fatalf("the original machine was offered its own change back: %+v", offers)
	}
	// The user removes mod 1: that is pushed, and the mod that never downloaded stays.
	mb.edit("local-Main", 2)
	scan(t, b)
	if got := payloadMods(t, folder); !slices.Equal(got, []int{2, 3}) {
		t.Fatalf("pushed mods = %v, want [2 3]", got)
	}
}

func TestAnEditWhileDownloadsFinishSyncsAndKeepsTheMissingMod(t *testing.T) {
	folder := t.TempDir()
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a, ma := newModMachine(t, folder, "Desktop", &clock)
	b, mb := newModMachine(t, folder, "Laptop", &clock)
	mb.fail[3] = true
	ma.edit("main", 1, 2, 3)
	scan(t, a)
	offers := scan(t, b)
	if err := b.Resolve(context.Background(), "stardew", "main", offers[0].Revision, Theirs); err != nil {
		t.Fatal(err)
	}
	mb.note = "my note"
	mb.edit("local-Main", 1, 2)
	scan(t, b)
	if got := payloadMods(t, folder); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("pushed mods = %v, want the missing one kept", got)
	}
	raw, err := fsx.ReadFile(filepath.Join(folder, "stardew", "main.mortar"))
	if err != nil {
		t.Fatal(err)
	}
	pv, err := share.ReadBytes(raw)
	if err != nil || pv.Entries[0].Note != "my note" {
		t.Fatalf("the note edit did not sync: %+v, %v", pv.Entries, err)
	}
}
