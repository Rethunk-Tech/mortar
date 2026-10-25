package profile

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// A removed package's files leave the profile in the background right after, not at the next launch's sync.
func TestRemovingAPackageTakesItsFilesNow(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "gone.dll-1-1-0-0-1700625972.zip"), map[string]string{"gone.dll": "MZ"})
	res, err := e.InstallArchive(t.Context(), "lethal-company", p.ID, zip)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages("lethal-company", p.ID); err != nil {
		t.Fatal(err)
	}
	key := res.Profile.Entries[0].Key
	dir, _ := e.ProfileDir("lethal-company", p.ID)
	dll := filepath.Join(dir, "BepInEx", "plugins", key, "gone.dll")
	if _, err := fsx.Stat(dll); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RemoveEntries("lethal-company", p.ID, []string{key}); err != nil {
		t.Fatal(err)
	}
	e.WaitSyncs()
	if _, err := fsx.Stat(dll); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the removed package's file is still there: %v", err)
	}
}
