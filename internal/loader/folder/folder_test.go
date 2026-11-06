package folder_test

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/folder"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testcatalog"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const gameID = "folder-game"

func useFolderGame(t *testing.T, modsPath string) {
	t.Helper()
	testcatalog.AddGames(t, components.GameInfo{
		ID: gameID, Name: "Folder Game", Enabled: true, Marker: "G.exe", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods", MaxDepth: map[string]int{"ts4script": 1}, KeepWhole: []string{"ts4script"}}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "1"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
		Paths:   map[string]components.PathTemplate{"mods": {Windows: modsPath, Linux: modsPath, Darwin: modsPath}},
	})
}

func TestEnabledModsAreSwappedIntoTheModsFolderAndTakenBack(t *testing.T) {
	useFolderGame(t, "{documents}/Mods")
	dir := t.TempDir()
	_, ps := testenv.StoresIn(t, dir)
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
	before := testfs.Tree(t, mods)

	d := deploy.Place{}
	dp, err := d.Plan(deploy.View{JournalDir: filepath.Join(dir, "journal"), Roots: map[string]string{"mods": mods}}, filepath.Join(dir, "install"), plan.Files)
	if err != nil {
		t.Fatal(err)
	}
	m, err := d.Apply(t.Context(), dp)
	if err != nil {
		t.Fatal(err)
	}
	during := testfs.Tree(t, mods)
	if during["Resource.cfg"] != "players own" {
		t.Fatal("a file the profile does not own must be left alone")
	}
	if during["one/x.package"] != "one" || during["two/y.ts4script"] != "two" {
		t.Fatalf("enabled mods are in the folder: %v", during)
	}
	if _, has := during["three/x.package"]; has {
		t.Fatal("a disabled mod must not be placed")
	}
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	after := testfs.Tree(t, mods)
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
	_, ps := testenv.StoresIn(t, dir)
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
		testfs.WriteAt(t, full, []byte(body))
	}
	info, _ := components.Game(gameID)
	modsDir, err := runtime.Resolve(runtime.Install{Game: gameID, Dir: install, Platform: goruntime.GOOS, Home: t.TempDir()}, info.Paths["mods"])
	if err != nil {
		t.Fatal(err)
	}
	if modsDir != install {
		t.Fatalf("mods folder resolves to %s, want the install %s", modsDir, install)
	}
	before := testfs.Tree(t, install)

	d := deploy.Place{}
	dp, err := d.Plan(deploy.View{JournalDir: filepath.Join(dir, "journal"), Roots: map[string]string{"mods": modsDir}}, install, plan.Files)
	if err != nil {
		t.Fatal(err)
	}
	m, err := d.Apply(t.Context(), dp)
	if err != nil {
		t.Fatal(err)
	}
	during := testfs.Tree(t, install)
	if during["mods/modA/content/blob0.bundle"] != "mod" || during["bin/config/matrix.xml"] != "mods version" ||
		during["dlc/dlcB/readme.txt"] != "dlc" || during["mods/modX/own.bundle"] != "player's mod" {
		t.Fatalf("the install during the launch: %v", during)
	}
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	after := testfs.Tree(t, install)
	if len(after) != len(before) {
		t.Fatalf("install differs\nbefore %v\n after %v", before, after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s: %q, want %q", k, after[k], v)
		}
	}
}

func TestProfileModsDirErrors(t *testing.T) {
	if _, err := folder.ProfileModsDir("no-such-game", t.TempDir()); err == nil || !strings.Contains(err.Error(), "unknown game") {
		t.Errorf("unknown game: %v", err)
	}
	for _, g := range components.Games() {
		if _, ok := g.Target(folder.TargetID); ok {
			continue
		}
		if _, err := folder.ProfileModsDir(g.ID, t.TempDir()); err == nil || !strings.Contains(err.Error(), "no "+folder.TargetID+" target") {
			t.Errorf("%s without a mods target: %v", g.ID, err)
		}
		return
	}
	t.Skip("every catalog game has a mods target")
}

