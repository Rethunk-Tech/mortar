// Package testenv opens the stores a service test needs; only tests import it.
package testenv

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
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

// Profile creates a Stardew profile or fails the test.
func Profile(t testing.TB, profiles *profile.Store, name string) profile.Profile {
	t.Helper()
	p, err := profiles.Create("stardew", name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
