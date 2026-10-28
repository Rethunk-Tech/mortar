package testfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTouchEditsTheFileAndPutsItBack(t *testing.T) {
	path := WriteFile(t, t.TempDir(), "a.json", "{}")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("touch", func(t *testing.T) {
		Touch(t, path)
		if got := Read(t, path); got != "{}\n" {
			t.Fatalf("touched = %q", got)
		}
	})
	after, err := os.Stat(path)
	if err != nil || Read(t, path) != "{}" || !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("not restored: %q %v, %v", Read(t, path), after.ModTime(), err)
	}
}

func TestWriteFarmNamesTheFarm(t *testing.T) {
	saves := t.TempDir()
	WriteFarm(t, saves, "Farm_1", "Sunny", "x")
	if Read(t, filepath.Join(saves, "Farm_1", "Farm_1")) != "x" {
		t.Fatal("save file")
	}
	if got := Read(t, filepath.Join(saves, "Farm_1", "SaveGameInfo")); got != "<Farmer><name>Ann</name><farmName>Sunny</farmName></Farmer>" {
		t.Fatalf("SaveGameInfo = %q", got)
	}
}
