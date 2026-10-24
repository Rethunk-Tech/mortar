package profile

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// A point repeating the last one's counts within the window is dropped; a later one, or one with other counts, is kept.
func TestAppendHealthKeepsAPointUnlessItRepeatsTheLast(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	first := HealthPoint{At: at, Problems: 2, Warnings: 1, Updates: 3}
	for _, tc := range []struct {
		name   string
		points []HealthPoint
		want   int
	}{
		{"no file", nil, 0},
		{"same counts within the window", []HealthPoint{first, {At: at.Add(5 * time.Minute), Problems: 2, Warnings: 1, Updates: 3}}, 1},
		{"same counts after the window", []HealthPoint{first, {At: at.Add(11 * time.Minute), Problems: 2, Warnings: 1, Updates: 3}}, 2},
		{"other counts within the window", []HealthPoint{first, {At: at.Add(time.Minute), Problems: 3, Warnings: 1, Updates: 3}}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, pt := range tc.points {
				if err := AppendHealth(dir, pt); err != nil {
					t.Fatal(err)
				}
			}
			got, err := ReadHealth(dir)
			if err != nil || len(got) != tc.want {
				t.Fatalf("points = %d, %v; want %d", len(got), err, tc.want)
			}
		})
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
