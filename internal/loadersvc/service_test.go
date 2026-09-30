package loadersvc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/loader"
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
	Attach(svc, "stardew")
	all, err := profiles.List("stardew")
	if err != nil || len(all) != 1 || len(all[0].Entries) != 2 || all[0].Entries[0].Key != "smapi-4.5.2" || all[0].Entries[1].Key != bridgeKey() || all[0].Entries[1].Source.Kind != profile.SourceMortar {
		t.Fatalf("existing profile = %+v, %v", all, err)
	}
	late, err := profiles.Create("stardew", "Late")
	if err != nil || len(late.Entries) != 2 {
		t.Fatalf("late = %+v, %v", late, err)
	}
	if _, err := os.Stat(filepath.Join(dir0(t, items, bridgeKey()), "MortarSmapiBridge", "manifest.json")); err != nil {
		t.Fatal(err)
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

func dir0(t *testing.T, items *store.Store, key string) string {
	t.Helper()
	dir, err := items.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

type ensureEnv struct {
	svc      *Service
	profiles *profile.Store
	game     string
	installs *atomic.Int32
}

// newEnsureEnv is a Stardew folder without SMAPI, whose installer is replaced by a counter: nothing real is installed.
func newEnsureEnv(t *testing.T) ensureEnv {
	t.Helper()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("LOCALAPPDATA", data)
	folder := t.TempDir()
	put(t, filepath.Join(folder, "Stardew Valley.dll"), "x")
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders["stardew"] = folder }); err != nil {
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
	var installs atomic.Int32
	svc.run = func(context.Context, string, bool) (loader.Status, error) {
		installs.Add(1)
		return loader.Status{Installed: true}, nil
	}
	Attach(svc, "stardew")
	return ensureEnv{svc, profiles, folder, &installs}
}

func TestCreatingAProfileInstallsAMissingLoader(t *testing.T) {
	e := newEnsureEnv(t)
	if _, err := e.profiles.Create("stardew", "Main"); err != nil {
		t.Fatal(err)
	}
	e.svc.background.Wait()
	if got := e.installs.Load(); got != 1 {
		t.Fatalf("installs = %d, want 1", got)
	}
}

func TestStartupInstallsAMissingLoaderOnlyWhenProfilesExist(t *testing.T) {
	e := newEnsureEnv(t)
	EnsureExisting(e.svc, "stardew")
	e.svc.background.Wait()
	if got := e.installs.Load(); got != 0 {
		t.Fatalf("no profiles yet, installs = %d", got)
	}
	if _, err := e.profiles.Create("stardew", "Main"); err != nil {
		t.Fatal(err)
	}
	e.svc.background.Wait()
	EnsureExisting(e.svc, "stardew")
	e.svc.background.Wait()
	if got := e.installs.Load(); got != 2 {
		t.Fatalf("installs = %d, want 2 (creation and startup)", got)
	}
}

func TestEnsureLeavesAWorkingLoaderAlone(t *testing.T) {
	e := newEnsureEnv(t)
	put(t, filepath.Join(e.game, "StardewValley-original"), "x")
	put(t, filepath.Join(e.game, "StardewValley"), "#!/bin/sh\nexec StardewModdingAPI\n")
	put(t, filepath.Join(e.game, "Mods", "ConsoleCommands", "manifest.json"), `{"Name": "Console Commands", "Version": "4.5.2", "UniqueId": "SMAPI.ConsoleCommands", "EntryDll": "ConsoleCommands.dll"}`)
	if _, err := e.svc.Ensure(context.Background(), "stardew", false); err != nil {
		t.Fatal(err)
	}
	if got := e.installs.Load(); got != 0 {
		t.Fatalf("installs = %d, want 0", got)
	}
	// A game update replaces the launcher: SMAPI is broken and is reinstalled.
	put(t, filepath.Join(e.game, "StardewValley"), "native launcher")
	if _, err := e.svc.Ensure(context.Background(), "stardew", false); err != nil {
		t.Fatal(err)
	}
	if got := e.installs.Load(); got != 1 {
		t.Fatalf("installs = %d, want 1", got)
	}
}

func TestInstallRefusesWhileTheGameRunsOutsideMortar(t *testing.T) {
	e := newEnsureEnv(t)
	e.svc.procDir = t.TempDir()
	put(t, filepath.Join(e.svc.procDir, "4242", "cmdline"), "/games/Stardew Valley/Stardew Valley\x00")
	if _, err := e.svc.install(context.Background(), "stardew", false); err == nil || !strings.Contains(err.Error(), "is running") {
		t.Fatalf("err = %v, want a running refusal", err)
	}
}

func bridgeKey() string { return store.BridgeKey(game.Find("stardew").BridgeVersion()) }
