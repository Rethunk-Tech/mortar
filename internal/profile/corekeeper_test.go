package profile

import (
	"path/filepath"
	"strings"
	"testing"
)

// Core Keeper installs a Thunderstore package through BepInEx in the profile and a mod.io mod whole into the game's own
// Mods folder, both into one profile.
func TestCoreKeeperRoutesEachSourceToItsOwnTarget(t *testing.T) {
	t.Parallel()
	const game = "core-keeper"
	e := newEnv(t)
	mods := filepath.Join(t.TempDir(), "Mods")
	e.TrayFolder = func(_, role string) (string, error) {
		if role != "modioMods" {
			t.Errorf("role = %q", role)
		}
		return mods, nil
	}
	p, err := e.Create(game, "S")
	if err != nil {
		t.Fatal(err)
	}
	sdk := zipOf(t, "m.zip", map[string]string{"ModManifest.json": "{}", "Data/x.bundle": "b"})
	if _, err := e.InstallSource(t.Context(), game, p.ID, sdk, Source{Kind: KindModio, Name: "42", ModName: "SdkMod", Version: "1.0"}); err != nil {
		t.Fatal(err)
	}
	got := trayTree(t, mods)
	if len(got) != 2 || got[filepath.Join("modio-42", "ModManifest.json")] != "{}" {
		t.Fatalf("Mods folder = %v", got)
	}

	plugin := zipOf(t, "t.zip", map[string]string{"plugins/Plug.dll": "d"})
	res, err := e.InstallSource(t.Context(), game, p.ID, plugin, Source{Kind: KindThunderstore, Name: "Author-Plug", ModName: "Plug", Version: "1.0"})
	if err != nil {
		t.Fatal(err)
	}
	owners, _ := e.PackageFileOwners(game, p.ID)
	inPlugins := false
	for rel := range owners {
		inPlugins = inPlugins || (strings.HasPrefix(rel, "BepInEx/plugins/") && strings.HasSuffix(rel, "/Plug.dll"))
	}
	if !inPlugins {
		t.Fatalf("the Thunderstore plugin is not in the profile's BepInEx folder: %v", owners)
	}
	if len(trayTree(t, mods)) != 2 || len(res.Profile.Entries) != 2 {
		t.Fatalf("Mods folder %v, entries %v", trayTree(t, mods), keysOf(res.Profile))
	}
}
