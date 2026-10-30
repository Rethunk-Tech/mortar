// Package meta reads SMAPI's update API and the Pathoschild/StardewModDataset, and caches both. Failures degrade
// to stale cache or an unknown result and never block the caller.
package meta

import (
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/go-smapiweb"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// Client fetches and caches mod metadata. The zero value uses the public endpoints and <datadir>/cache.
type Client struct {
	HTTP     *http.Client
	CacheDir string
	// Version is Mortar's, sent to every site so it can say who is asking.
	Version string
	// Endpoint overrides, for tests.
	IndexURL   string
	PageBase   string
	UpdatesURL string
	CompatURL  string
	// NexusURL is the Nexus API origin; empty means the public one.
	NexusURL  string
	Now       func() time.Time
	index     indexMemo
	nexusOnce sync.Once
	nexus     *nexusmods.Client
	compat    compatMemo
	// updatesMu serializes update checks so concurrent profiles share one fetch and one cache write.
	updatesMu sync.Mutex
	// updatesPause holds off smapi.io after a failed ask; until then stale cache or unknown is served.
	updatesPause time.Time
	// updatesRefreshing is set while answers past their hour are asked again in the background.
	updatesRefreshing atomic.Bool
}

func (c *Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// web is the stateless client for the same endpoints; Mortar's own cache sits around its calls.
func (c *Client) web() *smapiweb.Client {
	return &smapiweb.Client{HTTP: c.client(), UpdatesURL: c.UpdatesURL, CompatURL: c.CompatURL, IndexURL: c.IndexURL, PageBase: c.PageBase}
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now().UTC()
}

// Cache is the client's cache: CacheDir, else <datadir>/cache. With no data folder it is disabled, and every
// answer is fetched.
func (c *Client) Cache() *diskcache.Cache {
	dir := c.CacheDir
	if dir == "" {
		if base, err := datadir.Dir(); err == nil {
			dir = filepath.Join(base, "cache")
		}
	}
	return &diskcache.Cache{Dir: dir, Now: c.Now}
}

// young reports whether an entry fetched at t is younger than ttl; one stamped in the future (a clock set back) is not.
func (c *Client) young(t time.Time, ttl time.Duration) bool {
	age := c.now().Sub(t)
	return age >= 0 && age < ttl
}
