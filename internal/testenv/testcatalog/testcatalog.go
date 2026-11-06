// Package testcatalog serves a test its own game catalog; only tests import it.
package testcatalog

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// Use serves the bundled catalog, changed by edit, until the test ends. edit may add to or replace m.Games: the
// bundled slice is cloned first, but a game in it is edited in place only after cloning that game's own fields.
func Use(t testing.TB, edit func(m *components.Manifest)) {
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
}

// AddGames serves the bundled catalog plus games until the test ends.
func AddGames(t testing.TB, games ...components.GameInfo) {
	t.Helper()
	Use(t, func(m *components.Manifest) { m.Games = append(m.Games, games...) })
}
