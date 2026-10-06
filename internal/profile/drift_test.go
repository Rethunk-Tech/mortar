package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func writeTimed(t *testing.T, path, body string, when time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if !when.IsZero() {
		if err := os.Chtimes(path, when, when); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanDriftUnknownDeletedChanged(t *testing.T) {
	t.Parallel()
	mods := t.TempDir()
	when := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	writeTimed(t, filepath.Join(mods, "keep-me", "manifest.json"), "a", when)
	writeTimed(t, filepath.Join(mods, "changed", "manifest.json"), "old", when)
	keepStat, err := walkFolderStat(filepath.Join(mods, "keep-me"), "")
	if err != nil {
		t.Fatal(err)
	}
	changeStat, err := walkFolderStat(filepath.Join(mods, "changed"), "")
	if err != nil {
		t.Fatal(err)
	}
	snap := ModsSnapshot{Folders: map[string]FolderStat{
		"keep-me": keepStat,
		"changed": changeStat,
		"gone":    {Files: 1, Size: 1, Newest: 1},
	}}

	writeTimed(t, filepath.Join(mods, "changed", "manifest.json"), "new", when.Add(time.Hour))
	writeTimed(t, filepath.Join(mods, "dropped", "manifest.json"), "x", when)

	names, err := liveFolders(mods, "")
	if err != nil {
		t.Fatal(err)
	}
	stats := map[string]FolderStat{}
	for key, folder := range names {
		st, err := folderStatAt(mods, "", folder, "")
		if err != nil {
			t.Fatal(err)
		}
		stats[key] = st
	}
	got := scanDrift(names, stats, []string{"keep-me", "changed", "gone"}, snap)
	kinds := map[DriftKind]string{}
	for _, d := range got {
		kinds[d.Kind] = d.Key
	}
	if kinds[DriftUnknown] != "dropped" || kinds[DriftDeleted] != "gone" || kinds[DriftChanged] != "changed" {
		t.Fatalf("got %#v", got)
	}
	if _, ok := kinds[DriftKind("keep")]; ok {
		t.Fatal("keep-me reported")
	}
	for _, d := range got {
		if d.Key == "keep-me" {
			t.Fatalf("unchanged entry reported: %#v", d)
		}
	}
}

func TestScanDriftIgnoresConfigJSON(t *testing.T) {
	t.Parallel()
	mods := t.TempDir()
	when := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	writeTimed(t, filepath.Join(mods, "mod", "manifest.json"), "a", when)
	st, err := walkFolderStat(filepath.Join(mods, "mod"), "")
	if err != nil {
		t.Fatal(err)
	}
	writeTimed(t, filepath.Join(mods, "mod", "config.json"), `{"x":1}`, when.Add(time.Hour))
	after, err := walkFolderStat(filepath.Join(mods, "mod"), "")
	if err != nil {
		t.Fatal(err)
	}
	if after != st {
		t.Fatalf("config.json counted as an outside edit: %+v vs %+v", st, after)
	}
	names, err := liveFolders(mods, "")
	if err != nil {
		t.Fatal(err)
	}
	got := scanDrift(names, map[string]FolderStat{"mod": after}, []string{"mod"}, ModsSnapshot{Folders: map[string]FolderStat{"mod": st}})
	if len(got) != 0 {
		t.Fatalf("config-only change: %#v", got)
	}
}

func TestScanDriftIgnoresModDataWrites(t *testing.T) {
	t.Parallel()
	mods := t.TempDir()
	when := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	writeTimed(t, filepath.Join(mods, "mod", "manifest.json"), "a", when)
	st, err := walkFolderStat(filepath.Join(mods, "mod"), "")
	if err != nil {
		t.Fatal(err)
	}
	writeTimed(t, filepath.Join(mods, "mod", "data", "x.json"), `{"x":1}`, when.Add(time.Hour))
	after, err := walkFolderStat(filepath.Join(mods, "mod"), "")
	if err != nil {
		t.Fatal(err)
	}
	if after != st {
		t.Fatalf("data/ counted as an outside edit: %+v vs %+v", st, after)
	}
}

func TestScanDriftLinkedShippedFileUnchanged(t *testing.T) {
	t.Parallel()
	mods := t.TempDir()
	store := t.TempDir()
	when := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	writeTimed(t, filepath.Join(store, "manifest.json"), "a", when)
	if err := os.MkdirAll(filepath.Join(mods, "mod"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(store, "manifest.json"), filepath.Join(mods, "mod", "manifest.json")); err != nil {
		t.Fatal(err)
	}
	st, err := walkFolderStat(filepath.Join(mods, "mod"), store)
	if err != nil {
		t.Fatal(err)
	}
	after, err := walkFolderStat(filepath.Join(mods, "mod"), store)
	if err != nil {
		t.Fatal(err)
	}
	if after != st {
		t.Fatalf("hardlink counted as modified: %+v vs %+v", st, after)
	}
	names, err := liveFolders(mods, "")
	if err != nil {
		t.Fatal(err)
	}
	got := scanDrift(names, map[string]FolderStat{"mod": after}, []string{"mod"}, ModsSnapshot{Folders: map[string]FolderStat{"mod": st}})
	if len(got) != 0 {
		t.Fatalf("linked shipped file: %#v", got)
	}
}

func TestScanModsDriftOnProfile(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	dir, err := s.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	mods := filepath.Join(dir, "mods")
	when := time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)
	writeTimed(t, filepath.Join(mods, "local-one", "manifest.json"), "one", when)
	p.Entries = []Entry{{Key: "local-one"}}
	if err := writeProfile(dir, p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScanModsDrift("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	writeTimed(t, filepath.Join(mods, "loose", "manifest.json"), "x", when)
	if err := os.RemoveAll(filepath.Join(mods, "local-one")); err != nil {
		t.Fatal(err)
	}
	writeTimed(t, filepath.Join(mods, "local-two", "manifest.json"), "two", when)
	p.Entries = []Entry{{Key: "local-one"}, {Key: "local-two"}}
	if err := writeProfile(dir, p); err != nil {
		t.Fatal(err)
	}
	if err := writeSnapshot(dir, ModsSnapshot{Folders: map[string]FolderStat{
		"local-one": {Files: 1, Size: 3, Newest: when.UnixNano()},
		"local-two": {Files: 1, Size: 3, Newest: when.UnixNano()},
	}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ScanModsDrift("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[DriftKind]bool{}
	for _, d := range got {
		seen[d.Kind] = true
	}
	if !seen[DriftUnknown] || !seen[DriftDeleted] {
		t.Fatalf("profile scan = %#v", got)
	}
}

func TestInstallThenScanReportsNoDrift(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	zip := buildZip(t, "mod.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	if _, err := e.InstallArchive("stardew", p.ID, zip); err != nil {
		t.Fatal(err)
	}
	got, err := e.ScanModsDrift("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("after Mortar install: %#v", got)
	}
}

func TestParkUnknownFoldersKeepsEarlierCopy(t *testing.T) {
	t.Parallel()
	s := newEnv(t)
	p := mustCreate(t, s, "Farm")
	writeFile(t, s.mods(p.ID), "dropped/first.txt", "first")
	if _, err := s.Mods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	writeFile(t, s.mods(p.ID), "dropped/second.txt", "second")
	if _, err := s.UserMods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}

	hold := filepath.Join(s.root, "stardew", p.ID, modsHoldDir)
	if got := read(t, filepath.Join(hold, "dropped", "first.txt")); got != "first" {
		t.Fatalf("first parked folder = %q", got)
	}
	if got := read(t, filepath.Join(hold, "dropped-2", "second.txt")); got != "second" {
		t.Fatalf("second parked folder = %q", got)
	}
}

func TestSwitchingAModOffIsNotDrift(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	zip := buildZip(t, "pack.zip", map[string]string{
		"A/manifest.json": manifestJSON("X.A"),
		"A/assets/a.png":  "a",
		"B/manifest.json": manifestJSON("X.B"),
	})
	if _, err := e.InstallArchive("stardew", p.ID, zip); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ScanModsDrift("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "", "smapi:X.A", false); err != nil {
		t.Fatal(err)
	}
	got, err := e.ScanModsDrift("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("after switching a mod off: %#v", got)
	}
}

func TestRefreshDependenciesReadsOptionalFromStoreManifest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	zip := buildZip(t, "pack.zip", map[string]string{
		"A/manifest.json": `{"UniqueID":"X.A","Name":"A","Version":"1.0","Dependencies":[{"UniqueID":"X.Opt","IsRequired":"false"}]}`,
	})
	if _, err := e.InstallArchive("stardew", p.ID, zip); err != nil {
		t.Fatal(err)
	}
	if _, err := e.update("stardew", p.ID, func(p *Profile, _ string) error {
		p.Entries[0].Mods[0].Optional = nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := e.RefreshDependencies("stardew"); err != nil {
		t.Fatal(err)
	}
	got, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if opt := got.Entries[0].Mods[0].Optional; len(opt) != 1 || opt[0] != "smapi:X.Opt" {
		t.Fatalf("Optional = %v, want [X.Opt]", opt)
	}
}

func TestRestoreModsFolderUndoesTrash(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	dir, _ := e.profileDir("stardew", p.ID)
	im := filepath.Join(dir, "mods", "Loose")
	if err := os.MkdirAll(im, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(im, "a.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	token, err := e.trashModsFolder("stardew", p.ID, "Loose")
	if err != nil || token == "" {
		t.Fatalf("trash = %q, %v", token, err)
	}
	if _, err := os.Stat(im); err == nil {
		t.Fatal("folder still in mods")
	}
	if err := os.MkdirAll(im, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := e.RestoreModsFolder("stardew", p.ID, token); err == nil {
		t.Fatal("restore over a taken name accepted")
	}
	if err := os.RemoveAll(im); err != nil {
		t.Fatal(err)
	}
	e.Running = func(string, string) bool { return true }
	if err := e.RestoreModsFolder("stardew", p.ID, token); err == nil {
		t.Fatal("restore while running accepted")
	}
	e.Running = nil
	if err := e.RestoreModsFolder("stardew", p.ID, "../x"); err == nil {
		t.Fatal("bad token accepted")
	}
	if err := e.RestoreModsFolder("stardew", p.ID, token); err != nil {
		t.Fatal(err)
	}
	if b, err := fsx.ReadFile(filepath.Join(im, "a.txt")); err != nil || string(b) != "x" {
		t.Fatalf("restored file = %q, %v", b, err)
	}
}

func TestListStoreItemMatchesSeparateWalks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	when := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	writeTimed(t, filepath.Join(root, "manifest.json"), "{}", when)
	writeTimed(t, filepath.Join(root, "assets", "a.png"), "png", when.Add(time.Hour))
	writeTimed(t, filepath.Join(root, "config.json"), "user", when.Add(2*time.Hour))
	got, err := listStoreItem(root)
	if err != nil {
		t.Fatal(err)
	}
	files, err := relFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := walkFolderStat(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.files) != len(files) || got.stat != st {
		t.Fatalf("one walk = %v %+v, separate walks = %v %+v", got.files, got.stat, files, st)
	}
	for f := range files {
		if _, ok := got.files[f]; !ok {
			t.Fatalf("one walk is missing %s", f)
		}
	}
}

// A store item is listed once: the listing survives a restart until the item's folder is replaced.
func TestStoreListingKeptAcrossStarts(t *testing.T) {
	testfs.DataHome(t)
	reset := func() {
		storeListings.Lock()
		storeListings.byPath, storeListings.loaded, storeListings.dirty = map[string]storeListing{}, false, false
		storeListings.Unlock()
	}
	reset()
	t.Cleanup(reset)
	peer := t.TempDir()
	writeTimed(t, filepath.Join(peer, "sub", "a.dll"), "a", time.Time{})
	first, err := storeListingFor(peer)
	if err != nil {
		t.Fatal(err)
	}
	saveStoreListings()
	reset()
	// Removing a nested file leaves the item folder's own mtime alone, so only a kept listing still names it.
	if err := os.Remove(filepath.Join(peer, "sub", "a.dll")); err != nil {
		t.Fatal(err)
	}
	got, err := storeListingFor(peer)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.files[filepath.Join("sub", "a.dll")]; !ok || got.stat != first.stat {
		t.Fatalf("after a restart the item was listed again: %+v", got)
	}
}

func TestRefreshDependenciesReadsAPackagesThunderstoreManifest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
		"manifest.json": `{"name":"Mod","version_number":"1.0.0","dependencies":["BepInEx-BepInExPack-5.4.2100","Ns-Lib-2.0.1"]}`, "Mod.dll": "x",
	})
	res, err := e.InstallSource("lethal-company", p.ID, zip, Source{Kind: KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	want := []mod.ID{"thunderstore:Ns-Lib"}
	if got := res.Profile.Entries[0].Mods[0].Needs; !slices.Equal(got, want) {
		t.Fatalf("Needs at install = %v, want %v", got, want)
	}
	if _, err := e.update("lethal-company", p.ID, func(p *Profile, _ string) error {
		p.Entries[0].Mods[0].Needs = nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := e.RefreshDependencies("lethal-company"); err != nil {
		t.Fatal(err)
	}
	got, err := e.read("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if needs := got.Entries[0].Mods[0].Needs; !slices.Equal(needs, want) {
		t.Fatalf("Needs after refresh = %v, want %v", needs, want)
	}
}

func TestFolderStatKeptListingsStillSeeEditsAndNewFiles(t *testing.T) {
	testfs.DataHome(t)
	reset := func() {
		dirListings.Lock()
		dirListings.byPath, dirListings.used, dirListings.loaded, dirListings.dirty = map[string]dirListing{}, map[string]bool{}, false, false
		dirListings.Unlock()
	}
	reset()
	t.Cleanup(reset)
	root := t.TempDir()
	old := time.Now().Add(-time.Hour)
	writeTimed(t, filepath.Join(root, "sub", "a.json"), "a", old)
	for _, dir := range []string{filepath.Join(root, "sub"), root} {
		if err := os.Chtimes(dir, old, old); err != nil {
			t.Fatal(err)
		}
	}
	first, err := walkFolderStat(root, "")
	if err != nil {
		t.Fatal(err)
	}
	saveDirListings()
	reset()

	// An edit in place leaves every folder's time alone; the file's own stat still shows it.
	writeTimed(t, filepath.Join(root, "sub", "a.json"), "edited", old.Add(time.Minute))
	edited, err := walkFolderStat(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if edited.Files != 1 || edited.Size == first.Size || edited.Newest == first.Newest {
		t.Fatalf("in-place edit unseen: %+v, was %+v", edited, first)
	}

	writeTimed(t, filepath.Join(root, "sub", "b.json"), "b", old)
	added, err := walkFolderStat(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if added.Files != 2 {
		t.Fatalf("new nested file unseen: %+v", added)
	}
}
