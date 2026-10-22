package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestSuggestFoldersFindsACopyTwoLevelsDown(t *testing.T) {
	home := t.TempDir()
	info, ok := catalogGame("stardew")
	if !ok {
		t.Fatal("no stardew in the catalog")
	}
	want := filepath.Join(home, "Games", "Heroic", "Stardew Valley")
	marker := filepath.Join(want, filepath.FromSlash(info.MarkerPath()))
	testfs.WriteAt(t, marker, nil)
	if err := os.MkdirAll(filepath.Join(home, "Desktop", "notes"), 0o700); err != nil {
		t.Fatal(err)
	}
	got := (&Service{home: home}).SuggestFolders("stardew")
	if !slices.Equal(got, []string{want}) {
		t.Fatalf("suggested %v, want %v", got, []string{want})
	}
}
