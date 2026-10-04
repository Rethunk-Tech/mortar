package datasvc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

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
	testfs.WriteFile(t, src, "settings.json", `{"accent":"sand"}`)
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
