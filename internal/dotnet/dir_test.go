//go:build unix

package dotnet

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestDLLsInStaysInsideTheFolder(t *testing.T) {
	outside := t.TempDir()
	testfs.WriteFile(t, outside, "Elsewhere.dll", "x")
	testfs.WriteFile(t, outside, "sub/Deep.dll", "x")
	dir := t.TempDir()
	testfs.WriteFile(t, dir, "Mod.dll", "x")
	testfs.WriteFile(t, dir, "sub/Other.DLL", "x")
	testfs.WriteFile(t, dir, "readme.txt", "x")
	if err := os.Symlink(filepath.Join(outside, "Elsewhere.dll"), filepath.Join(dir, "Link.dll")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "sub"), filepath.Join(dir, "linked")); err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(dir, "Mod.dll"), filepath.Join(dir, "sub", "Other.DLL")}
	if got := DLLs(dir + "/./"); !slices.Equal(got, want) {
		t.Fatalf("DLLs = %v, want %v", got, want)
	}
	if got := DLLs("relative/plugins"); got != nil {
		t.Fatalf("a relative folder gave %v", got)
	}
}
