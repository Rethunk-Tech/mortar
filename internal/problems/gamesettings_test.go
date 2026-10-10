package problems

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

var modSettings = []components.RequiredSetting{
	{Path: "options", Key: "ModsEnabled", Value: "1", Message: "custom content is off"},
	{Path: "options", Key: "ScriptMods", Value: "1", Message: "script mods are off"},
}

func TestSettingFailuresReportOnlyAKeyWithAnotherValue(t *testing.T) {
	got := settingFailures("[Options]\n; a comment\nmodsenabled = 0\nOther=1\n", modSettings)
	if len(got) != 1 || got[0].Kind != KindGameSetting || got[0].Plugin != "ModsEnabled" || got[0].Message != "custom content is off" {
		t.Fatalf("got %+v", got)
	}
	if got := settingFailures("modsenabled=1\nscriptmods = 1\n", modSettings); len(got) != 0 {
		t.Fatalf("switched on settings are not findings: %+v", got)
	}
	if got := settingFailures("", modSettings); len(got) != 0 {
		t.Fatalf("keys the file lacks keep the game's default: %+v", got)
	}
}

func TestCheckingTheSettingsFileNeverChangesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Options.ini")
	body := "[Options]\r\nModsEnabled = 0\r\nScriptMods = 0\r\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(path)
	if got := settingFailures(body, modSettings); len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	after, _ := fsx.ReadFile(path)
	stat, _ := os.Stat(path)
	if string(after) != body || !stat.ModTime().Equal(before.ModTime()) {
		t.Fatal("the file must be untouched")
	}
}

func TestSettingRowsFollowTheProfilesGameSettingsMode(t *testing.T) {
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	const id = "options-game"
	m.Games = append(slices.Clone(m.Games), components.GameInfo{
		ID: id, Name: "Options Game", Enabled: true, Marker: "G.dll", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods", MaxDepth: map[string]int{"pkg": 1}}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "1"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
		Paths: map[string]components.PathTemplate{
			"mods":    {Windows: "{documents}/G/Mods", Linux: "{documents}/G/Mods", Darwin: "{documents}/G/Mods"},
			"options": {Windows: "{documents}/G/Options.ini", Linux: "{documents}/G/Options.ini", Darwin: "{documents}/G/Options.ini"},
		},
		RequiredSettings: []components.RequiredSetting{{Path: "options", Key: "ModsDisabled", Value: "0", Message: "mods off"}},
	})
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	c := components.NewClient(nil)
	c.SetManifest(m)
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })

	datadirtest.Use(t, t.TempDir())
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "G.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders[id] = folder }); err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, id, "A")
	home := t.TempDir()
	opts := filepath.Join(home, "Documents", "G", "Options.ini")
	if err := os.MkdirAll(filepath.Dir(opts), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(opts, []byte("[o]\nModsDisabled = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewService(home, set, profiles, nil)

	if got := s.gameSettingFailures(id, p.ID); len(got) != 0 {
		t.Fatalf("edit mode fixes it at launch, so no row: %+v", got)
	}
	if _, err := profiles.SetOverride(id, p.ID, "gameSettingsMode", settings.GameSettingsWarn); err != nil {
		t.Fatal(err)
	}
	if got := s.gameSettingFailures(id, p.ID); len(got) != 1 {
		t.Fatalf("a profile set to warn must get the row: %+v", got)
	}
	if _, err := set.Update(func(v *settings.Settings) {
		gp := v.GamePrefs(id)
		gp.GameSettingsMode = settings.GameSettingsWarn
		v.Games = map[string]*settings.GameSettings{id: &gp}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.SetOverride(id, p.ID, "gameSettingsMode", settings.GameSettingsEdit); err != nil {
		t.Fatal(err)
	}
	if got := s.gameSettingFailures(id, p.ID); len(got) != 0 {
		t.Fatalf("a profile set to edit on a warn game needs no row: %+v", got)
	}
}

func TestPerFileEntriesOfOneItemAreOneStoreItemToProblems(t *testing.T) {
	const item = "nexus-123-456-1700000000"
	keys := []string{item + "#A/a.package", item + "#A/b.package", item + "#B/c.ts4script"}
	if _, page, ok := nexusFile(keys[2]); !ok || page != 456 {
		t.Fatalf("a per-file key must resolve to its item's Nexus file: %d %v", page, ok)
	}
	var mods []framework.Mod
	var updates []Update
	for _, k := range keys {
		mods = append(mods, framework.Mod{Key: k, Name: k})
		updates = append(updates, Update{Key: k, Version: "2.0"})
	}
	rows := damagedRows(mods, map[string]store.Damage{item: {Missing: []string{"x"}}})
	if len(rows) != 1 {
		t.Fatalf("a damaged item must be one row, got %+v (no false missing-from-store rows)", rows)
	}
	if got := oncePerItem(updates); len(got) != 1 {
		t.Fatalf("one download updates all three files once, got %d", len(got))
	}
	if !coveredBy(updates[:1], keys[1], "2.0") {
		t.Fatal("a sibling file's update covers the same item")
	}
}
