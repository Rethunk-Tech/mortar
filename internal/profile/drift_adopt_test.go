package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// Adopting an untracked folder from mods/ or from the hold dir turns it into an entry whose files sit in the store,
// and removes the source; a bad name removes nothing.
func TestAdoptDriftFolder(t *testing.T) {
	t.Parallel()
	for _, where := range []string{"mods", "hold"} {
		t.Run(where, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			p := mustCreate(t, e, "Farm")
			st, err := settings.OpenIn(e.base)
			if err != nil {
				t.Fatal(err)
			}
			svc := NewService(e.Store, t.TempDir(), st)
			dir, err := e.profileDir("stardew", p.ID)
			if err != nil {
				t.Fatal(err)
			}
			parent := filepath.Join(dir, "mods")
			if where == "hold" {
				parent = holdDir(dir)
			}
			writeFile(t, parent, "Loose/manifest.json", manifestJSON("Me.Loose"))
			src := filepath.Join(parent, "Loose")

			for _, bad := range []string{"", "..", "Nope"} {
				if _, err := svc.AdoptDriftFolder(t.Context(), "stardew", p.ID, bad); err == nil {
					t.Fatalf("adopt %q succeeded", bad)
				}
				if !exists(filepath.Join(src, "manifest.json")) {
					t.Fatalf("adopt %q removed the folder", bad)
				}
			}
			if _, err := svc.AdoptDriftFolder(t.Context(), "stardew", p.ID, "Nope"); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing name err = %v", err)
			}

			got, err := svc.AdoptDriftFolder(t.Context(), "stardew", p.ID, "Loose")
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Entries) != 1 {
				t.Fatalf("entries = %+v", got.Entries)
			}
			if exists(src) {
				t.Fatal("source folder still there")
			}
			stored, err := e.items.Path("stardew", got.Entries[0].Key)
			if err != nil {
				t.Fatal(err)
			}
			if !exists(filepath.Join(stored, "manifest.json")) {
				t.Fatalf("store holds %v", names(t, stored))
			}
		})
	}
}
