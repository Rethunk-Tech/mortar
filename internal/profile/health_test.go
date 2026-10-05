package profile

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

func TestAppendHealthDedupesWithinWindow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	at := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	if err := AppendHealth(dir, HealthPoint{At: at, Problems: 2, Warnings: 1, Updates: 3}); err != nil {
		t.Fatal(err)
	}
	later := at.Add(5 * time.Minute)
	if err := AppendHealth(dir, HealthPoint{At: later, Problems: 2, Warnings: 1, Updates: 3}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHealth(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("points = %d, want 1", len(got))
	}
}

func TestAppendHealthAllowsAfterWindow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	at := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	if err := AppendHealth(dir, HealthPoint{At: at, Problems: 1, Warnings: 0, Updates: 0}); err != nil {
		t.Fatal(err)
	}
	later := at.Add(11 * time.Minute)
	if err := AppendHealth(dir, HealthPoint{At: later, Problems: 1, Warnings: 0, Updates: 0}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHealth(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("points = %d, want 2", len(got))
	}
}

func TestAppendHealthCapsHistory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	seed := make([]HealthPoint, maxHealthPoints)
	for i := range seed {
		seed[i] = HealthPoint{At: time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC), Problems: i}
	}
	if err := datadir.WriteJSON(filepath.Join(dir, healthFile), healthFileData{Points: seed}); err != nil {
		t.Fatal(err)
	}
	for i := maxHealthPoints; i < maxHealthPoints+5; i++ {
		if err := AppendHealth(dir, HealthPoint{
			At:       time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC),
			Problems: i,
		}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ReadHealth(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != maxHealthPoints {
		t.Fatalf("points = %d, want %d", len(got), maxHealthPoints)
	}
	if got[0].Problems != 5 {
		t.Fatalf("oldest problems = %d, want 5", got[0].Problems)
	}
	if got[len(got)-1].Problems != maxHealthPoints+4 {
		t.Fatalf("newest problems = %d, want %d", got[len(got)-1].Problems, maxHealthPoints+4)
	}
}

func TestAppendHealthAppendsWhenCountsChange(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	at := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	if err := AppendHealth(dir, HealthPoint{At: at, Problems: 0, Warnings: 0, Updates: 0}); err != nil {
		t.Fatal(err)
	}
	if err := AppendHealth(dir, HealthPoint{At: at.Add(time.Minute), Problems: 1, Warnings: 0, Updates: 0}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHealth(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("points = %d, want 2", len(got))
	}
}

func TestReadHealthMissingFile(t *testing.T) {
	t.Parallel()
	got, err := ReadHealth(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("points = %d", len(got))
	}
}

func TestReadHealthCorruptFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := datadir.WriteJSON(filepath.Join(dir, healthFile), "not-json"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadHealth(dir); err == nil {
		t.Fatal("expected error")
	}
}
