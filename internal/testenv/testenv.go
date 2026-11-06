// Package testenv opens the stores a service test needs; only tests import it.
package testenv

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// Stores opens the item store and the profile store over it in the data folder the test's environment names.
func Stores(t testing.TB) (*store.Store, *profile.Store) {
	t.Helper()
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	return items, profiles
}

// Env gives the test a data folder of its own, then opens the settings and the profile store in it.
func Env(t testing.TB) (*settings.Store, *profile.Store) {
	t.Helper()
	datadirtest.Use(t, t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := Stores(t)
	return set, profiles
}

// StoresIn opens the item store and the profile store over it in dir, for a test that keeps its data folder to itself.
func StoresIn(t testing.TB, dir string) (*store.Store, *profile.Store) {
	t.Helper()
	items := store.OpenAt(filepath.Join(dir, "store"))
	return items, profile.OpenIn(filepath.Join(dir, "profiles"), items)
}

// Profile creates a profile for gameID or fails the test.
func Profile(t testing.TB, profiles *profile.Store, gameID, name string) profile.Profile {
	t.Helper()
	p, err := profiles.Create(gameID, name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// LethalFriends opens private stores and creates the Lethal Company profile "Friends" in them.
func LethalFriends(t testing.TB) (*profile.Store, profile.Profile) {
	t.Helper()
	_, ps := StoresIn(t, t.TempDir())
	return ps, Profile(t, ps, "lethal-company", "Friends")
}
