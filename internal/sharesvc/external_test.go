package sharesvc

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/migrate"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

func TestExternalLocalModsKeepProfileStateForImport(t *testing.T) {
	mods := externalLocalMods([]migrate.ModPreview{
		{ID: "smapi:Example.Mod", Name: "Example Mod", Version: "1.2.3", Enabled: false, SourcePath: "/mods/example"},
	})
	if len(mods) != 1 {
		t.Fatalf("mods = %#v", mods)
	}
	if mods[0].Site != SiteLocal || mods[0].State != StateDownload || mods[0].Key != "external:0" || mods[0].Enabled {
		t.Fatalf("mod = %#v", mods[0])
	}
	if len(mods[0].IDs) != 1 || mods[0].IDs[0] != "smapi:Example.Mod" {
		t.Fatalf("unique IDs = %#v", mods[0].IDs)
	}
}

func TestExternalImportWritesEachModsOwnConfigWithinTheCap(t *testing.T) {
	s, _ := newService(t, true)
	src := t.TempDir()
	for _, id := range []string{"A", "B"} {
		dir := filepath.Join(src, id)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		manifest := `{"Name":"` + id + `","Author":"a","Version":"1.0.0","UniqueID":"Me.` + id + `","EntryDll":"m.dll"}`
		if err := fsx.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(filepath.Join(dir, "config.json"), []byte(`"folder"`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	big := []byte(`"` + strings.Repeat("x", share.MaxConfigBytes) + `"`)
	external := migrate.ProfilePreview{Name: "Ext", Mods: []migrate.ModPreview{
		{ID: "smapi:Me.A", SourcePath: filepath.Join(src, "A"), Enabled: true, Config: []byte(`"profile"`)},
		{ID: "smapi:Me.B", SourcePath: filepath.Join(src, "B"), Enabled: true, Config: big},
	}}
	pv, err := s.PreviewExternal(context.Background(), "stardew", external, "")
	if err != nil {
		t.Fatal(err)
	}
	if pv.Settings != 1 || !slices.Equal(pv.SkippedSettings, []string{"Me.B/config.json"}) {
		t.Fatalf("settings = %d, skipped = %v", pv.Settings, pv.SkippedSettings)
	}
	res, err := s.Import(context.Background(), "stardew", pv.Session, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err := s.d.Profiles.History("stardew", res.Profile.ID)
	if err != nil || len(events) == 0 {
		t.Fatalf("history: %v %v", events, err)
	}
	if res.Profile.LastChange == "" || res.Profile.LastChange != events[0].ID {
		t.Fatalf("LastChange %q, newest event %q", res.Profile.LastChange, events[0].ID)
	}
	dir, err := s.d.Profiles.ModsDir("stardew", res.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"Me.A": `"profile"`, "Me.B": `"folder"`}
	for _, e := range res.Profile.Entries {
		for _, m := range e.Mods {
			got, err := fsx.ReadFile(filepath.Join(dir, e.Key, filepath.FromSlash(m.Folder), "config.json"))
			if err != nil || string(got) != want[m.ID.Local()] {
				t.Fatalf("%s config = %q, %v", m.ID, got, err)
			}
			delete(want, m.ID.Local())
		}
	}
	if len(want) != 0 {
		t.Fatalf("not imported: %v", want)
	}
}
