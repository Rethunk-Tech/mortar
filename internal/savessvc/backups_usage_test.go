package savessvc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backup"
	"github.com/Rethunk-AI/mortar/internal/saves"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

func TestBackupsUsageAndTrimKeepNewestPerSaveAndPinned(t *testing.T) {
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
	dir := filepath.Join(mustData(t), "mortar", "backups")
	day := func(d int) time.Time { return time.Date(2026, 7, d, 0, 0, 0, 0, time.UTC) }
	for _, d := range []int{1, 2} {
		if _, err := backup.Saves(savesDir, dir, settings.MaxBackupsKept, day(d), backup.Cause{Kind: backup.KindUpdate}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := backup.Folder(savesDir, dir, "Farm_1", settings.MaxBackupsKept, day(3), backup.Cause{Kind: backup.KindManual, Pinned: true}); err != nil {
		t.Fatal(err)
	}

	u, err := s.BackupsUsage("stardew")
	if err != nil || u.TotalBytes == 0 || len(u.PerSave) != 2 {
		t.Fatalf("usage = %+v, %v", u, err)
	}
	counts := map[string]int{}
	for _, p := range u.PerSave {
		counts[p.Save] = p.Count
	}
	if counts["Farm_1"] != 3 || counts["Farm_2"] != 2 {
		t.Fatalf("counts = %v", counts)
	}

	res, err := s.TrimBackups("stardew", 1)
	if err != nil || res.Removed != 1 || res.FreedBytes == 0 {
		t.Fatalf("trim = %+v, %v", res, err)
	}
	left, err := s.ListBackups()
	if err != nil || len(left) != 2 {
		t.Fatalf("left = %+v, %v", left, err)
	}
	if _, err := s.TrimBackups("stardew", 0); err == nil {
		t.Fatal("keep 0 must be refused")
	}
}
