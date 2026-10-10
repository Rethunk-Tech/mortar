//go:build !windows

package launchsvc

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const cacheGame = "cache-game"

func cachesEnv(t *testing.T) (svc *Service, profiles *profile.Store, dir string) {
	t.Helper()
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = append(slices.Clone(m.Games), components.GameInfo{
		ID: cacheGame, Name: "Cache Game", Enabled: true, Marker: "G.dll", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods"}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "1"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
		Paths: map[string]components.PathTemplate{
			"mods":     {Windows: "{documents}/G/Mods", Linux: "{documents}/G/Mods", Darwin: "{documents}/G/Mods"},
			"userData": {Windows: "{documents}/G", Linux: "{documents}/G", Darwin: "{documents}/G"},
		},
		Caches: []components.CachePath{{Role: "userData", Path: "thumb.package"}, {Role: "userData", Path: "cachestr"}},
	})
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	c := components.NewClient(nil)
	c.SetManifest(m)
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })

	data := t.TempDir()
	datadirtest.Use(t, data)
	t.Setenv("LOCALAPPDATA", data)
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "G.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders[cacheGame] = folder }); err != nil {
		t.Fatal(err)
	}
	_, profiles = testenv.Stores(t)
	home := t.TempDir()
	return NewService(home, set, profiles), profiles, filepath.Join(home, "Documents", "G")
}

func install(t *testing.T, profiles *profile.Store, pid, name string) {
	t.Helper()
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), name+".zip"), map[string]string{name + ".package": name})
	src := profile.Source{Kind: profile.KindCurseForge, Name: name, ModID: 1, FileID: 1}
	if _, err := profiles.InstallSource(t.Context(), cacheGame, pid, zip, src); err != nil {
		t.Fatal(err)
	}
}

func seedCaches(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "thumb.package"), "t")
	writeFile(t, filepath.Join(dir, "cachestr", "one"), "1")
	writeFile(t, filepath.Join(dir, "cachestr", "sub", "two"), "2")
	writeFile(t, filepath.Join(dir, "keep.ini"), "k")
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

func TestCachesClearOnceWhenTheModSetChangesAndPerProfile(t *testing.T) {
	svc, profiles, dir := cachesEnv(t)
	a := testenv.Profile(t, profiles, cacheGame, "A")
	install(t, profiles, a.ID, "one")
	seedCaches(t, dir)
	logs := captureLog(t)
	svc.clearCaches(cacheGame, a.ID, "")
	if exists(filepath.Join(dir, "thumb.package")) || exists(filepath.Join(dir, "cachestr", "one")) || exists(filepath.Join(dir, "cachestr", "sub")) {
		t.Fatal("a changed mod set must clear the caches")
	}
	if !exists(filepath.Join(dir, "cachestr")) || !exists(filepath.Join(dir, "keep.ini")) {
		t.Fatal("a named folder and other files stay")
	}
	if !strings.Contains(logs.String(), "cleared 3 cache files") {
		t.Fatalf("log: %q", logs.String())
	}
	seedCaches(t, dir)
	svc.clearCaches(cacheGame, a.ID, "")
	if !exists(filepath.Join(dir, "thumb.package")) || !strings.Contains(logs.String(), "mod set unchanged, nothing cleared") {
		t.Fatalf("an unchanged set must clear nothing: %q", logs.String())
	}
	b := testenv.Profile(t, profiles, cacheGame, "B")
	install(t, profiles, b.ID, "two")
	svc.clearCaches(cacheGame, b.ID, "")
	if exists(filepath.Join(dir, "thumb.package")) {
		t.Fatal("a second profile with another set clears again")
	}
	seedCaches(t, dir)
	install(t, profiles, a.ID, "three")
	svc.clearCaches(cacheGame, a.ID, "")
	if exists(filepath.Join(dir, "thumb.package")) {
		t.Fatal("a changed set clears again")
	}
}

func TestCachesStayWhenClearingIsOff(t *testing.T) {
	svc, profiles, dir := cachesEnv(t)
	if _, err := svc.settings.Update(func(v *settings.Settings) {
		gp := v.GamePrefs(cacheGame)
		gp.CacheClearing = settings.CacheClearingOff
		v.Games = map[string]*settings.GameSettings{cacheGame: &gp}
	}); err != nil {
		t.Fatal(err)
	}
	a := testenv.Profile(t, profiles, cacheGame, "A")
	install(t, profiles, a.ID, "one")
	seedCaches(t, dir)
	svc.clearCaches(cacheGame, a.ID, "")
	if !exists(filepath.Join(dir, "thumb.package")) {
		t.Fatal("clearing is off")
	}
}

func TestClearCacheRefusesAPathThatLeavesItsFolder(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "f"), "x")
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../x", "/etc/passwd", "link/f", "link", `a\b`} {
		if _, err := clearCache(root, rel); err == nil {
			t.Fatalf("%q was accepted", rel)
		}
	}
	if !exists(filepath.Join(outside, "f")) {
		t.Fatal("a file outside the folder was deleted")
	}
	if n, err := clearCache(root, "missing.package"); err != nil || n != 0 {
		t.Fatalf("a missing cache is nothing to do: %d, %v", n, err)
	}
}

func TestACacheThatCannotBeDeletedIsLoggedAndTheSetIsNotRecorded(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root deletes from a read-only folder")
	}
	svc, profiles, dir := cachesEnv(t)
	a := testenv.Profile(t, profiles, cacheGame, "A")
	install(t, profiles, a.ID, "one")
	seedCaches(t, dir)
	locked := filepath.Join(dir, "cachestr")
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	logs := captureLog(t)
	svc.clearCaches(cacheGame, a.ID, "")
	if !strings.Contains(logs.String(), "cachestr") || profiles.CacheSet(cacheGame, a.ID) != "" {
		t.Fatalf("log %q, recorded %q", logs.String(), profiles.CacheSet(cacheGame, a.ID))
	}
}
