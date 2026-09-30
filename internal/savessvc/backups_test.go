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
