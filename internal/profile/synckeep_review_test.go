package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// A profile copy with no placed record (one a restored backup brought, or the player dropped in) is kept by the first
// sync; the second must keep it too, not treat the pending record the first wrote as licence to overwrite it.
func TestASecondSyncKeepsAnUnrecordedProfileCopy(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"mc.package": "store bytes"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10)); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	dir, _ := e.ProfileDir(folderGame, p.ID)
	copyPath := filepath.Join(dir, "Mods", "mc.package")
	if err := os.WriteFile(copyPath, []byte("player=2 and more"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, placedFile)); err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		if err := e.SyncPackages(folderGame, p.ID); err != nil {
			t.Fatal(err)
		}
		if got, _ := fsx.ReadFile(copyPath); string(got) != "player=2 and more" {
			t.Fatalf("sync %d overwrote the profile's copy: %q", i+1, got)
		}
	}
}
