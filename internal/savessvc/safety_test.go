package savessvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/queue"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestRecordRunStoresEnabledMods(t *testing.T) {
	s := NewStore(t.TempDir())
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	mods := []PlayedMod{{UniqueID: "A.Mod", Name: "Alpha", Version: "1.2", Key: "nexus-1-2", SourceKind: "nexus", ModID: 1, FileID: 2}}
	if err := s.RecordRun("stardew", "Farm_1", "p1", at, mods); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.Get("stardew", "Farm_1")
	if err != nil || !ok || got.ProfileID != "p1" || len(got.Mods) != 1 || got.Mods[0].Key != "nexus-1-2" {
		t.Fatalf("got %#v %v %v", got, ok, err)
	}
}

func TestNotePlayedRecordsProfileMods(t *testing.T) {
	e := newSaveEnv(t)
	p, err := e.profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	e.item(t, "local-a", map[string]string{"manifest.json": `{"Name":"Alpha","Author":"me","Version":"1.0.0","UniqueID":"A.Mod"}`})
	if _, err := e.profiles.AddEntry("stardew", p.ID, "local-a", profile.Source{Kind: profile.KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	svc := &Service{last: NewStore(t.TempDir()), profiles: e.profiles}
	svc.NotePlayed("stardew", p.ID, "Farm_1")
	rec, ok, err := svc.last.Get("stardew", "Farm_1")
	if err != nil || !ok || rec.ProfileID != p.ID || len(rec.Mods) != 1 || rec.Mods[0].UniqueID != "A.Mod" {
		t.Fatalf("rec %#v %v %v", rec, ok, err)
	}
}

func TestMissingFromSortsContentPacksFirstAndFlagsDisabled(t *testing.T) {
	recorded := []PlayedMod{
		{UniqueID: "Code.Mod", Name: "Code", Version: "1"},
		{UniqueID: "Pack.B", Name: "Pack B", Version: "1", ContentPackFor: "Code.Mod"},
		{UniqueID: "Pack.A", Name: "Pack A", Version: "1", ContentPackFor: "Code.Mod"},
		{UniqueID: "Have.On", Name: "On", Version: "1"},
	}
	present := map[string]bool{"code.mod": true, "have.on": true}
	enabled := map[string]bool{"have.on": true}
	got := MissingFrom(recorded, present, enabled)
	if len(got) != 3 || got[0].UniqueID != "Pack.A" || got[1].UniqueID != "Pack.B" || got[2].UniqueID != "Code.Mod" {
		t.Fatalf("order %#v", got)
	}
	if !got[2].Disabled || got[0].Disabled {
		t.Fatalf("disabled flags %#v", got)
	}
}

func TestCheckUsesRecordedList(t *testing.T) {
	last := NewStore(t.TempDir())
	if err := last.RecordRun("stardew", "Farm_1", "gone", time.Now(), []PlayedMod{
		{UniqueID: "A.Mod", Name: "Alpha", ContentPackFor: "SMAPI"},
	}); err != nil {
		t.Fatal(err)
	}
	svc := &Service{last: last}
	ch, err := svc.Check(context.Background(), "stardew", "Farm_1", "")
	if err != nil || ch.ContentMissing != 1 || len(ch.Missing) != 1 || ch.Missing[0].Name != "Alpha" {
		t.Fatalf("check %#v %v", ch, err)
	}
}

type saveEnv struct {
	profiles *profile.Store
	items    *store.Store
}

func newSaveEnv(t *testing.T) saveEnv {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_DATA_HOME", base)
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("LOCALAPPDATA", base)
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ps, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	return saveEnv{ps, items}
}

func (e saveEnv) item(t *testing.T, key string, files map[string]string) {
	t.Helper()
	src := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(src, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.items.AddDir("stardew", key, src); err != nil {
		t.Fatal(err)
	}
}

func TestFromSaveReusesStoreAndQueuesTheRest(t *testing.T) {
	e := newSaveEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": `{"Name":"Alpha","Author":"me","Version":"1.0.0","UniqueID":"A.Mod"}`})
	last := NewStore(t.TempDir())
	if err := last.RecordRun("stardew", "Sunny_1", "old", time.Now(), []PlayedMod{
		{UniqueID: "A.Mod", Name: "Alpha", Version: "1.0.0", Key: "local-a", SourceKind: profile.KindLocal},
		{UniqueID: "B.Mod", Name: "Beta", Version: "2.0.0", Key: "nexus-9-8", SourceKind: profile.KindNexus, ModID: 9, FileID: 8},
	}); err != nil {
		t.Fatal(err)
	}
	var queued []queue.Request
	svc := &Service{
		profiles: e.profiles,
		last:     last,
		Enqueue: func(reqs []queue.Request) ([]queue.Item, error) {
			queued = append(queued, reqs...)
			return nil, nil
		},
	}
	got, err := svc.FromSave("stardew", "Sunny_1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Profile.Name != "Sunny_1" && got.Profile.Name != "Sunny" {
		if got.Profile.Name == "" {
			t.Fatalf("empty name: %#v", got)
		}
	}
	if len(got.Added) != 1 || got.Added[0] != "local-a" {
		t.Fatalf("added %#v", got.Added)
	}
	if len(got.Queued) != 1 || got.Queued[0] != "B.Mod" || len(queued) != 1 || queued[0].ModID != 9 {
		t.Fatalf("queued %#v %#v", got.Queued, queued)
	}
	mods, err := e.profiles.Mods("stardew", got.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range mods {
		if m.UniqueID == "A.Mod" {
			found = true
		}
	}
	if !found {
		t.Fatalf("profile mods %#v", mods)
	}
}
