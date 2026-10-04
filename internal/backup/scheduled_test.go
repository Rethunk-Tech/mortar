package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func writeSave(t *testing.T, saves, folder string, at time.Time) {
	t.Helper()
	dir := filepath.Join(saves, folder)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, folder)
	if err := fsx.WriteFile(p, []byte(at.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, x := range []string{p, dir} {
		if err := os.Chtimes(x, at, at); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScheduledBacksUpChangedSavesAndRotatesEachSaveApart(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	out := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	writeSave(t, saves, "A_1", start.Add(-time.Hour))
	writeSave(t, saves, "B_2", start.Add(-time.Hour))
	if _, err := Saves(saves, out, 5, start.Add(-30*time.Minute), Cause{Kind: KindLaunch}); err != nil {
		t.Fatal(err)
	}

	run, err := Scheduled(saves, out, 2, start)
	if err != nil || run != (Run{Saved: 2}) {
		t.Fatalf("first run = %+v %v", run, err)
	}
	run, err = Scheduled(saves, out, 2, start.Add(time.Hour))
	if err != nil || run != (Run{Unchanged: 2}) {
		t.Fatalf("unchanged run = %+v %v", run, err)
	}
	for i := range 3 {
		at := start.Add(time.Duration(i+2) * time.Hour)
		writeSave(t, saves, "A_1", at.Add(-time.Minute))
		if run, err = Scheduled(saves, out, 2, at); err != nil || run != (Run{Saved: 1, Unchanged: 1}) {
			t.Fatalf("run %d = %+v %v", i, run, err)
		}
	}

	listed, err := List(out)
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	for _, b := range listed {
		c := readCause(filepath.Join(out, b.Name))
		count[c.Kind+":"+c.Save]++
	}
	want := map[string]int{"launch:": 1, "scheduled:A_1": 2, "scheduled:B_2": 1}
	if len(count) != len(want) {
		t.Fatalf("kept %v, want %v", count, want)
	}
	for k, n := range want {
		if count[k] != n {
			t.Fatalf("kept %v, want %v", count, want)
		}
	}
}

func TestSavesStandInIgnoresSingleSaveBackups(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	out := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	writeSave(t, saves, "A_1", start.Add(-time.Hour))
	if _, err := Scheduled(saves, out, 5, start); err != nil {
		t.Fatal(err)
	}
	got, err := Saves(saves, out, 5, start.Add(time.Minute), Cause{Kind: KindLaunch})
	if err != nil {
		t.Fatal(err)
	}
	if readCause(got).Kind != KindLaunch {
		t.Fatalf("before-Play backup stood in with %s", got)
	}
}

func TestScheduledIgnoresBackupsStampedInTheFuture(t *testing.T) {
	saves := filepath.Join(t.TempDir(), "Saves")
	out := t.TempDir()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	writeSave(t, saves, "A_1", now.Add(-time.Hour))
	if run, err := Scheduled(saves, out, 5, now.Add(48*time.Hour)); err != nil || run.Saved != 1 {
		t.Fatalf("future-clock run = %+v %v", run, err)
	}
	if last, err := LastScheduled(out, now); err != nil || !last.IsZero() {
		t.Fatalf("LastScheduled = %v %v, want zero", last, err)
	}
	if run, err := Scheduled(saves, out, 5, now); err != nil || run.Saved != 1 {
		t.Fatalf("run after the clock was fixed = %+v %v", run, err)
	}
}
