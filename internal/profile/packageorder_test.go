package profile_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestMovePackageChangesWhoWins(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ps := profile.OpenIn(dir, store.OpenAt(filepath.Join(dir, "store")))
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
		if loaded, err = ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-" + name, Version: "1.0.0"}); err != nil {
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

func TestSyncStoppedPartWayStillTakesItsFilesBack(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ps := profile.OpenIn(dir, store.OpenAt(filepath.Join(dir, "store")))
	const lc = "lethal-company"
	p, err := ps.Create(lc, "Friends")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "a.zip"), map[string]string{
		"manifest.json": `{"name":"A","version_number":"1.0.0"}`,
		"plugins/a.dll": "a",
		"plugins/z.dll": "z",
	})
	loaded, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-A", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	root, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	plugins := filepath.Join(root, "BepInEx", "plugins", "Ns-A")
	// A folder in z.dll's place stops the sync after a.dll is placed.
	if err := os.MkdirAll(filepath.Join(plugins, "z.dll"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(plugins, "z.dll", "in-the-way"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ps.SyncPackages(lc, p.ID); err == nil {
		t.Fatal("the sync placed a file over a folder")
	}
	if _, err := os.Stat(filepath.Join(plugins, "a.dll")); err != nil {
		t.Fatalf("a.dll was not placed before the stop: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(plugins, "z.dll")); err != nil {
		t.Fatal(err)
	}
	if _, err := ps.RemoveEntries(lc, p.ID, []string{loaded.Profile.Entries[0].Key}); err != nil {
		t.Fatal(err)
	}
	if err := ps.SyncPackages(lc, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(plugins, "a.dll")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the removed package's a.dll is still loaded: %v", err)
	}
}
