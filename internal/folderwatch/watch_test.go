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
		_ = Run(ctx, Deps{Targets: targets, Emit: g.emit, Quiet: 50 * time.Millisecond, Retarget: 50 * time.Millisecond})
		close(done)
	}()
	t.Cleanup(func() { cancel(); <-done })
	time.Sleep(200 * time.Millisecond)
	return g
}

func TestBurstFiresOnce(t *testing.T) {
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