func TestAGameWithAModsArgGetsTheProfileModsFolderOnItsCommandLine(t *testing.T) {
	testcatalog.AddGames(t, components.GameInfo{
		ID: "arg-game", Name: "Arg Game", Enabled: true, Marker: "G.exe", Deploy: "profile", ModsArg: "--mods-path",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profileMods}", ModRoot: "mod.modinfo"}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "1"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
	})

	_, ps := testenv.StoresIn(t, t.TempDir())
	prof, err := ps.Create("arg-game", "A")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "pkg.zip"), map[string]string{
		"manifest.json":     "{}",
		"Cool/mod.modinfo":  "{}",
		"Cool/Lua/main.lua": "x",
	})
	if _, err := ps.InstallArchive(t.Context(), "arg-game", prof.ID, zip); err != nil {
		t.Fatal(err)
	}
	if err := ps.SyncPackages("arg-game", prof.ID); err != nil {
		t.Fatal(err)
	}
	l, _ := loader.Get("folder")
	dir, err := ps.ProfileDir("arg-game", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := filepath.Glob(filepath.Join(dir, "mods", "*", "Lua", "main.lua")); len(got) != 1 {
		t.Fatalf("the mod folder is laid out whole in its own folder under the profile's mods folder: %v", testfs.Tree(t, dir))
	}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := l.Contribute(t.Context(), plan, loader.ProfileView{Game: "arg-game", Dir: dir}); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "mods")
	if len(plan.Args) != 2 || plan.Args[0] != "--mods-path" || plan.Args[1] != want || len(plan.Files) != 0 {
		t.Fatalf("args %v files %v, want --mods-path %s and no placed file", plan.Args, plan.Files, want)
	}
	places, ok := l.(loader.PlacesFolder)
	if !ok {
		t.Fatal("the folder loader must say which folder it places")
	}
	if _, _, ok := places.PlacedFolder("arg-game"); ok {
		t.Fatal("a game with no mods path role places nothing")
	}

	proton := launchplan.New(launchplan.ModeProfile)
	if err := l.Contribute(t.Context(), proton, loader.ProfileView{Game: "arg-game", Dir: dir, Runtime: "proton"}); err != nil {
		t.Fatal(err)
	}
	if got := proton.Args[1]; !strings.HasPrefix(got, "Z:") && goruntime.GOOS != "windows" {
		t.Fatalf("under Proton the folder is a Z: path, got %q", got)
	}
}

type orderExtension struct{}

func (orderExtension) Role() string                                    { return "test-order" }
func (orderExtension) Contribute(*launchplan.Plan, loader.ProfileView) {}
func (orderExtension) Prelaunch(loader.ProfileView) error              { return nil }
func (orderExtension) Order(loader.ProfileView) ([]loadorder.Row, error) {
	return nil, nil
}

var _ = folder.Extend(orderExtension{})

func TestOnlyAGameWithAnOrderingExtensionHasALoadOrder(t *testing.T) {
	plain := components.PathTemplate{Linux: "{install}/Mods", Windows: "{install}/Mods", Darwin: "{install}/Mods"}
	game := func(id string, paths map[string]components.PathTemplate) components.GameInfo {
		return components.GameInfo{
			ID: id, Name: id, Enabled: true, Marker: "G.exe", Deploy: "profile",
			Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods"}},
			Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "1"}},
			Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
			Paths:   paths,
		}
	}
	testcatalog.AddGames(t,
		game("plain-folder", map[string]components.PathTemplate{"mods": plain}),
		game("ordered-folder", map[string]components.PathTemplate{"mods": plain, "test-order": plain}))
	var l loader.OrderFor = folder.Loader{}
	if l.HasOrder("plain-folder") || !l.HasOrder("ordered-folder") {
		t.Fatalf("plain %v, ordered %v", l.HasOrder("plain-folder"), l.HasOrder("ordered-folder"))
	}
}
