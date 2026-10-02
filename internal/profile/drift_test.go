package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestScanModsDriftOnProfile(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
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
	e := newEnv(t)
	p, err := e.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
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
	s := newEnv(t)
	p, err := s.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
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
