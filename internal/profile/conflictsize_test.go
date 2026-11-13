package profile

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

// A package moved to win a file must lay its copy out even when the loser's copy has the same size: a recompiled DLL
// often keeps its size, and the loser's copy, laid out last, is newer than the winner's store file.
func TestAPackageMadeToWinAFileReplacesALoserOfTheSameSize(t *testing.T) {
	t.Parallel()
	e, p, dir := madeToWin(t, nil)
	if got := laidOut(dir, "same.package"); len(got) != 1 || got[0] != "AAAA" {
		t.Fatalf("laid out = %q, want the winner's AAAA", got)
	}
	if conflicts, err := e.FileConflicts(folderGame, p.ID); err != nil || len(conflicts) != 1 {
		t.Fatalf("conflicts = %+v, %v", conflicts, err)
	}
}

// A file the player edited in the old winner's copy does not hold the new winner off it: the new winner's file goes
// down, as the conflicts list says, and the edit is kept beside it under the rescue name.
func TestAPackageMadeToWinAnEditedFileRescuesTheEdit(t *testing.T) {
	t.Parallel()
	_, _, dir := madeToWin(t, func(dir string) {
		rel := laidOutPath(t, dir, "same.package")
		if err := safefs.WriteFile(rel, []byte("player's edit"), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	if got := laidOut(dir, "same.package"); len(got) != 1 || got[0] != "AAAA" {
		t.Fatalf("laid out = %q, want the winner's AAAA", got)
	}
	if got := laidOut(dir, "same.package.mortar-rescued"); len(got) != 1 || got[0] != "player's edit" {
		t.Fatalf("rescued = %q, want the player's edit", got)
	}
}

// madeToWin installs two packages that ship same.package (AAAA, then BBBB, which wins), runs edit on the profile
// folder, makes the first win, and syncs. It returns the profile folder.
func madeToWin(t *testing.T, edit func(dir string)) (env, Profile, string) {
	t.Helper()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	var res []InstallResult
	for _, c := range []struct {
		body string
		src  Source
	}{{"AAAA", cfSource(1)}, {"BBBB", Source{Kind: KindCurseForge, Name: "Other", ModID: 8, FileID: 1}}} {
		r, err := e.InstallSource(t.Context(), folderGame, p.ID, zipOf(t, "a.zip", map[string]string{"same.package": c.body}), c.src)
		if err != nil {
			t.Fatal(err)
		}
		res = append(res, r)
		if err := e.SyncPackages(folderGame, p.ID); err != nil {
			t.Fatal(err)
		}
	}
	dir, err := e.ProfileDir(folderGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		edit(dir)
	}
	if _, err := e.MakePackageWin(folderGame, p.ID, res[0].Profile.Entries[0].Key, res[1].Profile.Entries[1].Key); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	return e, p, dir
}

// laidOut is the content of every regular file named name below dir.
func laidOut(dir, name string) []string {
	var got []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, _ error) error {
		if d != nil && d.Name() == name && d.Type().IsRegular() {
			b, _ := safefs.ReadFile(path)
			got = append(got, string(b))
		}
		return nil
	})
	return got
}

func laidOutPath(t *testing.T, dir, name string) string {
	t.Helper()
	var found []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, _ error) error {
		if d != nil && d.Name() == name && d.Type().IsRegular() {
			found = append(found, path)
		}
		return nil
	})
	if len(found) != 1 {
		t.Fatalf("%s laid out at %q", name, found)
	}
	return found[0]
}
