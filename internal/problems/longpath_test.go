package problems

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/modfixture"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// pathOf is a path of exactly n characters below base.
func pathOf(t *testing.T, base string, n int) string {
	t.Helper()
	p := filepath.Join(base, "x")
	if pad := n - len(p); pad > 0 {
		p = filepath.Join(base, strings.Repeat("d", pad-1), "x")
	}
	if len(p) != n {
		t.Fatalf("cannot make a %d-character path below %s", n, base)
	}
	return p
}

func TestALongPathIsARowAndOneJustUnderIsNot(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "data")
	mods := []framework.Mod{{Key: "nexus-1-2", Name: "Pack"}, {Key: "nexus-3-4", Name: "Short"}}
	rows := longPathRows(mods, []readPath{
		{key: "nexus-1-2", path: pathOf(t, base, maxPath)},
		{key: "nexus-1-2", path: pathOf(t, base, maxPath+5)},
		{key: "nexus-3-4", path: pathOf(t, base, maxPath-1)},
		{folder: "loose", path: pathOf(t, base, maxPath+1), placed: true},
	})
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if r := rows[0]; r.Name != "Pack" || r.Files != 2 || r.Length != maxPath+5 || len(r.Longest) != maxPath+5 || r.Placed {
		t.Fatalf("the entry's row counts both files and names the longest: %+v", r)
	}
	if r := rows[1]; r.Key != "" || r.Name != "loose" || !r.Placed {
		t.Fatalf("a file no mod holds has no key and is named by its folder: %+v", r)
	}
	if (Result{LongPaths: rows}).Count() != 2 {
		t.Fatal("long paths are counted problems")
	}
}

