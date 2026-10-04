package datasvc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/testenv"
)

func TestMoveDataFolderRefusesWhileTheGameRuns(t *testing.T) {
	s := NewService(nil, nil, nil)
	s.Busy = func() bool { return true }
	if err := s.MoveDataFolder(t.TempDir()); !errors.Is(err, errGameRunning) {
		t.Fatalf("busy = %v", err)
	}
}

func TestMoveDataFolderRefusesWhileAnotherServiceIsBusy(t *testing.T) {
	s := NewService(nil, nil, nil, BusyFunc(func() bool { return true }))
	if err := s.MoveDataFolder(t.TempDir()); !errors.Is(err, errGameRunning) {
		t.Fatalf("busy = %v", err)
	}
}

func TestMoveDataFolderRelocatesWhenIdle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", home)
	t.Setenv("LOCALAPPDATA", home)
	items, profiles := testenv.Stores(t)
	src, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(src, "settings.json"), []byte(`{"accent":"sand"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewService(items, profiles, nil)
	restarted := false
	s.Restart = func() error {
		restarted = true
		return nil
	}
	dest := filepath.Join(t.TempDir(), "moved")
	if err := s.MoveDataFolder(dest); err != nil {
		t.Fatal(err)
	}
	if !restarted {
		t.Fatal("did not restart")
	}
	got, err := fsx.ReadFile(filepath.Join(dest, "settings.json"))
	if err != nil || string(got) != `{"accent":"sand"}` {
		t.Fatalf("moved = %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(src, "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("old remains: %v", err)
	}
}
