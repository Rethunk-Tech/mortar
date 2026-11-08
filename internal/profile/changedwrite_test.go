package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestValidChangedPath(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		game, rel string
		want      bool
	}{
		{game: "sims4", rel: "Mods/mc/x.cfg", want: true},
		{game: "sims4", rel: "Mods/mc/mc_state.dat", want: true},
		{game: "sims4", rel: "Mods/mc/evil.ts4script"},
		{game: "sims4", rel: "Mods/pack/evil.dll"},
		{game: "sims4", rel: "changed/Mods/old/x.cfg", want: true},
		{game: "vintage-story", rel: "Mods/pack/modinfo.json", want: true},
		// Stardew deploys by redirect, Lethal Company has no mods target: neither accepts a copy.
		{game: "stardew", rel: "Mods/mc/x.cfg"},
		{game: "lethal-company", rel: "BepInEx/plugins/x.dll"},
		{game: "no-such-game", rel: "Mods/x.cfg"},
		{game: "sims4", rel: "profile.json"},
		{game: "sims4", rel: ""},
		{game: "sims4", rel: `Mods\x.cfg`},
		{game: "sims4", rel: "../Mods/x.cfg"},
		{game: "sims4", rel: "Mods/../x.cfg"},
		{game: "sims4", rel: "Mods//x.cfg"},
		{game: "sims4", rel: "Mods"},
		{game: "sims4", rel: "changed"},
	} {
		if got := ValidChangedPath(tc.game, tc.rel); got != tc.want {
			t.Errorf("ValidChangedPath(%q, %q) = %v, want %v", tc.game, tc.rel, got, tc.want)
		}
	}
}

// changedPackage is a Sims 4 profile whose one package lays out Mods/mc/x.cfg, a place a changed copy may go.
func changedPackage(t *testing.T) (env, Profile, string) {
	t.Helper()
	e := newEnv(t)
	p, err := e.Create(folderGame, "S")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"mc/x.cfg": "store=1", "mc/other.cfg": "other=1"}
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, zipOf(t, "a.zip", files), cfSource(10)); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	dir, err := e.ProfileDir(folderGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return e, p, dir
}

func TestWriteChangedFilesRefusesAPathTheProfileDoesNotKeepCopiesAt(t *testing.T) {
	t.Parallel()
	e, p, dir := changedPackage(t)
	bad := []string{"", "..", "../x.cfg", "Mods/../x.cfg", "/abs.cfg", `Mods\mc\x.cfg`, "other/x.cfg", "changed"}
	for _, rel := range bad {
		files := map[string][]byte{"Mods/mc/x.cfg": []byte("player=3"), rel: []byte("bad")}
		if err := e.WriteChangedFiles(folderGame, p.ID, files, true); err == nil {
			t.Errorf("%q was accepted", rel)
		}
	}
	if got := testfs.Read(t, filepath.Join(dir, "Mods", "mc", "x.cfg")); got != "store=1" {
		t.Fatalf("Mods/mc/x.cfg = %q, a refused set was partly written", got)
	}
}

func TestWriteChangedFilesRefusesAPathThatReachesALink(t *testing.T) {
	t.Parallel()
	e, p, dir := changedPackage(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "Mods", "linked")); err != nil {
		t.Skipf("links are not available: %v", err)
	}
	files := map[string][]byte{"Mods/mc/x.cfg": []byte("player=3"), "Mods/linked/x.cfg": []byte("bad")}
	if err := e.WriteChangedFiles(folderGame, p.ID, files, true); err == nil {
		t.Fatal("a write through a link was accepted")
	}
	if _, err := os.Stat(filepath.Join(outside, "x.cfg")); err == nil {
		t.Fatal("a file landed outside the profile")
	}
	if got := testfs.Read(t, filepath.Join(dir, "Mods", "mc", "x.cfg")); got != "store=1" {
		t.Fatalf("Mods/mc/x.cfg = %q, a refused set was partly written", got)
	}
}

func TestWriteChangedFilesWithoutOverwriteKeepsTheProfilesFile(t *testing.T) {
	t.Parallel()
	e, p, dir := changedPackage(t)
	one := map[string][]byte{"Mods/mc/x.cfg": []byte("from the other computer")}
	if err := e.WriteChangedFiles(folderGame, p.ID, one, false); err != nil {
		t.Fatal(err)
	}
	if got := testfs.Read(t, filepath.Join(dir, "Mods", "mc", "x.cfg")); got != "store=1" {
		t.Fatalf("Mods/mc/x.cfg = %q, want the file the profile already had", got)
	}
	if err := e.WriteChangedFiles(folderGame, p.ID, one, true); err != nil {
		t.Fatal(err)
	}
	if got := testfs.Read(t, filepath.Join(dir, "Mods", "mc", "x.cfg")); got != "from the other computer" {
		t.Fatalf("Mods/mc/x.cfg = %q after overwrite", got)
	}
}

// A changed copy of a package's file is recorded so SyncPackages keeps it, and one of a file no package lays out is not,
// since there is no package copy for it to be kept over.
func TestWriteChangedFilesMarksOnlyTheCopiesAPackageLaysOutAsChanged(t *testing.T) {
	t.Parallel()
	e, p, dir := changedPackage(t)
	files := map[string][]byte{
		"Mods/mc/x.cfg":         []byte("player=3"),
		"changed/Mods/held.cfg": []byte("held"),
	}
	if err := e.WriteChangedFiles(folderGame, p.ID, files, true); err != nil {
		t.Fatal(err)
	}
	rec := readPlaced(dir)
	if got := rec["Mods/mc/x.cfg"]; got != (placedRec{Hash: "changed copy", Size: -1}) {
		t.Fatalf("the record of Mods/mc/x.cfg = %+v", got)
	}
	if _, ok := rec["changed/Mods/held.cfg"]; ok {
		t.Fatalf("a file no package lays out was marked changed: %+v", rec)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	if got := testfs.Read(t, filepath.Join(dir, "Mods", "mc", "x.cfg")); got != "player=3" {
		t.Fatalf("Mods/mc/x.cfg = %q, the sync replaced the changed copy with the package's", got)
	}
	if got := testfs.Read(t, filepath.Join(dir, "changed", "Mods", "held.cfg")); got != "held" {
		t.Fatalf("changed/Mods/held.cfg = %q", got)
	}
}
