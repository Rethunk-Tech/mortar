package savessvc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestServiceListsAndRestoresWithTempDataDirs(t *testing.T) {
	s, savesDir := backupService(t)
	writeFarm(t, savesDir, "Farm_1", "Sunny", "v1")
	if _, err := backup.Saves(savesDir, filepath.Join(mustData(t), "mortar", "backups", "stardew"), backup.DefaultKeep, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), backup.Cause{Profile: "p1", Kind: backup.KindUpdate}); err != nil {
		t.Fatal(err)
	}
	listed, err := s.ListBackups("stardew", "")
	if err != nil || len(listed) != 1 || listed[0].Profile != "p1" {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	if err := fsx.WriteFile(filepath.Join(savesDir, "Farm_1", "Farm_1"), []byte("v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreBackup("stardew", "", listed[0].Name, []string{"Farm_1"}); err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(filepath.Join(savesDir, "Farm_1", "Farm_1"))
	if err != nil || string(got) != "v1" {
		t.Fatalf("restored %q, %v", got, err)
	}
}

func TestServiceRefusesRestoreWhileBusy(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	s := &Service{scanners: map[string]*saves.Scanner{"stardew": {Dir: t.TempDir()}}, busy: func() bool { return true }}
	if err := s.RestoreBackup("stardew", "", "2026-01-01T00-00-00.000.zip", nil); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v", err)
	}
}

func mustData(t *testing.T) string {
	t.Helper()
	d := os.Getenv("XDG_DATA_HOME")
	if d == "" {
		t.Fatal("XDG_DATA_HOME")
	}
	return d
}

func writeFarm(t *testing.T, saves, folder, farm, body string) {
	t.Helper()
	dir := filepath.Join(saves, folder)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(dir, folder), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(dir, "SaveGameInfo"), []byte(`<Farmer><farmName>`+farm+`</farmName></Farmer>`), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCreateBackupIsManualPinnedAndOnlyThatSave(t *testing.T) {
	s, savesDir := backupService(t)
	writeFarm(t, savesDir, "Farm_1", "Sunny", "v1")
	writeFarm(t, savesDir, "Farm_2", "Rainy", "v2")
	if _, err := s.CreateBackup("stardew", "Farm_1"); err != nil {
		t.Fatal(err)
	}
	listed, err := s.ListBackups("stardew", "")
	if err != nil || len(listed) != 1 {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	b := listed[0]
	if b.Kind != backup.KindManual || !b.Pinned || len(b.Saves) != 1 || b.Saves[0].Folder != "Farm_1" {
		t.Fatalf("backup = %+v", b)
	}
	for _, folder := range []string{"", ".", "..", "../etc", "a/b", "Missing_123"} {
		if _, err := s.CreateBackup("stardew", folder); err == nil {
			t.Errorf("CreateBackup(%q) = nil, want an error", folder)
		}
	}
}

func TestCreateBackupUsesGameBackupLocationAndStillListsOldFolder(t *testing.T) {
	s, savesDir := backupService(t)
	writeFarm(t, savesDir, "Farm_1", "Sunny", "v1")
	if _, err := s.CreateBackup("stardew", "Farm_1"); err != nil {
		t.Fatal(err)
	}
	old, err := s.ListBackups("stardew", "")
	if err != nil || len(old) != 1 {
		t.Fatalf("default list = %+v, %v", old, err)
	}
	custom := filepath.Join(t.TempDir(), "backups")
	if _, err := s.settings.Update(func(cur *settings.Settings) {
		_ = settings.ApplyKeyGame(cur, "backupLocation", custom, "stardew")
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateBackup("stardew", "Farm_1"); err != nil {
		t.Fatal(err)
	}
	listed, err := s.ListBackups("stardew", "")
	if err != nil || len(listed) != 2 {
		t.Fatalf("list after custom = %+v, %v", listed, err)
	}
	names, err := os.ReadDir(filepath.Join(custom, "stardew"))
	if err != nil {
		t.Fatal(err)
	}
	zips := 0
	for _, n := range names {
		if filepath.Ext(n.Name()) == ".zip" {
			zips++
		}
	}
	if zips != 1 {
		t.Fatalf("custom dir %v", names)
	}
	if err := s.SetBackupPinned("stardew", old[0].Name, true); err != nil {
		t.Fatalf("pin a backup in the default folder: %v", err)
	}
	after, err := s.ListBackups("stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	pinned := 0
	for _, b := range after {
		if b.Name == old[0].Name && b.Pinned {
			pinned++
		}
	}
	if pinned != 1 {
		t.Fatalf("old backup not pinned: %+v", after)
	}
}

func TestOpenSaveFolderRefusesPathsOutsideSaves(t *testing.T) {
	s := &Service{scanners: map[string]*saves.Scanner{"stardew": {Dir: t.TempDir()}}}
	for _, folder := range []string{"", ".", "..", "../etc", "a/b", "Missing_123"} {
		if err := s.OpenSaveFolder("stardew", folder); err == nil {
			t.Errorf("OpenSaveFolder(%q) = nil, want an error", folder)
		}
	}
}

func TestSaveBackupsFiltersAndDeleteRefusesPinned(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	savesDir := filepath.Join(t.TempDir(), "Saves")
	writeFarm(t, savesDir, "Farm_1", "Sunny", "v1")
	writeFarm(t, savesDir, "Farm_2", "Rainy", "v1")
	dir := filepath.Join(mustData(t), "mortar", "backups", "stardew")
	t0 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if _, err := backup.Saves(savesDir, dir, backup.DefaultKeep, t0, backup.Cause{Kind: backup.KindLaunch}); err != nil {
		t.Fatal(err)
	}
	if _, err := backup.Folder(savesDir, dir, "Farm_2", backup.DefaultKeep, t0.Add(time.Hour), backup.Cause{Kind: backup.KindManual, Pinned: true}); err != nil {
		t.Fatal(err)
	}
	s := &Service{scanners: map[string]*saves.Scanner{"stardew": {Dir: savesDir}}}
	one, err := s.SaveBackups("stardew", "Farm_1")
	if err != nil || len(one) != 1 || one[0].Kind != backup.KindLaunch {
		t.Fatalf("Farm_1 = %+v, %v", one, err)
	}
	two, err := s.SaveBackups("stardew", "Farm_2")
	if err != nil || len(two) != 2 || two[0].Kind != backup.KindManual {
		t.Fatalf("Farm_2 = %+v, %v", two, err)
	}
	if err := s.DeleteBackup("stardew", two[0].Name); err == nil {
		t.Fatal("deleted a pinned backup")
	}
	if err := s.DeleteBackup("stardew", two[1].Name); err != nil {
		t.Fatal(err)
	}
	if left, _ := s.SaveBackups("stardew", "Farm_2"); len(left) != 1 {
		t.Fatalf("left = %+v", left)
	}
}

func TestListBackupsKeepsToTheGameAndProfile(t *testing.T) {
	s, savesDir := backupService(t)
	lc := filepath.Join(t.TempDir(), "Lethal Company")
	s.scanners["lethal-company"] = &saves.Scanner{Dir: lc}
	writeFarm(t, savesDir, "Farm_1", "Sunny", "v1")
	data, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	target, err := backup.TargetFor(data, s.settings.Get(), "stardew", nil)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i, c := range []backup.Cause{{Profile: "p1", Kind: backup.KindUpdate}, {Profile: "p2", Kind: backup.KindLaunch}, {Kind: backup.KindRestore}} {
		if _, err := backup.Saves(savesDir, target.Dir, backup.DefaultKeep, at.Add(time.Duration(i)*time.Hour), c); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := s.ListBackups("lethal-company", ""); err != nil || len(got) != 0 {
		t.Fatalf("another game's backups listed: %+v, %v", got, err)
	}
	if got, err := s.ListBackups("stardew", ""); err != nil || len(got) != 3 {
		t.Fatalf("stardew = %+v, %v", got, err)
	}
	got, err := s.ListBackups("stardew", "p1")
	if err != nil || len(got) != 2 || got[0].Kind != backup.KindRestore || got[1].Profile != "p1" {
		t.Fatalf("p1 = %+v, %v", got, err)
	}
}

func TestCreateBackupOfNoSaveMakesNone(t *testing.T) {
	s, savesDir := backupService(t)
	if err := os.MkdirAll(filepath.Join(savesDir, "Empty_1"), 0o750); err != nil {
		t.Fatal(err)
	}
	made, err := s.CreateBackup("stardew", "Empty_1")
	if err != nil || made {
		t.Fatalf("CreateBackup = %v, %v", made, err)
	}
	if got, err := s.ListBackups("stardew", ""); err != nil || len(got) != 0 {
		t.Fatalf("list = %+v, %v", got, err)
	}
}

func TestRestoreRotatesAgainstTheProfilesKeepCount(t *testing.T) {
	s, savesDir := backupService(t)
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	if s.profiles, err = profile.Open(items); err != nil {
		t.Fatal(err)
	}
	p, err := s.profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.profiles.SetOverride("stardew", p.ID, "saveBackupsKept", "1"); err != nil {
		t.Fatal(err)
	}
	writeFarm(t, savesDir, "Farm_1", "Sunny", "v1")
	if _, err := s.CreateBackup("stardew", "Farm_1"); err != nil {
		t.Fatal(err)
	}
	listed, err := s.ListBackups("stardew", "")
	if err != nil || len(listed) != 1 {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	for range 3 {
		if err := s.RestoreBackup("stardew", p.ID, listed[0].Name, nil); err != nil {
			t.Fatal(err)
		}
		// A new pre-restore zip only replaces a recent one when the saves changed since.
		future := time.Now().Add(time.Hour)
		if err := os.Chtimes(filepath.Join(savesDir, "Farm_1", "Farm_1"), future, future); err != nil {
			t.Fatal(err)
		}
	}
	after, err := s.ListBackups("stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	restores := 0
	for _, b := range after {
		if b.Kind == backup.KindRestore {
			restores++
		}
	}
	if restores != 1 {
		t.Fatalf("%d pre-restore backups kept, want the profile's 1: %+v", restores, after)
	}
}
