package datasvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestAHardLinkCostsNothingWhenItsFileWasCountedWithoutExtents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "store.bin")
	testfs.WriteAt(t, file, make([]byte, 8192))
	link := filepath.Join(dir, "link.bin")
	if err := os.Link(file, link); err != nil {
		t.Skipf("no hard links: %v", err)
	}
	first, err := os.Lstat(file)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	s := newShareAcc()
	// The file is read while its blocks await allocation, and its link once they have them.
	if _, ok := fileExclusive(file, first, fileProbe{}, s); !ok {
		t.Skip("the filesystem reports no inode")
	}
	if n, _ := fileExclusive(link, second, fileProbe{ok: true, runs: []extentRun{{physical: 4096, length: 8192}}}, s); n != 0 {
		t.Errorf("the link was charged %d bytes", n)
	}
}
