package datasvc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestServiceCacheLocationAndKeepErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	items, profiles := testenv.Stores(t)
	addItem(t, items, "local-a", map[string]string{"m.bin": "mod"})
	cleared := false
	cleaned := false
	svc := NewService(items, profiles, nil)
	svc.OnClearCache = func() { cleared = true }
	svc.OnCleanup = func() { cleaned = true }

	if p := svc.UsageProgress(); p.Measuring || p.Copying {
		t.Fatalf("idle progress = %+v", p)
	}
	info, err := svc.CacheInfo()
	if err != nil || info.Path == "" {
		t.Fatalf("CacheInfo = %+v %v", info, err)
	}
	if err := os.MkdirAll(info.Path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(info.Path, "x.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	loc, err := svc.DataLocation()
	if err != nil || loc.Dir == "" || loc.Portable {
		t.Fatalf("DataLocation = %+v %v", loc, err)
	}

	first, err := svc.ModUsage()
	if err != nil {
		t.Fatal(err)
	}
	again, err := svc.ModUsage()
	if err != nil || again.Total != first.Total {
		t.Fatalf("ModUsage reuse = %+v vs %+v %v", again, first, err)
	}
	sizes, err := svc.EntrySizes()
	if err != nil || len(sizes) == 0 {
		t.Fatalf("EntrySizes = %v %v", sizes, err)
	}
	cached, err := svc.EntrySizes()
	if err != nil || len(cached) != len(sizes) {
		t.Fatalf("EntrySizes reuse = %v vs %v %v", cached, sizes, err)
	}

	if err := svc.ClearCache(); err != nil || !cleared {
		t.Fatalf("ClearCache = %v cleared=%v", err, cleared)
	}
	if _, err := os.Stat(filepath.Join(info.Path, "x.json")); !os.IsNotExist(err) {
		t.Fatalf("cache file remains: %v", err)
	}

	preview, err := svc.CleanupPreview()
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Cleanup(preview); err != nil || !cleaned {
		t.Fatalf("Cleanup = %v cleaned=%v", err, cleaned)
	}

	fail := errors.New("keep")
	broken := NewService(items, profiles, []KeySource{func() (map[string][]string, error) { return nil, fail }})
	if _, err := KeepSet(profiles, false, broken.keep); !errors.Is(err, fail) {
		t.Fatalf("KeepSet = %v", err)
	}
	if _, err := broken.CleanupPreview(); !errors.Is(err, fail) {
		t.Fatalf("CleanupPreview = %v", err)
	}
	if err := broken.Cleanup(Preview{}); !errors.Is(err, fail) {
		t.Fatalf("Cleanup = %v", err)
	}

	est, err := svc.MoveDataFolderPreview(t.TempDir())
	if err != nil || est.FreeBytes == 0 {
		t.Fatalf("MoveDataFolderPreview = %+v %v", est, err)
	}
	if err := svc.MoveDataFolder(""); err == nil || err.Error() != "no folder chosen" {
		t.Fatalf("empty dest = %v", err)
	}
	if err := NewService(nil, nil, nil, nil).MoveDataFolder(""); err == nil {
		t.Fatal("nil busy still needs a dest")
	}

	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ModUsage(); err != nil {
		t.Fatal(err)
	}
	_ = root
}
