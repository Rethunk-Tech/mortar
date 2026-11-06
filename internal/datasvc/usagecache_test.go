package datasvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestUsageIsReusedUntilFreshOrForgotten(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, profiles := testenv.Stores(t)
	svc := NewService(items, profiles, nil)
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.Usage(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "backups"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backups", "b.zip"), make([]byte, 1<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	if again, _ := svc.Usage(false); again.Total != first.Total {
		t.Fatalf("not reused: %d vs %d", again.Total, first.Total)
	}
	if fresh, _ := svc.Usage(true); fresh.Total <= first.Total {
		t.Fatalf("fresh did not re-measure: %d vs %d", fresh.Total, first.Total)
	}
}

func TestLastUsageIsTheLastFinishedMeasurement(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, profiles := testenv.Stores(t)
	svc := NewService(items, profiles, nil)
	if got := svc.LastUsage(); got != nil {
		t.Fatalf("before any walk: %+v", got)
	}
	measured, err := svc.Usage(false)
	if err != nil {
		t.Fatal(err)
	}
	// A new Service is a restart: it has no walk of its own but reads the stored one without measuring.
	got := NewService(items, profiles, nil).LastUsage()
	if got == nil || got.At == 0 || got.Usage.Path != measured.Path || got.Usage.Total != measured.Total {
		t.Fatalf("LastUsage = %+v, want the walk %+v", got, measured)
	}
}
