package profile

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// A package moved to win a file must lay its copy out even when the loser's copy has the same size: a recompiled DLL
// often keeps its size, and the loser's copy, laid out last, is newer than the winner's store file.
func TestAPackageMadeToWinAFileReplacesALoserOfTheSameSize(t *testing.T) {
	t.Parallel()
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
	if _, err := e.MakePackageWin(folderGame, p.ID, res[0].Profile.Entries[0].Key, res[1].Profile.Entries[1].Key); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	dir, err := e.ProfileDir(folderGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, _ error) error {
		if d != nil && d.Name() == "same.package" && d.Type().IsRegular() {
			b, _ := os.ReadFile(path)
			got = append(got, string(b))
		}
		return nil
	})
	if len(got) != 1 || got[0] != "AAAA" {
		t.Fatalf("laid out = %q, want the winner's AAAA", got)
	}
}
