package savessvc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestScheduledBackupsFollowTheIntervalAndWaitForTheGame(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	savesDir := filepath.Join(t.TempDir(), "Saves")
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(v *settings.Settings) {
		for k, val := range map[string]string{"saveBackupHours": "6", "saveBackupKeep": "2"} {
			if err := settings.ApplyKeyGame(v, k, val, settings.GameStardew); err != nil {
				t.Fatal(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	busy := false
	s := &Service{settings: store, scanner: &saves.Scanner{Dir: savesDir}, busy: func() bool { return busy }}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	save := func(at time.Time) {
		writeFarm(t, savesDir, "Farm_1", "Sunny", at.String())
		dir := filepath.Join(savesDir, "Farm_1")
		for _, p := range []string{filepath.Join(dir, "Farm_1"), filepath.Join(dir, "SaveGameInfo"), dir} {
			if err := os.Chtimes(p, at, at); err != nil {
				t.Fatal(err)
			}
		}
	}
	scheduled := func() int {
		_, dir, err := s.backupDirs()
		if err != nil {
			t.Fatal(err)
		}
		listed, err := backup.List(dir)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, b := range listed {
			if b.Kind == backup.KindScheduled {
				n++
			}
		}
		return n
	}

	var runs []ScheduledRun
	s.Emit = func(name string, data any) {
		if run, ok := data.(ScheduledRun); ok && name == ScheduledEvent {
			runs = append(runs, run)
		}
	}
	save(t0.Add(-time.Hour))
	s.scheduledTick(t0)
	if n := scheduled(); n != 1 {
		t.Fatalf("first pass made %d", n)
	}
	if len(runs) != 1 || runs[0].Saved != 1 || runs[0].Error != "" || runs[0].At != t0.UnixMilli() {
		t.Fatalf("event = %+v", runs)
	}
	if last, err := s.LastScheduledBackup(); err != nil || last != t0.UnixMilli() {
		t.Fatalf("last = %d, %v", last, err)
	}
	save(t0.Add(time.Hour))
	s.scheduledTick(t0.Add(5 * time.Hour))
	if n := scheduled(); n != 1 {
		t.Fatalf("a pass inside the interval made %d", n)
	}
	busy = true
	s.scheduledTick(t0.Add(7 * time.Hour))
	if n := scheduled(); n != 1 {
		t.Fatalf("a pass while the game runs made %d", n)
	}
	busy = false
	s.scheduledTick(t0.Add(7*time.Hour + 10*time.Minute))
	if n := scheduled(); n != 2 {
		t.Fatalf("the pass after the game closed made %d in all", n)
	}

	restarted := &Service{settings: store, scanner: &saves.Scanner{Dir: savesDir}, busy: func() bool { return false }}
	save(t0.Add(8 * time.Hour))
	restarted.scheduledTick(t0.Add(9 * time.Hour))
	if n := scheduled(); n != 2 {
		t.Fatalf("a restart reset the schedule: %d", n)
	}
	restarted.scheduledTick(t0.Add(14 * time.Hour))
	if n := scheduled(); n != 2 {
		t.Fatalf("keep 2 per save, got %d", n)
	}
}
