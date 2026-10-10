package profile_test

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestAFileSavedFromAnItchPageIsAnEntryThatNamesThePage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ps := profile.OpenIn(dir, store.OpenAt(filepath.Join(dir, "store")))
	p, err := ps.Create("stardew", "A")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "ItchMod.zip"), map[string]string{
		"ItchMod/manifest.json": `{"Name":"Itch Mod","Author":"a","Version":"1.0.0","UniqueID":"a.ItchMod","EntryDll":"ItchMod.dll"}`,
		"ItchMod/ItchMod.dll": "x",
	})
	res, err := ps.InstallItch(t.Context(), "stardew", p.ID, zip, "someone/cool-mod")
	if err != nil {
		t.Fatal(err)
	}
	last := res.Profile.Entries[len(res.Profile.Entries)-1]
	if last.Source.Kind != profile.KindItch || last.Source.Name != "someone/cool-mod" {
		t.Fatalf("source = %+v", last.Source)
	}
}
