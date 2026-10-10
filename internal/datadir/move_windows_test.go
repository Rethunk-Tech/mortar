package datadir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Windows names one folder whatever the case of its path, so a default folder spelled in another case is still the
// default: the pointer file stays and the folder is not removed.
func TestRemoveOldKeepsTheDefaultFolderSpelledInAnotherCase(t *testing.T) {
	def := filepath.Join(t.TempDir(), "Mortar")
	if err := os.MkdirAll(filepath.Join(def, "profiles"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(def, PointerName), []byte("elsewhere"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeOld(strings.ToUpper(def), def); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(def, PointerName)); err != nil {
		t.Fatalf("the pointer file went with the folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(def, "profiles")); !os.IsNotExist(err) {
		t.Fatalf("the old data stayed: %v", err)
	}
}
