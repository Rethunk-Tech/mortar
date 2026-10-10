//go:build !windows

package launchsvc

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/gamestore"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const (
	sims4        = "sims4"
	sims4AppID   = "1222670"
	resourceCfg  = "Priority 500\nPackedFile *.package\nPackedFile */*.package\n"
	playerOption = "[options]\r\nmodsdisabled = 1\r\nvolume = 7\r\n"
)

// sims4World is the real sims4 catalog entry (enabled in the test's copy of the catalog only) over a fake Steam library
// and Proton prefix under a temp home.
type sims4World struct {
	svc       *Service
	profiles  *profile.Store
	set       *settings.Store
	home      string
	game      string
	docs      string
	profileID string
}

func newSims4World(t *testing.T) *sims4World {
	t.Helper()
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = slices.Clone(m.Games)
	i := slices.IndexFunc(m.Games, func(g components.GameInfo) bool { return g.ID == sims4 })
	if i < 0 {
		t.Fatal("the catalog has no sims4")
	}
	m.Games[i].Enabled = true
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
	home := t.TempDir()
	root := filepath.Join(home, ".local", "share", "Steam")
	gameDir := filepath.Join(root, "steamapps", "common", "The Sims 4")
	docs := filepath.Join(root, "steamapps", "compatdata", sims4AppID, "pfx", "drive_c", "users", "steamuser", "Documents", "Electronic Arts", "The Sims 4")
	w := &sims4World{home: home, game: gameDir, docs: docs}
	w.write(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), "\"libraryfolders\"\n{\n\"0\"\n{\n\"path\" \""+root+"\"\n}\n}\n")
	w.write(t, filepath.Join(root, "steamapps", "appmanifest_"+sims4AppID+".acf"), "\"AppState\"\n{\n\"installdir\" \"The Sims 4\"\n}\n")
	w.write(t, filepath.Join(gameDir, "Game", "Bin", "TS4_x64.exe"), "exe")
	w.write(t, filepath.Join(docs, "Mods", "Resource.cfg"), resourceCfg)
	w.write(t, filepath.Join(docs, "Mods", "player.package"), "players own")
	w.write(t, filepath.Join(docs, "saves", "Slot_00000001.save"), "the player's save")
	w.write(t, filepath.Join(docs, "Options.ini"), playerOption)
	w.write(t, filepath.Join(docs, "localthumbcache.package"), "cache")
	w.write(t, filepath.Join(docs, "avatarcache.package"), "cache")

	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	profiles.TrayFolder = func(id string) (string, error) { return game.PathFor(home, set.Get(), id, "", "tray") }
	p := testenv.Profile(t, profiles, sims4, "A")
	w.svc, w.profiles, w.set, w.profileID = NewService(home, set, profiles), profiles, set, p.ID
	return w
}

func (w *sims4World) write(t *testing.T, p, body string) {
	t.Helper()
	writeFile(t, p, body)
}

func (w *sims4World) install(t *testing.T, fileID int, files map[string]string) profile.Profile {
	t.Helper()
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "a.zip"), files)
	res, err := w.profiles.InstallSource(t.Context(), sims4, w.profileID, zip, profile.Source{Kind: profile.KindCurseForge, Name: "7", FileID: fileID})
	if err != nil {
		t.Fatal(err)
	}
	return res.Profile
}

