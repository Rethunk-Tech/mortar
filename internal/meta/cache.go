// Package meta reads SMAPI's update API and the Pathoschild/StardewModDataset, and caches both. Failures degrade
// to stale cache or an unknown result and never block the caller.
package meta

import (
	"context"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

const requestTimeout = 20 * time.Second

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
	Now        func() time.Time
	index      indexMemo
	compat     compatMemo
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

// do sends req and returns at most limit bytes of a 200 response body.
func (c *Client) do(req *http.Request, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(req.Context(), requestTimeout)
	defer cancel()
	return httpkit.DoCapped(c.client(), req.WithContext(ctx), limit, nil)
}

// young reports whether an entry fetched at t is younger than ttl; one stamped in the future (a clock set back) is not.
func (c *Client) young(t time.Time, ttl time.Duration) bool {
	age := c.now().Sub(t)
	return age >= 0 && age < ttl
}
