package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bundles"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/gamesettings"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestTemplateRoundTripFromProfileToNewProfile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("LOCALAPPDATA", tmp)
	items, profiles := testenv.Stores(t)
	dataDir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(`{"Name":"One","UniqueID":"A.One","Version":"1.0.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir("stardew", "local-a", src); err != nil {
		t.Fatal(err)
	}
	from, err := profiles.Create("stardew", "Source")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddEntry("stardew", from.ID, "local-a", profile.Source{Kind: profile.KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.SetLaunchOptions("stardew", from.ID, "--no-gui"); err != nil {
		t.Fatal(err)
	}
	mode := "borderless"
	stored := map[string]gamesettings.Settings{from.ID: {WindowMode: &mode}}
	svc := NewService(Deps{
		Profiles: profiles,
		Bundles:  bundles.NewService(profiles, dataDir),
		GameSettings: func(_, id string) (gamesettings.Settings, error) {
			return stored[id], nil
		},
		SetGameSettings: func(_, id string, s gamesettings.Settings) error {
			stored[id] = s
			return nil
		},
	}, dataDir)

	saved, err := svc.SaveTemplateFromProfile("stardew", from.ID, " Starter ")
	if err != nil || saved.Name != "Starter" || len(saved.Bundle) != 1 || saved.LaunchOptions != "--no-gui" {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	if _, err := svc.SaveTemplateFromProfile("stardew", from.ID, "starter"); err != nil {
		t.Fatal(err)
	}
	if list, err := svc.Templates("stardew"); err != nil || len(list) != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}

	res, err := svc.NewProfileFromTemplate("stardew", "Starter", "Fresh")
	if err != nil || res.Added != 1 || len(res.Missing) != 0 || res.Profile.Name != "Fresh" {
		t.Fatalf("result = %+v, %v", res, err)
	}
	if opts, _ := profiles.LaunchOptions("stardew", res.Profile.ID); opts != "--no-gui" {
		t.Fatalf("launch options = %q", opts)
	}
	if got := stored[res.Profile.ID]; got.WindowMode == nil || *got.WindowMode != "borderless" {
		t.Fatalf("settings = %+v", got)
	}

	other, err := profiles.Create("stardew", "Other")
	if err != nil {
		t.Fatal(err)
	}
	pre, err := svc.PreviewApplyTemplate("stardew", "Starter", other.ID)
	if err != nil || len(pre.Add) != 1 || len(pre.AlreadyHave) != 0 || len(pre.Missing) != 0 ||
		len(pre.SettingsChanges) != 2 {
		t.Fatalf("preview = %+v, %v", pre, err)
	}
	applied, err := svc.ApplyTemplate("stardew", "Starter", other.ID)
	if err != nil || applied.Added != 1 || len(applied.Profile.Entries) != 1 {
		t.Fatalf("apply = %+v, %v", applied, err)
	}
	if opts, _ := profiles.LaunchOptions("stardew", other.ID); opts != "--no-gui" {
		t.Fatalf("launch options = %q", opts)
	}
	if again, err := svc.PreviewApplyTemplate("stardew", "Starter", other.ID); err != nil || len(again.Add) != 0 ||
		len(again.AlreadyHave) != 1 || len(again.SettingsChanges) != 0 {
		t.Fatalf("second preview = %+v, %v", again, err)
	}
	if evs, err := profiles.History("stardew", other.ID); err != nil || len(evs) != 2 {
		t.Fatalf("history = %+v, %v (want the baseline and the apply)", evs, err)
	}
	back, err := svc.UndoApplyTemplate("stardew", other.ID, applied.Undo)
	if err != nil || len(back.Entries) != 0 || back.LaunchOptions != "" {
		t.Fatalf("undo = %+v, %v", back, err)
	}
	if got := stored[other.ID]; got.WindowMode != nil {
		t.Fatalf("settings after undo = %+v", got)
	}

	if _, err := svc.NewProfileFromTemplate("stardew", "Nope", "X"); err == nil {
		t.Fatal("unknown template must fail")
	}
	if err := svc.DeleteTemplate("stardew", "STARTER"); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTemplate("stardew", "Starter"); err == nil {
		t.Fatal("second delete must fail")
	}
}

func TestRenameAndRestoreTemplate(t *testing.T) {
	svc := NewService(Deps{}, t.TempDir())
	for _, n := range []string{"A", "B"} {
		if err := svc.RestoreTemplate("stardew", Template{Name: n}); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.RenameTemplate("stardew", "A", "b"); err == nil {
		t.Fatal("rename onto another template must fail")
	}
	if err := svc.RenameTemplate("stardew", "A", "C"); err != nil {
		t.Fatal(err)
	}
	list, err := svc.Templates("stardew")
	if err != nil || len(list) != 2 || list[0].Name != "C" {
		t.Fatalf("list = %+v, %v", list, err)
	}
}

func TestTemplateCarriesDisabledLaunchStateAndSkipsLoaderMods(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("LOCALAPPDATA", tmp)
	items, profiles := testenv.Stores(t)
	dataDir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	mod := func(key, id string) {
		t.Helper()
		src := t.TempDir()
		manifest := `{"Name":"` + id + `","UniqueID":"` + id + `","Version":"1.0.0"}`
		if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := items.AddDir("stardew", key, src); err != nil {
			t.Fatal(err)
		}
	}
	mod("local-a", "A.One")
	mod("local-b", "A.Two")
	mod("smapi-1", "SMAPI.Console")
	from, err := profiles.Create("stardew", "Source")
	if err != nil {
		t.Fatal(err)
	}
	for key, src := range map[string]profile.Source{
		"local-a": {Kind: profile.KindLocal, Name: "a"}, "local-b": {Kind: profile.KindLocal, Name: "b"},
		"smapi-1": {Kind: profile.SourceSMAPI, Name: "smapi"},
	} {
		if _, err := profiles.AddEntry("stardew", from.ID, key, src); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := profiles.SetModEnabled("stardew", from.ID, "local-b", "A.Two", false); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.SetLaunchSettings("stardew", from.ID, "gamemoderun", "A=1"); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.SetOverride("stardew", from.ID, "defaultLaunchMethod", "direct"); err != nil {
		t.Fatal(err)
	}
	svc := NewService(Deps{
		Profiles: profiles, Bundles: bundles.NewService(profiles, dataDir),
		GameSettings:    func(string, string) (gamesettings.Settings, error) { return gamesettings.Settings{}, nil },
		SetGameSettings: func(string, string, gamesettings.Settings) error { return nil },
	}, dataDir)
	saved, err := svc.SaveTemplateFromProfile("stardew", from.ID, "T")
	if err != nil || len(saved.Bundle) != 2 || len(saved.Disabled) != 1 {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	keys, err := svc.ReferencedStoreKeys()
	if err != nil || len(keys["stardew"]) != 2 {
		t.Fatalf("keys = %v, %v", keys, err)
	}
	res, err := svc.NewProfileFromTemplate("stardew", "T", "Fresh")
	if err != nil || res.Added != 2 {
		t.Fatalf("new = %+v, %v", res, err)
	}
	got := res.Profile
	if got.LaunchPrefix != "gamemoderun" || got.LaunchEnv != "A=1" || got.Overrides["defaultLaunchMethod"] != "direct" {
		t.Fatalf("launch state = %+v", got)
	}
	for _, e := range got.Entries {
		if e.Key == "local-b" && (len(e.Disabled) != 1 || e.Disabled[0] != "A.Two") {
			t.Fatalf("local-b disabled = %v", e.Disabled)
		}
		if e.Key == "local-a" && len(e.Disabled) != 0 {
			t.Fatalf("local-a disabled = %v", e.Disabled)
		}
	}
}

func TestTemplateFileFormats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stardew.json")
	if err := os.WriteFile(path, []byte(`[{"name":"Old","game":"stardew"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	list, err := readList(path)
	if err != nil || len(list) != 1 || list[0].Name != "Old" {
		t.Fatalf("bare array = %v, %v", list, err)
	}
	newer := `{"formatVersion":99,"templates":[{"name":"New","game":"stardew"}]}`
	if err := os.WriteFile(path, []byte(newer), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readList(path); err == nil {
		t.Error("a file from a newer Mortar must be refused")
	}
	if err := datadir.WriteVersioned(path, templatesFile{FormatVersion: datadir.FormatVersion, Templates: list}); err == nil {
		t.Error("a newer file must not be overwritten")
	}
}
