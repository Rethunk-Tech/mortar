package profile

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestExtraFolderListsAndInstallsOnlyItsOwnFolders(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	st, err := settings.OpenIn(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(e.Store, t.TempDir(), st)
	p := mustCreate(t, e, "P")
	if got, err := svc.ExtraFolderMods("stardew"); err != nil || len(got.Mods) != 0 {
		t.Fatalf("unset folder = %+v, %v", got, err)
	}
	extra := t.TempDir()
	writeFile(t, extra, "A/manifest.json", manifestJSON("me.a"))
	writeFile(t, extra, "Pack/B/manifest.json", manifestJSON("me.b"))
	second := t.TempDir()
	writeFile(t, second, "C/manifest.json", manifestJSON("me.c"))
	setGamePref(t, st, "extraModsFolder", extra+string(filepath.ListSeparator)+second)
	got, err := svc.ExtraFolderMods("stardew")
	if err != nil || len(got.Mods) != 3 {
		t.Fatalf("listed = %+v, %v", got, err)
	}
	folder := ""
	for _, m := range got.Mods {
		if m.ID == "smapi:me.b" {
			folder = m.Folder
		}
	}
	if folder != filepath.Join(extra, "Pack") {
		t.Fatalf("folder = %q", folder)
	}
	res, err := svc.InstallExtraFolderMod(t.Context(), "stardew", p.ID, folder)
	if err != nil || len(res.Added) != 1 {
		t.Fatalf("install = %+v, %v", res, err)
	}
	if res, err := svc.InstallExtraFolderMod(t.Context(), "stardew", p.ID, filepath.Join(second, "C")); err != nil || len(res.Added) != 1 {
		t.Fatalf("install from second folder = %+v, %v", res, err)
	}
	for _, bad := range []string{t.TempDir(), filepath.Join(extra, "Pack", "B"), extra} {
		if _, err := svc.InstallExtraFolderMod(t.Context(), "stardew", p.ID, bad); err == nil {
			t.Errorf("installed %s from outside the extra folder", bad)
		}
	}
}

func setGamePref(t *testing.T, st *settings.Store, key, value string) {
	t.Helper()
	setPrefOf(t, st, "stardew", key, value)
}

func setPrefOf(t *testing.T, st *settings.Store, game, key, value string) {
	t.Helper()
	var applyErr error
	if _, err := st.Update(func(s *settings.Settings) { applyErr = settings.ApplyKeyGame(s, key, value, game) }); err != nil {
		t.Fatal(err)
	}
	if applyErr != nil {
		t.Fatal(applyErr)
	}
}

func TestExtraFolderTakesAPackageGamesModsAndABuildOutputFolder(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	st, err := settings.OpenIn(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(e.Store, t.TempDir(), st)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	// A folder of mods as r2modman lays them out, one with a Thunderstore manifest and no UniqueID, one a bare plugin.
	mods := t.TempDir()
	writeFile(t, mods, "Wallet-OuterWildsItems/manifest.json", `{"name":"OuterWildsItems","version_number":"1.2.0","author":"Wallet"}`)
	writeFile(t, mods, "Wallet-OuterWildsItems/OuterWildsItems.dll", "MZ")
	writeFile(t, mods, "Bare/Bare.dll", "MZ")
	// A C# project's Release folder: the plugin and its symbols, no manifest.
	build := t.TempDir()
	writeFile(t, build, "MyPlugin.dll", "MZ")
	writeFile(t, build, "MyPlugin.pdb", "pdb")
	setPrefOf(t, st, "lethal-company", "extraModsFolder", mods+string(filepath.ListSeparator)+build)
	got, err := svc.ExtraFolderMods("lethal-company")
	if err != nil || len(got.Mods) != 3 {
		t.Fatalf("listed = %+v, %v", got, err)
	}
	names := map[string]string{}
	for _, m := range got.Mods {
		if m.Status != outcomeImported {
			t.Errorf("%s not offered: %+v", m.Name, m)
		}
		names[m.Name] = m.Folder
	}
	if names["OuterWildsItems"] != filepath.Join(mods, "Wallet-OuterWildsItems") || names["Bare"] != filepath.Join(mods, "Bare") || names["MyPlugin"] != build {
		t.Fatalf("names and folders = %v", names)
	}
	for _, folder := range []string{build, filepath.Join(mods, "Wallet-OuterWildsItems")} {
		if res, err := svc.InstallExtraFolderMod(t.Context(), "lethal-company", p.ID, folder); err != nil || len(res.Added) != 1 {
			t.Fatalf("install %s = %+v, %v", folder, res, err)
		}
	}
	if _, err := svc.InstallExtraFolderMod(t.Context(), "lethal-company", p.ID, mods); err == nil {
		t.Error("installed the folder of mods as one mod")
	}
	// A rebuild installed again replaces the plugin's entry rather than adding a second.
	writeFile(t, build, "MyPlugin.dll", "MZ rebuilt")
	res, err := svc.InstallExtraFolderMod(t.Context(), "lethal-company", p.ID, build)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, en := range res.Profile.Entries {
		for _, m := range en.Mods {
			if m.Name == "MyPlugin" {
				n++
			}
		}
	}
	if n != 1 {
		t.Fatalf("the rebuilt plugin is in the profile %d times: %+v", n, res.Profile.Entries)
	}
}
