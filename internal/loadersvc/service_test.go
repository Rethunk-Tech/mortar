package loadersvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func put(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// SMAPI installed by hand: no settings value, no log, only the installer's Mods folder.
func TestBundledBuiltFromGameFolder(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("LOCALAPPDATA", data)
	game := t.TempDir()
	put(t, filepath.Join(game, "Stardew Valley.dll"), "x")
	put(t, filepath.Join(game, "StardewValley-original"), "x")
	put(t, filepath.Join(game, "StardewValley"), "#!/bin/sh\nexec StardewModdingAPI\n")
	put(t, filepath.Join(game, "Mods", "ConsoleCommands", "manifest.json"), `{"Name": "Console Commands", "Version": "4.5.2", "UniqueId": "SMAPI.ConsoleCommands", "EntryDll": "ConsoleCommands.dll"}`)
	put(t, filepath.Join(game, "Mods", "SaveBackup", "manifest.json"), `{"Name": "Save Backup", "Version": "4.5.2", "UniqueId": "SMAPI.SaveBackup", "EntryDll": "SaveBackup.dll"}`)

	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders["stardew"] = game }); err != nil {
		t.Fatal(err)
	}
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), set, items, profiles)
	early, err := profiles.Create("stardew", "Early")
	if err != nil || len(early.Entries) != 0 {
		t.Fatalf("early = %+v, %v", early, err)
	}
	profiles.Bundled = BundledKey(svc)
	SyncBundled(svc, "stardew")
	all, err := profiles.List("stardew")
	if err != nil || len(all) != 1 || len(all[0].Entries) != 1 || all[0].Entries[0].Key != "smapi-4.5.2" {
		t.Fatalf("existing profile = %+v, %v", all, err)
	}
	late, err := profiles.Create("stardew", "Late")
	if err != nil || len(late.Entries) != 1 {
		t.Fatalf("late = %+v, %v", late, err)
	}
	dir, err := items.Path("stardew", "smapi-4.5.2")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ConsoleCommands", "SaveBackup"} {
		if _, err := os.Stat(filepath.Join(dir, name, "manifest.json")); err != nil {
			t.Fatal(err)
		}
	}
	if set.Get().Loaders["stardew"] != "4.5.2" {
		t.Fatalf("loaders = %v", set.Get().Loaders)
	}
}
