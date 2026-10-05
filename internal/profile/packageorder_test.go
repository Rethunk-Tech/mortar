package profile_test

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestMovePackageChangesWhoWins(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ps, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	const lc = "lethal-company"
	p, err := ps.Create(lc, "Friends")
	if err != nil {
		t.Fatal(err)
	}
	var loaded profile.InstallResult
	for _, name := range []string{"A", "B"} {
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), name+".zip"), map[string]string{
			"manifest.json":     `{"name":"` + name + `","version_number":"1.0.0"}`,
			"config/Shared.cfg": name,
		})
		if loaded, err = ps.InstallSource(lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-" + name, Version: "1.0.0"}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ps.PackageOverrides(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	a, b := loaded.Profile.Entries[0].Key, loaded.Profile.Entries[1].Key
	if got[b] != 1 || got[a] != 0 {
		t.Fatalf("later package wins: %v", got)
	}
	if _, err := ps.MovePackage(lc, p.ID, b, -1); err != nil {
		t.Fatal(err)
	}
	got, err = ps.PackageOverrides(lc, p.ID)
	if err != nil || got[a] != 1 || got[b] != 0 {
		t.Fatalf("after move: %v, %v", got, err)
	}
}
