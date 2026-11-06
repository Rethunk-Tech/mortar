package folderwatch

import (
	"context"
	"path/filepath"
	"sync"
)

// Holds takes folders out of the watch for a while. Windows refuses to move a folder while a watch has a handle open
// on it or on a folder below it, so a caller about to move one to the trash holds it first.
type Holds struct {
	mu    sync.Mutex
	roots map[string]int
	// sync is Run's loop while it runs: a send makes it drop the held folders' watches and close the channel sent.
	sync    chan chan struct{}
	stopped chan struct{}
}

// NewHolds makes the holds a Run takes in its Deps.
func NewHolds() *Holds { return &Holds{roots: map[string]int{}, sync: make(chan chan struct{})} }

// Hold stops watching root and every folder below it, and returns once their watches are closed (or ctx ends); they
// stay unwatched until release is called.
func (h *Holds) Hold(ctx context.Context, root string) (release func()) {
	root = filepath.Clean(root)
	h.mu.Lock()
	h.roots[root]++
	stopped := h.stopped
	h.mu.Unlock()
	release = sync.OnceFunc(func() {
		h.mu.Lock()
		if h.roots[root]--; h.roots[root] <= 0 {
			delete(h.roots, root)
		}
		h.mu.Unlock()
	})
	if stopped == nil {
		// No watch is running, so nothing is open; one that starts later skips the held folders.
		return release
	}
	done := make(chan struct{})
	select {
	case h.sync <- done:
		select {
		case <-done:
		case <-ctx.Done():
		}
	case <-stopped:
	case <-ctx.Done():
	}
	return release
}

// held reports whether dir is a held folder or inside one.
func (h *Holds) held(dir string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for root := range h.roots {
		if rel, err := filepath.Rel(root, dir); err == nil && filepath.IsLocal(rel) {
			return true
		}
	}
	return false
}

// attach marks Run's loop as running; the returned func marks it stopped.
func (h *Holds) attach() (requests <-chan chan struct{}, detach func()) {
	if h == nil {
		return nil, func() {}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	stopped := make(chan struct{})
	h.stopped = stopped
	return h.sync, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.stopped = nil
		close(stopped)
	}
}
