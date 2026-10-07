package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// stagedInstall stops an install where a kill would: the folder is laid out under its pending name, and profile.json
// is written only when record is set.
func stagedInstall(t *testing.T, s *Store, record bool) (dir, placed, final string) {
	t.Helper()
	p := mustCreate(t, s, "Farm")
	zip := buildZip(t, "mod.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	key, err := s.items.AddArchive("stardew", zip)
	if err != nil {
		t.Fatal(err)
	}
	cur, dir, err := s.readDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	placed, final, err = s.addToStaged("stardew", &cur, dir, key, Source{Kind: KindLocal, Name: "mod.zip"}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if placed == final || !exists(placed) || exists(final) {
		t.Fatalf("staged folder: placed %s, final %s", placed, final)
	}
	if record {
		if err := writeProfile(dir, cur); err != nil {
			t.Fatal(err)
		}
	}
	return dir, placed, final
}

func TestKillBeforeProfileWriteLeavesNoFolderBehind(t *testing.T) {
	t.Parallel()
	s := newEnv(t).Store
	dir, placed, final := stagedInstall(t, s, false)
	id := filepath.Base(dir)
	if err := s.Rebuild("stardew", id); err != nil {
		t.Fatal(err)
	}
	if exists(placed) || exists(final) {
		t.Fatalf("an install profile.json never recorded left %s or %s", placed, final)
	}
	left, _ := os.ReadDir(filepath.Join(dir, "mods"))
	for _, e := range left {
		t.Errorf("mods/ still holds %s", e.Name())
	}
}

func TestKillBeforeFinalRenameIsFinishedByRebuild(t *testing.T) {
	t.Parallel()
	s := newEnv(t).Store
	dir, placed, final := stagedInstall(t, s, true)
	if err := s.Rebuild("stardew", filepath.Base(dir)); err != nil {
		t.Fatal(err)
	}
	if exists(placed) || !exists(final) {
		t.Fatalf("after rebuild placed exists = %v, final exists = %v", exists(placed), exists(final))
	}
}

func TestKillBeforeSnapshotIsRepairedFromProfile(t *testing.T) {
	t.Parallel()
	s := newEnv(t).Store
	p := mustCreate(t, s, "Farm")
	zip := buildZip(t, "mod.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	res, err := s.InstallArchive("stardew", p.ID, zip)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := loadSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	key := res.Profile.Entries[len(res.Profile.Entries)-1].Key
	delete(snap.Folders, key)
	if err := writeSnapshot(dir, snap); err != nil {
		t.Fatal(err)
	}
	if err := s.Rebuild("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	after, err := loadSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := after.Folders[key]; !ok {
		t.Fatalf("snapshot still lacks %s: %v", key, after.Folders)
	}
	drift, err := s.ScanModsDrift("stardew", p.ID)
	if err != nil || len(drift) != 0 {
		t.Fatalf("drift after repair = %v, %v", drift, err)
	}
}
