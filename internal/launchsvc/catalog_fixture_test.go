//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
)

// useCatalog serves the bundled catalog, changed by edit, for the length of the test, and points Mortar's data at a
// fresh folder.
func useCatalog(t *testing.T, edit func(m *components.Manifest)) {
	t.Helper()
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	m.Games = slices.Clone(m.Games)
	edit(&m)
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	c := components.NewClient(nil)
	c.SetManifest(m)
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })

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