// snapshot is every file under the player's Mods and saves folders and the Options.ini, by path.
func (w *sims4World) snapshot(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, sub := range []string{"Mods", "saves", "Options.ini"} {
		err := filepath.WalkDir(filepath.Join(w.docs, sub), func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.Type().IsRegular() {
				return err
			}
			rel, _ := filepath.Rel(w.docs, p)
			out[rel] = readFile(t, p)
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	return out
}

func TestSims4RealEntryIsFoundFromSteamAndTheEAAppAndResolvesInsideTheProtonPrefix(t *testing.T) {
	w := newSims4World(t)
	g := game.Find(sims4)
	if g == nil {
		t.Fatal("sims4 is not a game")
	}
	all, selected := w.svc.installs(g)
	if len(all) != 1 || selected.Dir != w.game || selected.Store != game.StoreSteam {
		t.Fatalf("installs = %+v, selected %+v", all, selected)
	}
	for role, want := range map[string]string{
		"mods": "Mods", "saves": "saves", "options": "Options.ini", "tray": "Tray", "userData": "",
	} {
		got, err := game.PathFor(w.home, w.set.Get(), sims4, "", role)
		if err != nil || got != filepath.Join(w.docs, want) {
			t.Errorf("%s resolves to %q (%v), want %q", role, got, err, filepath.Join(w.docs, want))
		}
	}
	info, _ := components.Game(sims4)
	lib := t.TempDir()
	w.write(t, filepath.Join(lib, "The Sims 4", "Game", "Bin", "TS4_x64.exe"), "exe")
	found := gamestore.Discover(t.TempDir(), map[string][]string{gamestore.LauncherEA: {lib}}, info)
	if !slices.ContainsFunc(found, func(in gamestore.Install) bool {
		return in.Store == gamestore.StoreEA && in.Dir == filepath.Join(lib, "The Sims 4")
	}) {
		t.Fatalf("EA App layout not found: %+v", found)
	}
}

func TestSims4NewProfileStartsWithSeparateSaves(t *testing.T) {
	w := newSims4World(t)
	if !w.profiles.SeparateSaves(sims4, w.profileID) {
		t.Fatal("a new Sims 4 profile does not start with separate saves")
	}
}

func TestSims4LaunchPlacesOnlyEnabledFilesSwapsOptionsAndSavesAndGivesTheTreeBack(t *testing.T) {
	w := newSims4World(t)
	w.install(t, 1, map[string]string{"A/one.package": "1", "two.package": "2", "three.package": "3"})
	cur := w.install(t, 2, map[string]string{"m.ts4script": "s", "m.package": "mp"})
	w.install(t, 3, map[string]string{"House.trayitem": "t", "House.blueprint": "b"})
	if n := len(cur.Entries); n < 4 {
		t.Fatalf("entries after three installs = %d, want three packages and one script archive", n)
	}
	tray := filepath.Join(w.docs, "Tray")
	if !fileExists(filepath.Join(tray, "House.trayitem")) || !fileExists(filepath.Join(tray, "House.blueprint")) {
		t.Fatal("the household archive's Tray files were not placed")
	}
	for _, e := range cur.Entries {
		if e.File == "three.package" {
			if _, err := w.profiles.SetModEnabled(sims4, w.profileID, e.Key, e.Mods[0].ID, false); err != nil {
				t.Fatal(err)
			}
		}
	}
	before := w.snapshot(t)
	g := game.Find(sims4)
	_, inst := w.svc.installs(g)
	plan, err := w.svc.planProfile(t.Context(), g, inst, w.profileID, launchplan.ModeProfile, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	dep, err := w.svc.deployProfile(t.Context(), sims4, inst, w.profileID, plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.svc.swapSaves(t.Context(), sims4, inst, w.profileID, "", dep); err != nil {
		dep.unwind(t.Context())
		t.Fatal(err)
	}
	if err := w.svc.swapOptions(t.Context(), sims4, inst, w.profileID, "", dep); err != nil {
		dep.unwind(t.Context())
		t.Fatal(err)
	}
	mods := filepath.Join(w.docs, "Mods")
	for _, rel := range []string{"A/one.package", "two.package", "m.ts4script", "m.package", "Resource.cfg"} {
		if !fileExists(filepath.Join(mods, filepath.FromSlash(rel))) {
			t.Errorf("%s is not in the Mods folder during the launch", rel)
		}
	}
	if fileExists(filepath.Join(mods, "three.package")) {
		t.Error("a switched-off file was placed")
	}
	if got := readFile(t, filepath.Join(mods, "Resource.cfg")); got != resourceCfg {
		t.Errorf("Resource.cfg during the launch = %q", got)
	}
	options := readFile(t, filepath.Join(w.docs, "Options.ini"))
	if !strings.Contains(options, "modsdisabled = 0") || !strings.Contains(options, "scriptmodsenabled = 1") || !strings.Contains(options, "volume = 7") {
		t.Errorf("the game's Options.ini = %q", options)
	}
	if fileExists(filepath.Join(w.docs, "saves", "Slot_00000001.save")) {
		t.Error("the shared saves are visible to a profile with separate saves")
	}
	dep.unwind(t.Context())
	after := w.snapshot(t)
	for rel, body := range before {
		if after[rel] != body {
			t.Errorf("%s after the launch = %q, want %q", rel, after[rel], body)
		}
	}
	if len(after) != len(before) {
		t.Errorf("the tree after the launch has %d files, before %d: %v", len(after), len(before), after)
	}
	if !fileExists(filepath.Join(tray, "House.trayitem")) {
		t.Error("the Tray files, shared by every profile, were taken back")
	}
}

func TestSims4CachesAreClearedOnceWhenTheModSetChanges(t *testing.T) {
	w := newSims4World(t)
	w.install(t, 1, map[string]string{"one.package": "1"})
	thumb := filepath.Join(w.docs, "localthumbcache.package")
	w.svc.clearCaches(sims4, w.profileID, "")
	if fileExists(thumb) || fileExists(filepath.Join(w.docs, "avatarcache.package")) {
		t.Fatal("the caches survived the first launch after a mod change")
	}
	w.write(t, thumb, "rebuilt")
	w.svc.clearCaches(sims4, w.profileID, "")
	if !fileExists(thumb) {
		t.Fatal("the cache was cleared again with the mod set unchanged")
	}
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

func TestSims4DepthLimitsFollowTheCatalog(t *testing.T) {
	w := newSims4World(t)
	zip := func(name string, files map[string]string) string {
		return testfs.WriteZip(t, filepath.Join(t.TempDir(), name), files)
	}
	src := func(id int) profile.Source {
		return profile.Source{Kind: profile.KindCurseForge, Name: "9", FileID: id}
	}
	if _, err := w.profiles.InstallSource(t.Context(), sims4, w.profileID, zip("ok.zip", map[string]string{"a/b/c/d/e/deep.package": "1"}), src(1)); err != nil {
		t.Fatalf("a package five folders deep was refused: %v", err)
	}
	if _, err := w.profiles.InstallSource(t.Context(), sims4, w.profileID, zip("deep.zip", map[string]string{"a/b/c/d/e/f/toodeep.package": "1"}), src(2)); err == nil {
		t.Error("a package six folders deep was accepted")
	}
	if _, err := w.profiles.InstallSource(t.Context(), sims4, w.profileID, zip("script.zip", map[string]string{"Mod/m.ts4script": "s"}), src(3)); err != nil {
		t.Fatalf("a script one folder deep was refused: %v", err)
	}
	if _, err := w.profiles.InstallSource(t.Context(), sims4, w.profileID, zip("script2.zip", map[string]string{"Mod/Inner/m.ts4script": "s"}), src(4)); err == nil {
		t.Error("a script two folders deep was accepted")
	}
}

func TestSims4WarnModeLeavesTheOptionsSwitchesAlone(t *testing.T) {
	w := newSims4World(t)
	if _, err := w.set.Update(func(v *settings.Settings) {
		gp := v.GamePrefs(sims4)
		gp.GameSettingsMode = settings.GameSettingsWarn
		v.Games = map[string]*settings.GameSettings{sims4: &gp}
	}); err != nil {
		t.Fatal(err)
	}
	before := w.snapshot(t)
	g := game.Find(sims4)
	_, inst := w.svc.installs(g)
	dep := &deployment{}
	if err := w.svc.swapOptions(t.Context(), sims4, inst, w.profileID, "", dep); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(w.docs, "Options.ini")); got != playerOption {
		t.Fatalf("the game's Options.ini in warn mode = %q, want the player's bytes", got)
	}
	if got := readFile(t, dep.options.Profile); got != playerOption {
		t.Fatalf("the profile copy in warn mode = %q", got)
	}
	dep.unwind(t.Context())
	if got := w.snapshot(t); got["Options.ini"] != before["Options.ini"] {
		t.Fatalf("the player's Options.ini after the launch = %q", got["Options.ini"])
	}
}
