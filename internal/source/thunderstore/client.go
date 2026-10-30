package thunderstore

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/Rethunk-Tech/go-safefs"
	tstore "github.com/Rethunk-Tech/go-thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

var (
	clientsMu sync.Mutex
	// clients holds the one Client per cache folder: two on one folder would not share build locks or the listing
	// held in memory.
	clients = map[string]*tstore.Client{}
	// onRefreshed is told, with the community key, when a background refresh replaced the stored listing.
	onRefreshed func(key string)
)

// Open makes the Client every Driver on d's cache folder shares, sending Mortar's user agent for version. Mortar
// calls it at start, before any Driver on the folder is used, so a caller that knows no version cannot name the
// client.
func (d Driver) Open(version string) (*tstore.Client, error) { return d.client(version) }

// client is the Client for d's cache folder, made on first use with d's site, HTTP client and clock.
func (d Driver) client(version string) (*tstore.Client, error) {
	root := d.cacheRoot()
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if c, ok := clients[root]; ok {
		return c, nil
	}
	c, err := tstore.New(tstore.Config{
		CacheDir: root, HTTP: d.HTTP, BaseURL: d.URL, UserAgent: source.UserAgentWithSite(version), Now: d.Now,
		Refreshed: func(key string) {
			clientsMu.Lock()
			notify := onRefreshed
			clientsMu.Unlock()
			if notify != nil {
				notify(key)
			}
		},
		Built: func(key, path string) {
			dir := filepath.Dir(path)
			_ = safefs.Remove(lookedPath(dir, key))
			refreshDerived(dir, key, path)
		},
	})
	if err != nil {
		return nil, err
	}
	clients[root] = c
	return c, nil
}

// OnRefreshed sets the function told when a background refresh brings in a changed listing, so searches answered from
// the stale one can run again.
func OnRefreshed(f func(key string)) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	onRefreshed = f
}

func (d Driver) cacheRoot() string {
	if d.CacheDir != "" {
		return d.CacheDir
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "mortar")
}

// packages is the community's listing, held in memory while in use: Browse searches it on every keystroke.
func (d Driver) packages(ctx context.Context, key, version string) ([]tstore.Package, error) {
	c, err := d.client(version)
	if err != nil {
		return nil, err
	}
	x, err := c.Index(ctx, key)
	if err != nil {
		return nil, err
	}
	return x.Packages(), nil
}

// Refresh rebuilds the community's stored listing now, without waiting for the refresh window; searches keep answering
// from the stored one until the rebuild lands.
func (d Driver) Refresh(key, version string) {
	if c, err := d.client(version); err == nil {
		c.Refresh(key)
	}
}
