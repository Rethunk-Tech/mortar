// Package meta reads SMAPI's update API and the Pathoschild/StardewModDataset, and caches both. Failures degrade
// to stale cache or an unknown result and never block the caller.
package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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

func (c *Client) cachePath(name string) (string, error) {
	dir := c.CacheDir
	if dir == "" {
		base, err := datadir.Dir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(base, "cache")
	}
	return filepath.Join(dir, name), nil
}

type entry[T any] struct {
	Fetched time.Time `json:"fetched"`
	Value   T         `json:"value"`
	// Build is the Mortar build that wrote a cache whose answers depend on Mortar's own logic.
	Build string `json:"build,omitempty"`
}

func readEntry[T any](path string) (entry[T], bool) {
	b, err := fsx.ReadFile(path)
	if err != nil {
		return entry[T]{}, false
	}
	// An entry with no "value" key is another shape under the same name (an older build's, or a foreign file), and
	// decoding it would serve a zero value as a fresh answer. A stored "value": null is still a value.
	var raw struct {
		Fetched time.Time       `json:"fetched"`
		Value   json.RawMessage `json:"value"`
		Build   string          `json:"build"`
	}
	if json.Unmarshal(b, &raw) != nil || raw.Fetched.IsZero() || raw.Value == nil {
		return entry[T]{}, false
	}
	e := entry[T]{Fetched: raw.Fetched, Build: raw.Build}
	if json.Unmarshal(raw.Value, &e.Value) != nil {
		return entry[T]{}, false
	}
	return e, true
}

// writeEntry replaces path without fsync: a cache that cannot be written only costs a refetch, and one a crash cuts
// short fails to parse and reads as absent, while a cold start writes hundreds of entries at once.
func writeEntry[T any](path string, e entry[T]) {
	b, err := json.Marshal(e)
	if err != nil || os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	_ = datadir.WriteCacheStream(path, 0o600, func(w io.Writer) error {
		_, err := w.Write(b)
		return err
	})
}

// Cached returns the entry at path while it is younger than ttl, else fetches; a failed fetch falls back to a
// stale entry, and only errors when there is none.
func Cached[T any](c *Client, name string, ttl time.Duration, fetch func() (T, error)) (T, error) {
	return cached(c, name, ttl, "", fetch)
}

// CachedForBuild is Cached for an answer that depends on how Mortar reads its source, so a cache another build wrote,
// or one from before builds were stamped, is fetched again and serves only while the fetch fails.
func CachedForBuild[T any](c *Client, name string, ttl time.Duration, fetch func() (T, error)) (T, error) {
	return cached(c, name, ttl, appversion.Build(), fetch)
}

// CachedStamped is Cached that also fetches again an entry whose stamp is not stamp; that entry still stands in when
// the fetch fails.
func CachedStamped[T any](c *Client, name, stamp string, ttl time.Duration, fetch func() (T, error)) (T, error) {
	return cached(c, name, ttl, stamp, fetch)
}

func cached[T any](c *Client, name string, ttl time.Duration, build string, fetch func() (T, error)) (T, error) {
	path, err := c.cachePath(name)
	if err != nil {
		return fetch()
	}
	old, ok := readEntry[T](path)
	if ok && c.young(old.Fetched, ttl) && old.Build == build {
		return old.Value, nil
	}
	v, err := fetch()
	if err != nil {
		if ok {
			return old.Value, nil
		}
		var zero T
		return zero, err
	}
	writeEntry(path, entry[T]{Fetched: c.now(), Value: v, Build: build})
	return v, nil
}

// Peek returns the entry at path however old, without fetching.
func Peek[T any](c *Client, name string) (T, bool) {
	path, err := c.cachePath(name)
	if err != nil {
		var zero T
		return zero, false
	}
	e, ok := readEntry[T](path)
	return e.Value, ok
}

// PeekStamped is Peek that also returns the stamp the entry was written with.
func PeekStamped[T any](c *Client, name string) (T, string, bool) {
	path, err := c.cachePath(name)
	if err != nil {
		var zero T
		return zero, "", false
	}
	e, ok := readEntry[T](path)
	return e.Value, e.Build, ok
}

// PeekFetched is Peek that also returns when the entry was fetched.
func PeekFetched[T any](c *Client, name string) (T, time.Time, bool) {
	path, err := c.cachePath(name)
	if err != nil {
		var zero T
		return zero, time.Time{}, false
	}
	e, ok := readEntry[T](path)
	return e.Value, e.Fetched, ok
}

// do sends req and returns at most limit bytes of a 200 response body.
func (c *Client) do(req *http.Request, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(req.Context(), requestTimeout)
	defer cancel()
	req.Header.Set("User-Agent", appversion.UserAgent(c.Version))
	resp, err := c.client().Do(req.WithContext(ctx))
	if err != nil {
		return nil, usererr.Wrap(usererr.Network, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return nil, usererr.Wrap(usererr.NotFound, &StatusError{Code: resp.StatusCode, Status: resp.Status})
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, usererr.Wrap(usererr.RateLimited, &StatusError{Code: resp.StatusCode, Status: resp.Status})
	}
	if resp.StatusCode != http.StatusOK {
		return nil, usererr.Wrap(usererr.Network, &StatusError{Code: resp.StatusCode, Status: resp.Status})
	}
	return fsx.ReadCapped(resp.Body, limit)
}

// StatusError is a non-200 answer.
type StatusError struct {
	Code   int
	Status string
}

func (e *StatusError) Error() string { return fmt.Sprintf("server answered %s", e.Status) }

// Fresh reports whether the entry at name exists and is younger than ttl.
func Fresh[T any](c *Client, name string, ttl time.Duration) bool {
	path, err := c.cachePath(name)
	if err != nil {
		return false
	}
	e, ok := readEntry[T](path)
	return ok && c.young(e.Fetched, ttl)
}

// young reports whether an entry fetched at t is younger than ttl; one stamped in the future (a clock set back) is not.
func (c *Client) young(t time.Time, ttl time.Duration) bool {
	age := c.now().Sub(t)
	return age >= 0 && age < ttl
}

// Put stores value at name as if it had just been fetched.
func Put[T any](c *Client, name string, value T) {
	if path, err := c.cachePath(name); err == nil {
		writeEntry(path, entry[T]{Fetched: c.now(), Value: value})
	}
}
