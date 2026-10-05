package problems

import (
	"context"
	"errors"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

type fakeDeprecations struct {
	m   map[string]thunderstore.Deprecation
	err error
}

func (f fakeDeprecations) Deprecated(context.Context, string, string) (map[string]thunderstore.Deprecation, error) {
	return f.m, f.err
}

func TestDeprecatedPackagesMatchesInstalledOnesAndCarriesTheReplacement(t *testing.T) {
	pkgs := []profile.PackageRef{{Key: "k1", Name: "Fay-Legacy", Version: "1.0.0"}, {Key: "k2", Name: "Ann-Fine"}}
	src := fakeDeprecations{m: map[string]thunderstore.Deprecation{"fay-legacy": {Replacement: "Alice-New"}}}
	got := deprecatedPackages(t.Context(), src, "lethal-company", "1", pkgs)
	if len(got) != 1 || got[0].Key != "k1" || got[0].Replacement != "Alice-New" {
		t.Fatalf("got %+v", got)
	}
	if deprecatedPackages(t.Context(), fakeDeprecations{err: errors.New("offline")}, "lethal-company", "1", pkgs) != nil {
		t.Fatal("an index failure must find nothing")
	}
	if deprecatedPackages(t.Context(), src, "", "1", pkgs) != nil {
		t.Fatal("a game with no Thunderstore community has nothing to check")
	}
}
