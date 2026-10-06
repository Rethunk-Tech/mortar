package datasvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/support"
)

// A restart exits without the app's shutdown, so the moved log it leaves must still read as a clean quit.
func TestRestartLeavesACleanShutdownRecord(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	startOver := func() bool {
		t.Helper()
		if err := os.Rename(filepath.Join(dir, "mortar.log"), filepath.Join(dir, "mortar.prev.log")); err != nil {
			t.Fatal(err)
		}
		return support.DetectLastRunCrashed(dir)
	}
	write := func() {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "mortar.log"), []byte("level=INFO msg=\"copied before the move\"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	if !startOver() {
		t.Fatal("a log without the record must read as a crash, or this test proves nothing")
	}
	write()
	logCleanShutdown()
	if startOver() {
		t.Fatal("a restart read as a crash")
	}
}