func TestPathLengthCountsAsWindowsDoes(t *testing.T) {
	// U+1F600 is two UTF-16 code units and four bytes; é is one unit and two bytes.
	p := `C:\` + strings.Repeat("é", 100) + `\` + strings.Repeat("😀", 50)
	if n := pathLength(p); n != 204 {
		t.Fatalf("pathLength = %d, want 204", n)
	}
	if len(p) < maxPath {
		t.Fatalf("the path is %d bytes; the test needs it past the early return", len(p))
	}
}

func TestPlacedFilesAreReadBelowTheGameFolder(t *testing.T) {
	dir, root := filepath.Join("/", "profile"), filepath.Join("/", "game", "Mods")
	got := placedPaths(dir, map[string]string{
		"Mods/one/x.package": "one",
		"Tray/a.trayitem":    "tray",
	}, "Mods", root)
	want := []readPath{
		{key: "one", path: filepath.Join(root, "one", "x.package"), placed: true},
		{key: "tray", path: filepath.Join(dir, "Tray", "a.trayitem")},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("placedPaths = %+v, want %+v", got, want)
	}
	if got := placedPaths(dir, map[string]string{"BepInEx/plugins/a.dll": "a"}, "", ""); got[0].placed || got[0].path != filepath.Join(dir, "BepInEx", "plugins", "a.dll") {
		t.Fatalf("a loader that places nothing is read in the profile: %+v", got)
	}
}

func TestOnlyLongFilesOfTheModsFolderAreListedWithTheirMod(t *testing.T) {
	modsDir := t.TempDir()
	deep := filepath.Join(modsDir, "nexus-1-2", "Pack")
	long := pathOf(t, deep, maxPath+3)
	// A switched-off mod's folder starts with a dot, and the game never reads it.
	off := pathOf(t, filepath.Join(modsDir, "nexus-3-4", ".Off"), maxPath+3)
	for _, p := range []string{long, off, filepath.Join(modsDir, "loose", "short.json")} {
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := longModsFolderPaths(modsDir, []framework.Mod{{Key: "nexus-1-2", Folder: deep}})
	if len(got) != 1 || got[0].key != "nexus-1-2" || got[0].path != long {
		t.Fatalf("longModsFolderPaths = %+v", got)
	}
}

// A mod's folder can sit inside another's, as a pack's parts do, and a folder Mortar did not install is no mod's.
func TestAModsFolderFileIsHeldByTheNearestModFolderOrByNone(t *testing.T) {
	modsDir := t.TempDir()
	outer := filepath.Join(modsDir, "Pack")
	inner := filepath.Join(outer, "Parts", "Inner")
	files := []string{
		pathOf(t, filepath.Join(inner, "assets"), maxPath+1),
		pathOf(t, filepath.Join(outer, "shared"), maxPath+2),
		pathOf(t, filepath.Join(modsDir, "Unknown", "deep"), maxPath+3),
	}
	for _, p := range files {
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := longModsFolderPaths(modsDir, []framework.Mod{{Key: "nexus-1-2", Folder: outer}, {Key: "nexus-3-4", Folder: inner}})
	want := []readPath{
		{key: "nexus-3-4", path: files[0]},
		{key: "nexus-1-2", path: files[1]},
		{folder: "Unknown", path: files[2]},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("longModsFolderPaths = %+v, want %+v", got, want)
	}
}

// A folder game's files are read where the folder loader places them, so a deep game folder makes a row that the
// profile's own shorter path would not.
func TestAPlacedFileIsJudgedAtItsPathInTheGameFolder(t *testing.T) {
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	const id = "long-game"
	m.Games = append(slices.Clone(m.Games), components.GameInfo{
		ID: id, Name: "Long Game", Enabled: true, Marker: "G.exe", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods"}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "1"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
		Paths:   map[string]components.PathTemplate{"mods": {Windows: "{documents}/G/Mods", Linux: "{documents}/G/Mods", Darwin: "{documents}/G/Mods"}},
	})
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	c := components.NewClient(nil)
	c.SetManifest(m)
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })

	datadirtest.Use(t, t.TempDir())
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "G.exe"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders[id] = folder }); err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, id, "A")
	// The game's mods folder is wherever game.PathFor puts it, which on Linux is inside the game's Proton prefix, so the
	// home is padded until that folder is about 200 characters long.
	base := t.TempDir()
	short, err := game.PathFor(base, set.Get(), id, "", "mods")
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(base, strings.Repeat("h", max(200-len(short), 1)))
	root, err := game.PathFor(home, set.Get(), id, "", "mods")
	if err != nil {
		t.Fatal(err)
	}
	placedDir := filepath.Join(root, "pack")
	name := strings.Repeat("n", maxPath+5-len(placedDir)-len(".pkg")-1) + ".pkg"
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "pack.zip"), map[string]string{"pack/" + name: "x"})
	if _, err := profiles.InstallArchive(t.Context(), id, p.ID, zip); err != nil {
		t.Fatal(err)
	}
	dir, err := profiles.ProfileDir(id, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n := pathLength(filepath.Join(dir, "Mods", "pack", name)); n >= maxPath {
		t.Fatalf("the profile's own path is %d characters; the test needs it short", n)
	}

	defer func(was bool) { checkLongPaths = was }(checkLongPaths)
	checkLongPaths = true
	s := NewService(home, set, profiles, nil)
	mods, err := s.installed(id, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	rows := s.longPaths(id, p.ID, mods)
	if len(rows) != 1 || !rows[0].Placed || rows[0].Files != 1 || rows[0].Longest != filepath.Join(placedDir, name) || rows[0].Length != maxPath+5 {
		t.Fatalf("rows = %+v", rows)
	}
	checkLongPaths = false
	if rows := s.longPaths(id, p.ID, mods); len(rows) != 0 {
		t.Fatalf("a build that is not for Windows checks nothing: %+v", rows)
	}
}

// An answer that does not wait holds the rows the walk before found until the new walk ends, which is then announced;
// one that waits has the new rows, and a fingerprint is walked once.
func TestAnAnswerThatDoesNotWaitKeepsTheLastRowsUntilTheWalkEnds(t *testing.T) {
	defer func(was bool) { checkLongPaths = was }(checkLongPaths)
	checkLongPaths = true
	emitted := make(chan Changed, 2)
	s := &Service{long: map[string]*longPathWalk{}, Emit: func(name string, data any) {
		if c, ok := data.(Changed); ok && name == ChangedEvent {
			emitted <- c
		}
	}}
	first := s.longPathWalkOf("stardew", "A", "one", func() []LongPath { return []LongPath{{Key: "a"}} })
	<-first.done
	if got := s.withLongPaths(t.Context(), first, false, Result{}).LongPaths; len(got) != 1 {
		t.Fatalf("a finished walk's rows are in the answer: %+v", got)
	}
	release := make(chan struct{})
	walks := 0
	walk := func() []LongPath {
		walks++
		<-release
		return []LongPath{{Key: "a"}, {Key: "b"}}
	}
	w := s.longPathWalkOf("stardew", "A", "two", walk)
	if s.longPathWalkOf("stardew", "A", "two", walk) != w {
		t.Fatal("a fingerprint whose walk has started is not walked again")
	}
	if got := s.withLongPaths(t.Context(), w, false, Result{}).LongPaths; len(got) != 1 || got[0].Key != "a" {
		t.Fatalf("until the walk ends, the rows before stay: %+v", got)
	}
	close(release)
	if got := s.withLongPaths(t.Context(), w, true, Result{}).LongPaths; len(got) != 2 || walks != 1 {
		t.Fatalf("an answer that waits has the new rows: %+v after %d walks", got, walks)
	}
	if got := <-emitted; got != (Changed{Game: "stardew", Profile: "A"}) {
		t.Fatalf("emitted %+v", got)
	}
	select {
	case got := <-emitted:
		t.Fatalf("only the walk an answer went without is announced: %+v", got)
	default:
	}
}

// Problems, which the pre-Play check and the command line ask, waits for the walk, and so does the check's health
// point; ProblemsNoWait then has its rows without walking again.
func TestProblemsWaitsForTheLongPathWalk(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "A")
	modsDir, err := profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	long := pathOf(t, filepath.Join(modsDir, "Loose"), maxPath+2)
	if err := os.MkdirAll(filepath.Dir(long), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(long, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer func(was bool) { checkLongPaths = was }(checkLongPaths)
	checkLongPaths = true
	s := NewService(t.TempDir(), set, profiles, nil)
	s.meta = fakeMeta{}
	got, err := s.Problems(t.Context(), "stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.LongPaths) != 1 || got.LongPaths[0].Name != "Loose" || got.LongPaths[0].Longest != long {
		t.Fatalf("Problems = %+v", got.LongPaths)
	}
	if health, err := profiles.HealthHistory("stardew", p.ID); err != nil || len(health) != 1 || health[0].Problems != 1 {
		t.Fatalf("the check's health point counts the walk's row: %+v, %v", health, err)
	}
	walk := s.long["stardew/"+p.ID]
	got, err = s.ProblemsNoWait(t.Context(), "stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.LongPaths) != 1 || s.long["stardew/"+p.ID] != walk || walk.owed {
		t.Fatalf("ProblemsNoWait = %+v", got.LongPaths)
	}
}

// TestTimeTheLongPathWalkOfAnEightHundredModProfile times the long-path check of a redirected game's mods folder on the
// modfixture profile, shaped like a real 811-mod one (about 48,000 files), and the first answer of a check with and
// without waiting for it. It is opt-in: MORTAR_SCALE=1.
// MORTAR_SCALE_DATA names the data folder to build it in and keeps it, and a later run reuses the profile there, so
// it times a process that did not just write the files.
func TestTimeTheLongPathWalkOfAnEightHundredModProfile(t *testing.T) {
	if os.Getenv("MORTAR_SCALE") != "1" {
		t.Skip("set MORTAR_SCALE=1 to time the long-path walk of an 800-mod profile")
	}
	data := os.Getenv("MORTAR_SCALE_DATA")
	if data == "" {
		data = t.TempDir()
	}
	datadirtest.Use(t, data)
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	const gameID = "stardew"
	var id string
	if have, err := profiles.List(gameID); err == nil && len(have) > 0 {
		id = have[0].ID
	} else {
		at := time.Now()
		id = modfixture.Profile(t, profiles, modfixture.Real)
		t.Logf("fixture written in %v", time.Since(at).Round(time.Millisecond))
	}
	defer func(was bool) { checkLongPaths = was }(checkLongPaths)
	checkLongPaths = true
	s := NewService(t.TempDir(), set, profiles, nil)
	mods, err := s.installed(gameID, id)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 5 {
		at := time.Now()
		rows := s.longPaths(gameID, id, mods)
		t.Logf("check %d: %v over %d mods, %d rows", i+1, time.Since(at).Round(100*time.Microsecond), len(mods), len(rows))
	}
	// The first answer of a new process, whose check reads the caches on disk: as the window's loads get it, and as
	// Problems gives it, waiting for the walk as every check did before it was left to finish after the answer.
	for i := range 5 {
		for _, wait := range []bool{false, true} {
			framework.Forget()
			s := NewService(t.TempDir(), set, profiles, nil)
			s.meta = fakeMeta{}
			at := time.Now()
			if _, err := s.problems(t.Context(), gameID, id, wait); err != nil {
				t.Fatal(err)
			}
			t.Logf("first answer %d, waiting for the walk %v: %v", i+1, wait, time.Since(at).Round(time.Millisecond))
			s.mu.Lock()
			walk := s.long[gameID+"/"+id]
			s.mu.Unlock()
			<-walk.done
		}
	}
}
