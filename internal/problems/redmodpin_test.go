package problems

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader/folder"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// The REDmod DLC is looked for in the copy of the game the profile runs, not the one Mortar found first.
func TestTheREDmodCheckLooksInTheProfilesOwnInstall(t *testing.T) {
	const id = "cyberpunk-2077"
	datadirtest.Use(t, t.TempDir())
	found, pinned := t.TempDir(), t.TempDir()
	for _, dir := range []string{found, pinned} {
		testfs.WriteAt(t, filepath.Join(dir, "bin", "x64", "Cyberpunk2077.exe"), []byte("x"))
	}
	testfs.WriteAt(t, filepath.Join(pinned, "tools", "redmod", "bin", "redMod.exe"), []byte("x"))
	testfs.WriteAt(t, filepath.Join(pinned, "REDprelauncher.exe"), []byte("x"))
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders[id] = found }); err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, id, "A")
	if _, err := profiles.SetInstall(id, p.ID, game.FolderPin(pinned)); err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "r.zip"), map[string]string{"mods/x/info.json": `{"name":"X","version":"1"}`, "mods/x/archives/a.archive": "a"})
	if _, err := profiles.InstallArchive(t.Context(), id, p.ID, zip); err != nil {
		t.Fatal(err)
	}
	s := NewService(t.TempDir(), set, profiles, nil)
	got := s.extensionFailures(id, p.ID, folder.Loader{})
	if slices.ContainsFunc(got, func(f LoadFailure) bool { return f.Plugin == "REDmod" }) {
		t.Fatalf("failures = %+v, want no missing-REDmod finding for an install that has it", got)
	}
}
