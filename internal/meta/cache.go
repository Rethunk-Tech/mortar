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
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const requestTimeout = 20 * time.Second

// Client fetches and caches mod metadata. The zero value uses the public endpoints and <datadir>/cache.
type Client struct {
	HTTP     *http.Client
	CacheDir string
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
}

func readEntry[T any](path string) (entry[T], bool) {
	b, err := fsx.ReadFile(path)
	if err != nil {
		return entry[T]{}, false
	}
	var e entry[T]
	if json.Unmarshal(b, &e) != nil || e.Fetched.IsZero() {
		return entry[T]{}, false
	}
	return e, true
}

func writeEntry[T any](path string, e entry[T]) {
	// A cache that cannot be written only costs a refetch.
	if os.MkdirAll(filepath.Dir(path), 0o700) == nil {
		_ = datadir.WriteJSON(path, e)
	}
}

// Cached returns the entry at path while it is younger than ttl, else fetches; a failed fetch falls back to a
// stale entry, and only errors when there is none.
func Cached[T any](c *Client, name string, ttl time.Duration, fetch func() (T, error)) (T, error) {
	path, err := c.cachePath(name)
	if err != nil {
		return fetch()
	}
	old, ok := readEntry[T](path)
	if ok && c.now().Sub(old.Fetched) < ttl {
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
	writeEntry(path, entry[T]{Fetched: c.now(), Value: v})
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

// do sends req and returns at most limit bytes of a 200 response body.
func (c *Client) do(req *http.Request, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(req.Context(), requestTimeout)
	defer cancel()
	resp, err := c.client().Do(req.WithContext(ctx))
	if err != nil {
		return nil, usererr.Wrap(usererr.Network, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return nil, usererr.Wrap(usererr.NotFound, &StatusError{Code: resp.StatusCode, Status: resp.Status})
	}
	if resp.StatusCode != http.StatusOK {
		return nil, usererr.Wrap(usererr.Network, &StatusError{Code: resp.StatusCode, Status: resp.Status})
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
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
	return ok && c.now().Sub(e.Fetched) < ttl
}

// Put stores value at name as if it had just been fetched.
func Put[T any](c *Client, name string, value T) {
	if path, err := c.cachePath(name); err == nil {
		writeEntry(path, entry[T]{Fetched: c.now(), Value: value})
	}
}
