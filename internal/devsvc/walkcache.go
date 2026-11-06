package devsvc

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WalkCache remembers each project's folder walk for the folder watch, which re-reads its targets every couple of
// seconds. A walk is reused until the watcher reports a change under its root, or the root's own modification time
// moves (a folder made or removed directly in it, which no watch may cover), so a quiet tick walks nothing. A nil cache
// walks every time.
type WalkCache struct {
	mu      sync.Mutex
	entries map[string]walkEntry
}

type walkEntry struct {
	root  string
	mtime time.Time
	dirs  []string
}

// Touch drops the walks of every project that holds path, which the watcher saw change.
func (c *WalkCache) Touch(path string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, e := range c.entries {
		if path == e.root || strings.HasPrefix(path, e.root+string(filepath.Separator)) {
			delete(c.entries, key)
		}
	}
}

// walk returns the cached result for key while root is unchanged, else runs walk and keeps what it returns.
func (c *WalkCache) walk(key, root string, walk func() []string) []string {
	if c == nil {
		return walk()
	}
	mtime := modTime(root)
	c.mu.Lock()
	e, ok := c.entries[key]
	c.mu.Unlock()
	if ok && e.mtime.Equal(mtime) {
		return e.dirs
	}
	dirs := walk()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]walkEntry{}
	}
	c.entries[key] = walkEntry{root: root, mtime: mtime, dirs: dirs}
	return dirs
}

// keep drops the walks under prefix that key list did not ask for this round, so a removed project is forgotten.
func (c *WalkCache) keep(prefix string, used map[string]bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if strings.HasPrefix(key, prefix) && !used[key] {
			delete(c.entries, key)
		}
	}
}

func modTime(path string) time.Time {
	st, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return st.ModTime()
}
