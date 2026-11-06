package devsvc

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestAQuietTickDoesNotWalkAgain(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	src := filepath.Join(dir, "src", "ShopDrop")
	cache := &WalkCache{}
	games := []string{"lethal-company"}
	watched := func() []string { return WatchedSources(svc.Settings.Get(), games, cache)["lethal-company"] }

	if err := os.MkdirAll(filepath.Join(src, "Patches"), 0o750); err != nil {
		t.Fatal(err)
	}
	first := watched()
	// A folder made below Patches changes only Patches' own time, which the cache does not look at: only the watcher's
	// report of it brings the new folder in, and a tick without one walks nothing.
	deeper := filepath.Join(src, "Patches", "Deeper")
	if err := os.MkdirAll(deeper, 0o750); err != nil {
		t.Fatal(err)
	}
	if got := watched(); !slices.Equal(got, first) || slices.Contains(got, deeper) {
		t.Fatalf("a quiet tick walked again: %v", got)
	}
	cache.Touch(filepath.Join(dir, "elsewhere", "file"))
	if got := watched(); slices.Contains(got, deeper) {
		t.Fatalf("a change outside the project walked it again: %v", got)
	}
	cache.Touch(filepath.Join(src, "Patches", "new.cs"))
	if got := watched(); !slices.Contains(got, deeper) {
		t.Fatalf("a reported change under the project did not refresh its walk: %v", got)
	}
	// A folder made directly in the walked root moves its time, so it is found without a report.
	direct := filepath.Join(src, "Direct")
	if err := os.MkdirAll(direct, 0o750); err != nil {
		t.Fatal(err)
	}
	if got := watched(); !slices.Contains(got, direct) {
		t.Fatalf("a folder made in the root was missed: %v", got)
	}
}
