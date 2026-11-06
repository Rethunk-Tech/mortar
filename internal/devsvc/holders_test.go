package devsvc

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

func TestProjectPathsListFoldersFirstSkipGitContentsAndStopAtTheCap(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.txt", "src/b.cs", ".git/HEAD", ".git/refs/x"} {
		p := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := projectPaths(dir)
	want := []string{dir, filepath.Join(dir, ".git"), filepath.Join(dir, "src"), filepath.Join(dir, "a.txt"), filepath.Join(dir, "src", "b.cs")}
	if !slices.Equal(got, want) {
		t.Errorf("projectPaths = %v, want %v", got, want)
	}

	big := t.TempDir()
	for i := range maxHolderPaths + 50 {
		if err := os.WriteFile(filepath.Join(big, strconv.Itoa(i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(projectPaths(big)); n != maxHolderPaths {
		t.Errorf("a tree over the cap listed %d paths, want %d", n, maxHolderPaths)
	}
}
