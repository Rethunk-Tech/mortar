package folderwatch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type got struct {
	mu  sync.Mutex
	evs []string
}

func (g *got) emit(name string, data any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.evs = append(g.evs, name+":"+fmt.Sprint(data))
}

func (g *got) wait(t *testing.T, n int) []string {
	t.Helper()
	for range 100 {
		g.mu.Lock()
		out := append([]string(nil), g.evs...)
		g.mu.Unlock()
		if len(out) >= n {
			return out
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("want %d events, got %v", n, g.evs)
	return nil
}

func start(t *testing.T, targets func() []Target) *got {
	g := &got{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = Run(ctx, Deps{Targets: targets, Emit: g.emit, Quiet: 50 * time.Millisecond, Stable: 300 * time.Millisecond, Retarget: 50 * time.Millisecond})
		close(done)
	}()
	t.Cleanup(func() { cancel(); <-done })
	time.Sleep(200 * time.Millisecond)
	return g
}

func TestBurstFiresOnce(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	g := start(t, func() []Target { return []Target{{DownloadsEvent, "stardew", dir}} })
	for i := range 5 {
		_ = os.WriteFile(filepath.Join(dir, string(rune('a'+i))+".zip"), nil, 0o600)
	}
	time.Sleep(400 * time.Millisecond)
	if evs := g.wait(t, 1); len(evs) != 1 || evs[0] != DownloadsEvent+":stardew" {
		t.Fatalf("events = %v", evs)
	}
}

func TestMissingFolderFiresWhenItAppears(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "Mods")
	g := start(t, func() []Target { return []Target{{ModsFolderEvent, "stardew", dir}} })
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	g.wait(t, 1)
	_ = os.WriteFile(filepath.Join(dir, "x"), nil, 0o600)
	g.wait(t, 2)
}

func TestRetargetStopsOldFolder(t *testing.T) {
	t.Parallel()
	a, b := t.TempDir(), t.TempDir()
	var mu sync.Mutex
	cur := a
	g := start(t, func() []Target {
		mu.Lock()
		defer mu.Unlock()
		return []Target{{ExtraFolderEvent, "stardew", cur}}
	})
	mu.Lock()
	cur = b
	mu.Unlock()
	time.Sleep(200 * time.Millisecond)
	_ = os.WriteFile(filepath.Join(a, "x"), nil, 0o600)
	time.Sleep(300 * time.Millisecond)
	if len(g.evs) != 0 {
		t.Fatalf("old folder still watched: %v", g.evs)
	}
	_ = os.WriteFile(filepath.Join(b, "x"), nil, 0o600)
	g.wait(t, 1)
}

func TestSlowDirectWriterWaitsForStableSize(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	g := start(t, func() []Target { return []Target{{DownloadsEvent, "stardew", dir}} })
	f, err := os.Create(filepath.Clean(filepath.Join(dir, "big.zip")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	_, _ = f.WriteString("a")
	time.Sleep(150 * time.Millisecond) // quiet period has passed; the first size check is under way
	_, _ = f.WriteString("b")
	time.Sleep(100 * time.Millisecond)
	g.mu.Lock()
	early := len(g.evs)
	g.mu.Unlock()
	if early != 0 {
		t.Fatalf("announced while the writer was still going: %v", g.evs)
	}
	if evs := g.wait(t, 1); len(evs) != 1 {
		t.Fatalf("events = %v", evs)
	}
}

func TestRenamedInFileIsAnnouncedAtOnce(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	g := start(t, func() []Target { return []Target{{DownloadsEvent, "stardew", dir}} })
	part := filepath.Join(t.TempDir(), "x.part")
	if err := os.WriteFile(part, []byte("zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(part, filepath.Join(dir, "x.zip")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond) // well under the 300 ms stability wait
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.evs) != 1 {
		t.Fatalf("events = %v", g.evs)
	}
}

// The open game changes on a retarget tick; a download that landed just before it must still be announced, under
// the new game, whether the folder was still quiet-waiting or already waiting for a stable size.
func TestRetargetKeepsPendingEvent(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct{ quiet, flipAfter time.Duration }{
		"quiet wait":  {400 * time.Millisecond, 0},
		"stable wait": {50 * time.Millisecond, 200 * time.Millisecond},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			var mu sync.Mutex
			game := "sims4"
			g := &got{}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				_ = Run(ctx, Deps{Targets: func() []Target {
					mu.Lock()
					defer mu.Unlock()
					return []Target{{DownloadsEvent, game, dir}}
				}, Emit: g.emit, Quiet: c.quiet, Stable: 500 * time.Millisecond, Retarget: 50 * time.Millisecond})
				close(done)
			}()
			t.Cleanup(func() { cancel(); <-done })
			time.Sleep(200 * time.Millisecond)
			if err := os.WriteFile(filepath.Join(dir, "a.zip"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
			time.Sleep(c.flipAfter)
			mu.Lock()
			game = "stardew"
			mu.Unlock()
			evs := g.wait(t, 1)
			time.Sleep(700 * time.Millisecond)
			if evs = g.wait(t, 1); len(evs) != 1 || evs[0] != DownloadsEvent+":stardew" {
				t.Fatalf("events = %v", evs)
			}
		})
	}
}
