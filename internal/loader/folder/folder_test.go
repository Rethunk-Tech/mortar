package folder_test

import (
	"io/fs"
	"os"
	"path/filepath"
	goruntime "runtime"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/folder"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const gameID = "folder-game"

func useFolderGame(t *testing.T, modsPath string) {
	t.Helper()
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = append(slices.Clone(m.Games), components.GameInfo{
		ID: gameID, Name: "Folder Game", Enabled: true, Marker: "G.exe", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods", MaxDepth: map[string]int{"ts4script": 1}, KeepWhole: []string{"ts4script"}}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "1"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
		Paths:   map[string]components.PathTemplate{"mods": {Windows: modsPath, Linux: modsPath, Darwin: modsPath}},
	})
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	c := components.NewClient(nil)
	c.SetManifest(m)
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })
}

func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, _ error) error {
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			out[rel+"/"] = ""
		} else {
			b, _ := fsx.ReadFile(p)
			out[rel] = string(b)
		}
		return nil
	})
	return out
}

func TestEnabledModsAreSwappedIntoTheModsFolderAndTakenBack(t *testing.T) {
	useFolderGame(t, "{documents}/Mods")
	dir := t.TempDir()
	ps := profile.OpenIn(dir, store.OpenAt(filepath.Join(dir, "store")))
	p, err := ps.Create(gameID, "A")
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, name := range []string{"one", "two", "three"} {
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), name+".zip"), map[string]string{
			name + "/x.package":   name,
			name + "/y.ts4script": name,
		})
		res, err := ps.InstallArchive(t.Context(), gameID, p.ID, zip)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, res.Profile.Entries[len(res.Profile.Entries)-1].Key)
	}
	if _, err := ps.SetModEnabled(gameID, p.ID, keys[2], mod.NewID(mod.FormatFolder, keys[2]), false); err != nil {
		t.Fatal(err)
	}
	if err := ps.SyncPackages(gameID, p.ID); err != nil {
		t.Fatal(err)
	}
	profDir, err := ps.ProfileDir(gameID, p.ID)
	if err != nil {
		t.Fatal(err)
	}

	l, ok := loader.Get("folder")
	if !ok {
		t.Fatal("no folder loader")
	}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := l.Contribute(t.Context(), plan, loader.ProfileView{Game: gameID, Dir: profDir}); err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 4 {
		t.Fatalf("two enabled mods of two files each must be placed, got %+v", plan.Files)
	}
	places, ok := l.(loader.PlacesFolder)
	if !ok {
		t.Fatal("the folder loader must say which folder it places")
	}
	folder, role, ok := places.PlacedFolder(gameID)
	if !ok || folder != "Mods" || role != "mods" {
		t.Fatalf("PlacedFolder = %q, %q, %v", folder, role, ok)
	}
	for _, f := range plan.Files {
		if f.Root != role || f.Src != filepath.Join(profDir, folder, f.Dst) {
			t.Fatalf("%+v is not the placed folder's file at the same path below the %s folder", f, role)
		}
	}

	mods := filepath.Join(t.TempDir(), "Documents", "Mods")
	if err := os.MkdirAll(filepath.Join(mods, "one"), 0o750); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		filepath.Join(mods, "Resource.cfg"):     "players own",
		filepath.Join(mods, "one", "x.package"): "players copy",
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	before := tree(t, mods)

	d, _ := deploy.Get("copy-into-install")
	dp, err := d.Plan(deploy.View{JournalDir: filepath.Join(dir, "journal"), Roots: map[string]string{"mods": mods}}, filepath.Join(dir, "install"), plan.Files)
	if err != nil {
		t.Fatal(err)
	}
	m, err := d.Apply(t.Context(), dp)
	if err != nil {
		t.Fatal(err)
	}
	during := tree(t, mods)
	if during["Resource.cfg"] != "players own" {
		t.Fatal("a file the profile does not own must be left alone")
	}
	if during[filepath.Join("one", "x.package")] != "one" || during[filepath.Join("two", "y.ts4script")] != "two" {
		t.Fatalf("enabled mods are in the folder: %v", during)
	}
	if _, has := during[filepath.Join("three", "x.package")]; has {
		t.Fatal("a disabled mod must not be placed")
	}
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	after := tree(t, mods)
	if len(after) != len(before) {
		t.Fatalf("folder differs\nbefore %v\n after %v", before, after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s: %q, want %q", k, after[k], v)
		}
	}
}

