// Package modpic caches Nexus mod pictures on disk and serves them to the UI.
package modpic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// Path is the URL prefix cached mod pictures are served under.
const Path = "/mod-picture/"

// MaxSize caps a cached picture, in bytes.
const MaxSize = 8 << 20

// MaxCacheBytes is the on-disk budget for cache/modpic; New drops oldest files until the folder fits.
const MaxCacheBytes = 64 << 20

const workers = 4

var allowedHosts = map[string]struct{}{
	"staticdelivery.nexusmods.com": {},
	"images.nexusmods.com":         {},
}

var imageTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/webp": {},
	"image/gif":  {},
}

// Cache stores Nexus mod pictures under the data dir's cache/.
type Cache struct {
	dir   string
	http  *http.Client
	slots chan struct{}
}

// New keeps pictures in <dataDir>/cache/modpic.
func New(dataDir string, client *http.Client) *Cache {
	if client == nil {
		client = http.DefaultClient
	}
	c := &Cache{
		dir:   filepath.Join(dataDir, "cache", "modpic"),
		http:  client,
		slots: make(chan struct{}, workers),
	}
	c.pruneTo(MaxCacheBytes)
	return c
}

func (c *Cache) pruneTo(budget int64) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	type item struct {
		name string
		mod  time.Time
		size int64
	}
	var files []item
	var total int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		files = append(files, item{name: e.Name(), mod: info.ModTime(), size: info.Size()})
		total += info.Size()
	}
	if total <= budget {
		return
	}
	slices.SortFunc(files, func(a, b item) int {
		if a.mod.Before(b.mod) {
			return -1
		}
		if a.mod.After(b.mod) {
			return 1
		}
		return strings.Compare(a.name, b.name)
	})
	for _, f := range files {
		if total <= budget {
			return
		}
		if err := os.Remove(filepath.Join(c.dir, f.name)); err != nil {
			continue
		}
		total -= f.size
	}
}

// AssetURL is the app-local URL for a remote picture, or empty when the remote URL is not cacheable.
func AssetURL(picture string) string {
	if _, err := parsePicture(picture); err != nil {
		return ""
	}
	return Path + "?u=" + url.QueryEscape(picture)
}

func key(picture string) string {
	sum := sha256.Sum256([]byte(picture))
	return hex.EncodeToString(sum[:])
}

func parsePicture(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" || u.User != nil || u.Hostname() == "" || u.EscapedPath() == "" || strings.Contains(u.Path, "..") {
		return nil, fmt.Errorf("the picture URL is not a Nexus image")
	}
	if _, ok := allowedHosts[strings.ToLower(u.Hostname())]; !ok {
		return nil, fmt.Errorf("the picture host is not a Nexus CDN")
	}
	return u, nil
}

func (c *Cache) file(picture string) string {
	return filepath.Join(c.dir, key(picture))
}

// Ensure fetches picture when it is not already on disk. Missing or invalid URLs are ignored.
func (c *Cache) Ensure(ctx context.Context, picture string) {
	if c == nil || picture == "" {
		return
	}
	_, _, err := c.get(ctx, picture)
	_ = err
}

func (c *Cache) get(ctx context.Context, picture string) ([]byte, string, error) {
	if _, err := parsePicture(picture); err != nil {
		return nil, "", err
	}
	path := c.file(picture)
	if b, typ, err := readFile(path); err == nil {
		return b, typ, nil
	}
	select {
	case c.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, "", ctx.Err()
	}
	defer func() { <-c.slots }()
	if b, typ, err := readFile(path); err == nil {
		return b, typ, nil
	}
	return c.fetch(ctx, picture, path)
}

func readFile(path string) ([]byte, string, error) {
	f, err := fsx.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxSize+1))
	if err != nil {
		return nil, "", err
	}
	if len(b) > MaxSize {
		return nil, "", fmt.Errorf("the image is larger than %d MB", MaxSize>>20)
	}
	typ := http.DetectContentType(b)
	if _, ok := imageTypes[typ]; !ok {
		return nil, "", errNotImage
	}
	return b, typ, nil
}

var errNotImage = fmt.Errorf("the picture is not a PNG, JPEG, WebP or GIF image")

func (c *Cache) fetch(ctx context.Context, picture, path string) ([]byte, string, error) {
	u, err := parsePicture(picture)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+u.Host+u.RequestURI(), nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("the picture server answered %s", resp.Status)
	}
	if resp.ContentLength > MaxSize {
		return nil, "", fmt.Errorf("the image is larger than %d MB", MaxSize>>20)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxSize+1))
	if err != nil {
		return nil, "", err
	}
	if len(b) > MaxSize {
		return nil, "", fmt.Errorf("the image is larger than %d MB", MaxSize>>20)
	}
	typ := http.DetectContentType(b)
	if _, ok := imageTypes[typ]; !ok {
		return nil, "", errNotImage
	}
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return nil, "", err
	}
	if err := datadir.WriteFile(path, b, 0o600); err != nil {
		return nil, "", err
	}
	return b, typ, nil
}

// Middleware serves GET /mod-picture/?u=<picture URL>: the cached bytes, fetching once if missing.
func Middleware(cache func() *Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rest, ok := strings.CutPrefix(r.URL.Path, Path)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			c := cache()
			if r.Method != http.MethodGet || c == nil || rest != "" || strings.Contains(r.URL.Path, "..") {
				http.NotFound(w, r)
				return
			}
			picture := r.URL.Query().Get("u")
			b, typ, err := c.get(r.Context(), picture)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", typ)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(b))
		})
	}
}
