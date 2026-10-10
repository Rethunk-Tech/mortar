package folder_test

import (
	"io/fs"
	"os"
	"path/filepath"
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
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const gameID = "folder-game"

func useFolderGame(t *testing.T) {
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
		Paths:   map[string]components.PathTemplate{"mods": {Windows: "{documents}/Mods", Linux: "{documents}/Mods", Darwin: "{documents}/Mods"}},
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
	useFolderGame(t)
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
}
