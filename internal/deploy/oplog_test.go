package deploy

import (
	"path/filepath"
	"testing"
)

// A test cannot observe an fsync, so the log skips it and both add and close succeed on a plain append.
func TestOpLogSkipsTheFlushUnderTest(t *testing.T) {
	t.Parallel()
	if syncWrites {
		t.Fatal("syncWrites is on under go test")
	}
	log, err := openLog(filepath.Join(t.TempDir(), "journal"))
	if err != nil {
		t.Fatal(err)
	}
	if err := log.add(step{M: "dir"}, true); err != nil {
		t.Fatal(err)
	}
	if err := log.close(); err != nil {
		t.Fatal(err)
	}
}
