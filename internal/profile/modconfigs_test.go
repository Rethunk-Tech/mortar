package profile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/melonloader"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// A switched-off mod's config is set aside and comes back when it is switched on; removing the mod sets it aside, so Undo and adding it again bring it back.
func TestModConfigFollowsTheMod(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.InstallSource(t.Context(), "lethal-company", p.ID, tsZip(t, "1.0.0"), Source{Kind: KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	en := res.Profile.Entries[0]
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	live := filepath.Join(dir, "BepInEx", "config", "Mod.cfg")
	aside := filepath.Join(dir, ConfigAsideDir, "Mod.cfg")
	writeFile(t, dir, "BepInEx/config/Mod.cfg", "[General]\nA = 1\n")
	exists := func(path string) bool {
		_, err := safefs.Stat(path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
		return err == nil
	}
	toggle := func(on bool) {
		t.Helper()
		if _, _, err := e.enableMod("lethal-company", p.ID, en.Key, en.Mods[0].ID, on); err != nil {
			t.Fatal(err)
		}
		if err := e.SyncPackages("lethal-company", p.ID); err != nil {
			t.Fatal(err)
		}
	}
	toggle(false)
	if exists(live) || !exists(aside) {
		t.Fatalf("switched off: live %v, aside %v", exists(live), exists(aside))
	}
	toggle(true)
	if !exists(live) || exists(aside) {
		t.Fatalf("switched on: live %v, aside %v", exists(live), exists(aside))
	}
	if _, err := e.RemoveEntries("lethal-company", p.ID, []string{en.Key}); err != nil {
		t.Fatal(err)
	}
	if exists(live) || !exists(aside) {
		t.Fatalf("removed: live %v, aside %v", exists(live), exists(aside))
	}
	if _, err := e.RestoreEntries("lethal-company", p.ID, []Entry{en}); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages("lethal-company", p.ID); err != nil {
		t.Fatal(err)
	}
	if b, _ := safefs.ReadFile(live); string(b) != "[General]\nA = 1\n" || exists(aside) {
		t.Fatalf("undone: live %q, aside %v", b, exists(aside))
	}
}

// A MelonLoader profile keeps its mods' settings in UserData, so a mod's config there follows it off and back on, and a
// backup of the profile carries it.
func TestModConfigFollowsTheModInALoaderThatKeepsItElsewhere(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("beetleball", "ML")
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.InstallSource(t.Context(), "beetleball", p.ID, tsZip(t, "1.0.0"), Source{Kind: KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	en := res.Profile.Entries[0]
	dir, _ := e.ProfileDir("beetleball", p.ID)
	live := filepath.Join(dir, "UserData", "Mod.cfg")
	aside := filepath.Join(dir, ConfigAsideDir, "Mod.cfg")
	writeFile(t, dir, "UserData/Mod.cfg", "[General]\nA = 1\n")
	exists := func(path string) bool {
		_, err := safefs.Stat(path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
		return err == nil
	}
	toggle := func(on bool) {
		t.Helper()
		if _, _, err := e.enableMod("beetleball", p.ID, en.Key, en.Mods[0].ID, on); err != nil {
			t.Fatal(err)
		}
		if err := e.SyncPackages("beetleball", p.ID); err != nil {
			t.Fatal(err)
		}
	}
	toggle(false)
	if exists(live) || !exists(aside) {
		t.Fatalf("switched off: live %v, aside %v", exists(live), exists(aside))
	}
	toggle(true)
	if !exists(live) || exists(aside) {
		t.Fatalf("switched on: live %v, aside %v", exists(live), exists(aside))
	}
	if _, err := e.RemoveEntries("beetleball", p.ID, []string{en.Key}); err != nil {
		t.Fatal(err)
	}
	if exists(live) || !exists(aside) {
		t.Fatalf("removed: live %v, aside %v", exists(live), exists(aside))
	}
}

// A config file is one in any of the profile's loader config folders, not only BepInEx's.
func TestAConfigFileIsOneInAnyLoaderConfigFolder(t *testing.T) {
	t.Parallel()
	roots := loaderConfigRoots("beetleball", Profile{})
	for _, rel := range []string{"UserData/Mod.cfg", "BepInEx/config/Mod.cfg"} {
		if !isConfig(rel, roots) {
			t.Errorf("%s is not a config file in %v", rel, roots)
		}
	}
	if isConfig("Mods/Mod.cfg", roots) {
		t.Errorf("a plugin is a config file in %v", roots)
	}
}

// A fork that shares a package name with a mod still switched on does not take that mod's config, and a removal that
// fails leaves every config where it was.
func TestModConfigStaysWithTheEnabledOwnerAndAFailedRemoveKeepsIt(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	var keys []Entry
	for _, name := range []string{"Ns-Mod", "Fork-Mod"} {
		res, err := e.InstallSource(t.Context(), "lethal-company", p.ID, tsZip(t, "1.0.0"), Source{Kind: KindThunderstore, Name: name, Version: "1.0.0"})
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, res.Profile.Entries[len(res.Profile.Entries)-1])
	}
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	writeFile(t, dir, "BepInEx/config/Mod.cfg", "[General]\nA = 1\n")
	live := filepath.Join(dir, "BepInEx", "config", "Mod.cfg")
	if _, _, err := e.enableMod("lethal-company", p.ID, keys[0].Key, keys[0].Mods[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages("lethal-company", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := safefs.Stat(live); err != nil {
		t.Fatalf("the enabled fork's config was parked: %v", err)
	}
	if _, err := e.RemoveEntries("lethal-company", p.ID, []string{keys[0].Key, "no-such-key"}); err == nil {
		t.Fatal("a missing key removed")
	}
	if _, err := e.RemoveEntries("lethal-company", p.ID, []string{keys[0].Key}); err != nil {
		t.Fatal(err)
	}
	if _, err := safefs.Stat(live); err != nil {
		t.Fatalf("removing the disabled one took the enabled one's config: %v", err)
	}
}

// A package named BepInEx (or a plugin with that GUID) never takes the loader's own BepInEx.cfg with it.
func TestTheLoadersConfigIsNoModsOwn(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.InstallSource(t.Context(), "lethal-company", p.ID, tsZip(t, "1.0.0"), Source{Kind: KindThunderstore, Name: "AnyTeam-BepInEx", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	en := res.Profile.Entries[0]
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	writeFile(t, dir, "BepInEx/config/BepInEx.cfg", "[Logging]\nA = 1\n")
	if _, _, err := e.enableMod("lethal-company", p.ID, en.Key, en.Mods[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages("lethal-company", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RemoveEntries("lethal-company", p.ID, []string{en.Key}); err != nil {
		t.Fatal(err)
	}
	if _, err := safefs.Stat(filepath.Join(dir, "BepInEx", "config", "BepInEx.cfg")); err != nil {
		t.Fatalf("BepInEx.cfg went with the mod: %v", err)
	}
}

// A config set aside is not seeded over by the package's own copy, so switching the mod on brings back the player's.
func TestSeedConfigsSkipsAConfigSetAside(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(t.TempDir(), "Mod.cfg")
	if err := os.WriteFile(src, []byte("default"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, ConfigAsideDir+"/Mod.cfg", "mine")
	if err := seedConfigs(dir, Profile{}, []string{bepInExConfigRoot}, map[string]packageFile{"BepInEx/config/Mod.cfg": {src: src}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "BepInEx", "config", "Mod.cfg")); err == nil {
		t.Fatal("seeded over a config set aside")
	}
}

// Switching a mod off replaces an older copy already set aside with the live one, and switching it on brings that back.
func TestModConfigAsideCopyIsReplacedByTheLiveOne(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.InstallSource(t.Context(), "lethal-company", p.ID, tsZip(t, "1.0.0"), Source{Kind: KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	en := res.Profile.Entries[0]
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	writeFile(t, dir, "BepInEx/config/Mod.cfg", "new")
	writeFile(t, dir, ConfigAsideDir+"/Mod.cfg", "old")
	toggle := func(on bool) {
		t.Helper()
		if _, _, err := e.enableMod("lethal-company", p.ID, en.Key, en.Mods[0].ID, on); err != nil {
			t.Fatal(err)
		}
		if err := e.SyncPackages("lethal-company", p.ID); err != nil {
			t.Fatal(err)
		}
	}
	read := func(rel string) string {
		b, _ := safefs.ReadFile(filepath.Join(dir, rel))
		return string(b)
	}
	toggle(false)
	if got := read(ConfigAsideDir + "/Mod.cfg"); got != "new" {
		t.Fatalf("aside holds %q", got)
	}
	toggle(true)
	if got := read("BepInEx/config/Mod.cfg"); got != "new" {
		t.Fatalf("live holds %q", got)
	}
}

// A package's config file is taken when the package brings it, over another mod's own copy (a theme pack's settings for
// the mod it themes), and again when an update changes it; the player's edits in between stay, and removing the
// package and adding it back applies it again.
func TestAPackagesConfigIsTakenOnArrivalAndKeepsLaterEdits(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "BepInEx", "config", "Emblem.cfg")
	read := func() string {
		t.Helper()
		b, err := os.ReadFile(filepath.Clean(cfg))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	src := filepath.Join(t.TempDir(), "Emblem.cfg")
	ship := func(text string) map[string]packageFile {
		t.Helper()
		if err := os.WriteFile(src, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return map[string]packageFile{"BepInEx/config/Emblem.cfg": {src: src, key: "theme"}}
	}
	withTheme := Profile{Entries: []Entry{{Key: "theme"}}}
	// The profile's first seed with this record adopts what it has: nothing is overwritten.
	writeFile(t, dir, "BepInEx/config/Emblem.cfg", "emblem's own")
	if err := seedConfigs(dir, Profile{}, []string{bepInExConfigRoot}, nil); err != nil {
		t.Fatal(err)
	}
	if err := seedConfigs(dir, withTheme, []string{bepInExConfigRoot}, ship("theme v1")); err != nil {
		t.Fatal(err)
	}
	if read() != "theme v1" {
		t.Fatalf("the theme's config was not taken: %q", read())
	}
	if b, err := os.ReadFile(filepath.Clean(filepath.Join(dir, replacedDir, "BepInEx", "config", "Emblem.cfg"))); err != nil || string(b) != "emblem's own" {
		t.Fatalf("the replaced copy was not kept: %q, %v", b, err)
	}
	writeFile(t, dir, "BepInEx/config/Emblem.cfg", "player's edit")
	if err := seedConfigs(dir, withTheme, []string{bepInExConfigRoot}, ship("theme v1")); err != nil || read() != "player's edit" {
		t.Fatalf("a later sync overwrote the player's edit: %q, %v", read(), err)
	}
	if err := seedConfigs(dir, withTheme, []string{bepInExConfigRoot}, ship("theme v2")); err != nil || read() != "theme v2" {
		t.Fatalf("an update's changed config was not taken: %q, %v", read(), err)
	}
	writeFile(t, dir, "BepInEx/config/Emblem.cfg", "player's edit")
	if err := seedConfigs(dir, Profile{}, []string{bepInExConfigRoot}, nil); err != nil {
		t.Fatal(err)
	}
	if err := seedConfigs(dir, withTheme, []string{bepInExConfigRoot}, ship("theme v2")); err != nil || read() != "theme v2" {
		t.Fatalf("adding the package again did not apply its config: %q, %v", read(), err)
	}
}

// A profile that has no record yet keeps every config it already has, so an upgrade overwrites no player's settings.
func TestAProfileWithoutARecordKeepsItsConfigs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(t.TempDir(), "Mod.cfg")
	if err := os.WriteFile(src, []byte("default"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "BepInEx/config/Mod.cfg", "mine")
	files := map[string]packageFile{"BepInEx/config/Mod.cfg": {src: src, key: "mod"}}
	if err := seedConfigs(dir, Profile{Entries: []Entry{{Key: "mod"}}}, []string{bepInExConfigRoot}, files); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Clean(filepath.Join(dir, "BepInEx", "config", "Mod.cfg"))); string(b) != "mine" {
		t.Fatalf("an existing config was overwritten on the first seed: %q", b)
	}
}

// Issue #23: a theme pack (Wallet-OuterWildsMenu) ships config/Darkbrewery.Emblem.cfg to set up Emblem, the mod it
// themes. Installed after Emblem has written its own config, the pack's copy takes over at the next sync, the replaced
// one is kept, and an edit after that survives later syncs.
func TestAThemePacksConfigForAnotherModAppliesOnInstall(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	sync := func() {
		t.Helper()
		if err := e.SyncPackages("lethal-company", p.ID); err != nil {
			t.Fatal(err)
		}
	}
	emblem := testfs.WriteZip(t, filepath.Join(t.TempDir(), "Emblem.zip"), map[string]string{
		"manifest.json": `{"name":"Emblem","version_number":"1.0.0"}`, "plugins/Emblem.dll": "MZ",
	})
	if _, err := e.InstallSource(t.Context(), "lethal-company", p.ID, emblem, Source{Kind: KindThunderstore, Name: "Darkbrewery-Emblem", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	sync()
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	// Emblem's first run writes its own config.
	writeFile(t, dir, "BepInEx/config/Darkbrewery.Emblem.cfg", "emblem defaults")
	theme := testfs.WriteZip(t, filepath.Join(t.TempDir(), "OuterWildsMenu.zip"), map[string]string{
		"manifest.json":                                 `{"name":"OuterWildsMenu","version_number":"1.0.1"}`,
		"config/Darkbrewery.Emblem.cfg":                 "outer wilds theme",
		"plugins/Emblem/Wallet_OuterWildsMenu/logo.png": "png",
	})
	if _, err := e.InstallSource(t.Context(), "lethal-company", p.ID, theme, Source{Kind: KindThunderstore, Name: "Wallet-OuterWildsMenu", Version: "1.0.1"}); err != nil {
		t.Fatal(err)
	}
	sync()
	read := func(rel string) string {
		b, _ := safefs.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		return string(b)
	}
	if got := read("BepInEx/config/Darkbrewery.Emblem.cfg"); got != "outer wilds theme" {
		t.Fatalf("the theme's config was not applied: %q", got)
	}
	if got := read(replacedDir + "/BepInEx/config/Darkbrewery.Emblem.cfg"); got != "emblem defaults" {
		t.Fatalf("the replaced config was not kept: %q", got)
	}
	writeFile(t, dir, "BepInEx/config/Darkbrewery.Emblem.cfg", "tweaked")
	sync()
	if got := read("BepInEx/config/Darkbrewery.Emblem.cfg"); got != "tweaked" {
		t.Fatalf("a later sync overwrote the player's edit: %q", got)
	}
}
