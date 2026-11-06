package folderwatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A held project folder has no watch open below it, so Windows lets it be moved; released, it is watched again.
func TestHoldClosesTheWatchesBelowAFolderUntilReleased(t *testing.T) {
	t.Parallel()
	proj := filepath.Join(t.TempDir(), "Cadavers Redone²")
	src := filepath.Join(proj, "src")
	if err := os.MkdirAll(src, 0o750); err != nil {
		t.Fatal(err)
	}
	holds := NewHolds()
	g := &got{}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		_ = Run(ctx, Deps{
			Targets: func() []Target { return []Target{{Event: ExtraFolderEvent, Game: "lethal-company", Dir: src}} },
			Emit:    g.emit, Quiet: 50 * time.Millisecond, Stable: 100 * time.Millisecond, Retarget: 50 * time.Millisecond, Holds: holds,
		})
		close(done)
	}()
	t.Cleanup(func() { cancel(); <-done })
	time.Sleep(200 * time.Millisecond)

	release := holds.Hold(t.Context(), proj)
	moved := proj + " moved"
	if err := os.Rename(proj, moved); err != nil {
		t.Fatalf("moving the held folder: %v", err)
	}
	if err := os.Rename(moved, proj); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(src, "Plugin.cs"), []byte("x"), 0o600)
	time.Sleep(400 * time.Millisecond)
	g.mu.Lock()
	early := len(g.evs)
	g.mu.Unlock()
	if early != 0 {
		t.Fatalf("a held folder was still watched: %v", g.evs)
	}

	release()
	time.Sleep(200 * time.Millisecond)
	_ = os.WriteFile(filepath.Join(src, "Patch.cs"), []byte("x"), 0o600)
	g.wait(t, 1)
}

// Without a running watch, Hold returns at once.
func TestHoldWithoutAWatchReturns(t *testing.T) {
	t.Parallel()
	NewHolds().Hold(t.Context(), t.TempDir())()
}
