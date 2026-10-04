package savessvc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backup"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/saves"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

func TestServiceListsAndRestoresWithTempDataDirs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	savesDir := filepath.Join(cfg, "StardewValley", "Saves")
	writeFarm(t, savesDir, "Farm_1", "Sunny", "v1")
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{settings: store, scanner: &saves.Scanner{Dir: savesDir}}
	if _, err := backup.Saves(savesDir, filepath.Join(mustData(t), "mortar", "backups"), backup.DefaultKeep, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), backup.Cause{Profile: "p1", Kind: backup.KindUpdate}); err != nil {
		t.Fatal(err)
	}
	listed, err := s.ListBackups()
	if err != nil || len(listed) != 1 || listed[0].Profile != "p1" {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	if err := fsx.WriteFile(filepath.Join(savesDir, "Farm_1", "Farm_1"), []byte("v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreBackup(listed[0].Name, []string{"Farm_1"}); err != nil {
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
	s := &Service{scanner: &saves.Scanner{Dir: t.TempDir()}, busy: func() bool { return true }}
	if err := s.RestoreBackup("2026-01-01T00-00-00.000.zip", nil); !errors.Is(err, ErrBusy) {
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
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	savesDir := filepath.Join(cfg, "StardewValley", "Saves")
	writeFarm(t, savesDir, "Farm_1", "Sunny", "v1")
	writeFarm(t, savesDir, "Farm_2", "Rainy", "v2")
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{settings: store, scanner: &saves.Scanner{Dir: savesDir}}
	if err := s.CreateBackup("Farm_1"); err != nil {
		t.Fatal(err)
	}
	listed, err := s.ListBackups()
	if err != nil || len(listed) != 1 {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	b := listed[0]
	if b.Kind != backup.KindManual || !b.Pinned || len(b.Saves) != 1 || b.Saves[0].Folder != "Farm_1" {
		t.Fatalf("backup = %+v", b)
	}
	for _, folder := range []string{"", ".", "..", "../etc", "a/b", "Missing_123"} {
		if err := s.CreateBackup(folder); err == nil {
			t.Errorf("CreateBackup(%q) = nil, want an error", folder)
		}
	}
}

func TestCreateBackupUsesGameBackupLocationAndStillListsOldFolder(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	savesDir := filepath.Join(cfg, "StardewValley", "Saves")
	writeFarm(t, savesDir, "Farm_1", "Sunny", "v1")
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{settings: store, scanner: &saves.Scanner{Dir: savesDir}}
	if err := s.CreateBackup("Farm_1"); err != nil {
		t.Fatal(err)
	}
	old, err := s.ListBackups()
	if err != nil || len(old) != 1 {
		t.Fatalf("default list = %+v, %v", old, err)
	}
	custom := filepath.Join(t.TempDir(), "backups")
	if _, err := store.Update(func(cur *settings.Settings) {
		_ = settings.ApplyKeyGame(cur, "backupLocation", custom, settings.GameStardew)
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBackup("Farm_1"); err != nil {
		t.Fatal(err)
	}
	listed, err := s.ListBackups()
	if err != nil || len(listed) != 2 {
		t.Fatalf("list after custom = %+v, %v", listed, err)
	}
	names, err := os.ReadDir(custom)
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
}

func TestOpenSaveFolderRefusesPathsOutsideSaves(t *testing.T) {
	s := &Service{scanner: &saves.Scanner{Dir: t.TempDir()}}
	for _, folder := range []string{"", ".", "..", "../etc", "a/b", "Missing_123"} {
		if err := s.OpenSaveFolder(folder); err == nil {
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
	dir := filepath.Join(mustData(t), "mortar", "backups")
	t0 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if _, err := backup.Saves(savesDir, dir, backup.DefaultKeep, t0, backup.Cause{Kind: backup.KindLaunch}); err != nil {
		t.Fatal(err)
	}
	if _, err := backup.Folder(savesDir, dir, "Farm_2", backup.DefaultKeep, t0.Add(time.Hour), backup.Cause{Kind: backup.KindManual, Pinned: true}); err != nil {
		t.Fatal(err)
	}
	s := &Service{scanner: &saves.Scanner{Dir: savesDir}}
	one, err := s.SaveBackups("Farm_1")
	if err != nil || len(one) != 1 || one[0].Kind != backup.KindLaunch {
		t.Fatalf("Farm_1 = %+v, %v", one, err)
	}
	two, err := s.SaveBackups("Farm_2")
	if err != nil || len(two) != 2 || two[0].Kind != backup.KindManual {
		t.Fatalf("Farm_2 = %+v, %v", two, err)
	}
	if err := s.DeleteBackup(two[0].Name); err == nil {
		t.Fatal("deleted a pinned backup")
	}
	if err := s.DeleteBackup(two[1].Name); err != nil {
		t.Fatal(err)
	}
	if left, _ := s.SaveBackups("Farm_2"); len(left) != 1 {
		t.Fatalf("left = %+v", left)
	}
}