func TestAnUnknownGameIsRefused(t *testing.T) {
	l, _ := loader.Get("folder")
	if err := l.Contribute(t.Context(), launchplan.New(launchplan.ModeProfile), loader.ProfileView{Game: "no-such-game"}); err == nil {
		t.Fatal("an unknown game must be refused")
	}
	if places, ok := l.(loader.PlacesFolder); !ok {
		t.Fatal("the folder loader must say which folder it places")
	} else if _, _, ok := places.PlacedFolder("no-such-game"); ok {
		t.Fatal("an unknown game places no folder")
	}
}

// A mods folder named `{install}/...` needs nothing beyond the catalog: the install is the one place a launch must
// leave as it found it, so the whole folder is compared, including a game file a mod replaces.
func TestAModsFolderInsideTheInstallIsFilledAndPurgedWithTheInstallUnchanged(t *testing.T) {
	useFolderGame(t, "{install}")
	dir := t.TempDir()
	ps := profile.OpenIn(dir, store.OpenAt(filepath.Join(dir, "store")))
	p, err := ps.Create(gameID, "A")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "mod.zip"), map[string]string{
		"mods/modA/content/blob0.bundle": "mod",
		"bin/config/matrix.xml":          "mods version",
		"dlc/dlcB/readme.txt":            "dlc",
	})
	if _, err := ps.InstallArchive(t.Context(), gameID, p.ID, zip); err != nil {
		t.Fatal(err)
	}
	if err := ps.SyncPackages(gameID, p.ID); err != nil {
		t.Fatal(err)
	}
	profDir, err := ps.ProfileDir(gameID, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	l, _ := loader.Get("folder")
	plan := launchplan.New(launchplan.ModeProfile)
	if err := l.Contribute(t.Context(), plan, loader.ProfileView{Game: gameID, Dir: profDir}); err != nil {
		t.Fatal(err)
	}

	install := t.TempDir()
	for path, body := range map[string]string{
		"G.exe":                 "game",
		"bin/config/matrix.xml": "players version",
		"mods/modX/own.bundle":  "player's mod",
	} {
		full := filepath.Join(install, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	info, _ := components.Game(gameID)
	modsDir, err := runtime.Resolve(runtime.Install{Game: gameID, Dir: install, Platform: goruntime.GOOS, Home: t.TempDir()}, info.Paths["mods"])
	if err != nil {
		t.Fatal(err)
	}
	if modsDir != install {
		t.Fatalf("mods folder resolves to %s, want the install %s", modsDir, install)
	}
	before := tree(t, install)

	d, _ := deploy.Get("copy-into-install")
	dp, err := d.Plan(deploy.View{JournalDir: filepath.Join(dir, "journal"), Roots: map[string]string{"mods": modsDir}}, install, plan.Files)
	if err != nil {
		t.Fatal(err)
	}
	m, err := d.Apply(t.Context(), dp)
	if err != nil {
		t.Fatal(err)
	}
	during := tree(t, install)
	if during[filepath.Join("mods", "modA", "content", "blob0.bundle")] != "mod" || during[filepath.Join("bin", "config", "matrix.xml")] != "mods version" ||
		during[filepath.Join("dlc", "dlcB", "readme.txt")] != "dlc" || during[filepath.Join("mods", "modX", "own.bundle")] != "player's mod" {
		t.Fatalf("the install during the launch: %v", during)
	}
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	after := tree(t, install)
	if len(after) != len(before) {
		t.Fatalf("install differs\nbefore %v\n after %v", before, after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s: %q, want %q", k, after[k], v)
		}
	}
}
