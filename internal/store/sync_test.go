package store

import (
	"path/filepath"
	"testing"
)

// A test cannot observe an fsync, so syncPath writes nothing and accepts a path no flush could open.
func TestSyncPathSkipsTheFlushUnderTest(t *testing.T) {
	t.Parallel()
	if err := syncPath(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("syncPath = %v, want nil", err)
	}
}
