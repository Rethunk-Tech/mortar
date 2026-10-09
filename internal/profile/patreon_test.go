package profile_test

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestAFileSavedFromAPatreonPostIsAnEntryThatNamesThePost(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ps := profile.OpenIn(dir, store.OpenAt(filepath.Join(dir, "store")))
	p, err := ps.Create("stardew", "A")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "PatronMod.zip"), map[string]string{
		"PatronMod/manifest.json": `{"Name":"Patron Mod","Author":"a","Version":"1.0.0","UniqueID":"a.PatronMod","EntryDll":"PatronMod.dll"}`,
		"PatronMod/PatronMod.dll": "x",
	})
	res, err := ps.InstallPatreon(t.Context(), "stardew", p.ID, zip, "12345678")
	if err != nil {
		t.Fatal(err)
	}
	last := res.Profile.Entries[len(res.Profile.Entries)-1]
	if last.Source.Kind != profile.KindPatreon || last.Source.Name != "12345678" {
		t.Fatalf("source = %+v", last.Source)
	}
}
