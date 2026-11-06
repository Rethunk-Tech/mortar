//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testcatalog"
)

// useCatalog serves the bundled catalog, changed by edit, for the length of the test, and points Mortar's data at a
// fresh folder.
func useCatalog(t *testing.T, edit func(m *components.Manifest)) {
	t.Helper()
	testcatalog.Use(t, edit)
	data := t.TempDir()
	datadirtest.Use(t, data)
	t.Setenv("LOCALAPPDATA", data)
}

// markedGameFolder is an install folder holding only the marker file the test games name.
func markedGameFolder(t *testing.T) string {
	t.Helper()
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "G.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return folder
}
