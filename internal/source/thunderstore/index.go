package thunderstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// refreshAfter is how long a cached index is used without asking the site whether it changed.
const refreshAfter = time.Hour

// version is what Mortar keeps of one package version.
type version struct {
	Number string   `json:"v"`
	Size   int64    `json:"size,omitempty"`
	Deps   []string `json:"deps,omitempty"`
}

// pkg is what Mortar keeps of one listing: the fields search shows and install needs, latest version first.
type pkg struct {
	Owner     string `json:"owner"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Icon      string `json:"icon"`
	URL       string `json:"url"`
	Updated   string `json:"updated"`
	Rating    int    `json:"rating"`
	Downloads int    `json:"downloads"`
	// Hidden is a deprecated package: not searchable, but still resolvable as a dependency.
	Hidden   bool      `json:"hidden,omitempty"`
	Adult    bool      `json:"adult,omitempty"`
	Versions []version `json:"versions"`
}

// wirePackage is a v1 package listing as the site serves it.
type wirePackage struct {
	Name           string `json:"name"`
	Owner          string `json:"owner"`
	PackageURL     string `json:"package_url"`
	DateUpdated    string `json:"date_updated"`
	RatingScore    int    `json:"rating_score"`
	IsDeprecated   bool   `json:"is_deprecated"`
	HasNSFWContent bool   `json:"has_nsfw_content"`
	Versions       []struct {
		Description   string   `json:"description"`
		Icon          string   `json:"icon"`
		VersionNumber string   `json:"version_number"`
		Dependencies  []string `json:"dependencies"`
		Downloads     int      `json:"downloads"`
		FileSize      int64    `json:"file_size"`
	} `json:"versions"`
}

// cacheMeta records which index blob the cached package file was built from.
type cacheMeta struct {
	Hash    string    `json:"hash"`
	Fetched time.Time `json:"fetched"`
}

var (
	memoMu sync.Mutex
	memo   = map[string][]pkg{}
)

func (d Driver) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
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

func (d Driver) get(ctx context.Context, url, ua string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	client := d.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("thunderstore answered %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, source.MaxBody))
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// getJSON fetches url and decodes it, gunzipping a body that is still compressed.
func (d Driver) getJSON(ctx context.Context, url, ua string, out any) (hash string, err error) {
	raw, err := d.get(ctx, url, ua)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	body := raw
	if len(raw) > 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return "", err
		}
		if body, err = io.ReadAll(io.LimitReader(zr, source.MaxBody*4)); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(sum[:]), json.Unmarshal(body, out)
}

func (d Driver) indexURL(key string) string {
	base := d.URL
	if base == "" {
		base = BaseURL
	}
	return base + "/c/" + key + "/api/v1/package-listing-index/"
}

// packages returns the community's listing, from the cache while it is under an hour old or the index blob is
// unchanged, else rebuilt from the chunks.
func (d Driver) packages(ctx context.Context, key, ua string) ([]pkg, error) {
	dir := filepath.Join(d.cacheRoot(), "thunderstore")
	metaPath := filepath.Join(dir, key+".meta.json")
	var meta cacheMeta
	if b, err := fsx.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(b, &meta)
	}
	pkgPath := func(hash string) string { return filepath.Join(dir, key+"-"+hash+".json") }
	if meta.Hash != "" && d.now().Sub(meta.Fetched) < refreshAfter {
		if pk, err := loadPackages(pkgPath(meta.Hash)); err == nil {
			return pk, nil
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	var chunks []string
	hash, err := d.getJSON(ctx, d.indexURL(key), ua, &chunks)
	if err != nil {
		if meta.Hash != "" {
			if pk, lerr := loadPackages(pkgPath(meta.Hash)); lerr == nil {
				return pk, nil
			}
		}
		return nil, err
	}
	if hash != meta.Hash {
		if _, err := os.Stat(pkgPath(hash)); err != nil {
			if err := d.build(ctx, chunks, pkgPath(hash), ua); err != nil {
				return nil, err
			}
		}
		if meta.Hash != "" {
			_ = os.Remove(pkgPath(meta.Hash))
		}
	}
	nm, err := json.Marshal(cacheMeta{Hash: hash, Fetched: d.now()})
	if err != nil {
		return nil, err
	}
	if err := datadir.WriteFile(metaPath, nm, 0o644); err != nil {
		return nil, err
	}
	return loadPackages(pkgPath(hash))
}

// build downloads every chunk and writes the slimmed listing to path.
func (d Driver) build(ctx context.Context, chunks []string, path, ua string) error {
	var all []pkg
	for _, u := range chunks {
		var wire []wirePackage
		if _, err := d.getJSON(ctx, u, ua, &wire); err != nil {
			return err
		}
		for _, w := range wire {
			if len(w.Versions) == 0 {
				continue
			}
			p := pkg{
				Owner: w.Owner, Name: w.Name, URL: w.PackageURL, Updated: w.DateUpdated, Rating: w.RatingScore,
				Hidden: w.IsDeprecated, Adult: w.HasNSFWContent, Summary: w.Versions[0].Description, Icon: w.Versions[0].Icon,
			}
			for _, v := range w.Versions {
				p.Downloads += v.Downloads
				p.Versions = append(p.Versions, version{Number: v.VersionNumber, Size: v.FileSize, Deps: v.Dependencies})
			}
			all = append(all, p)
		}
	}
	b, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return datadir.WriteFile(path, b, 0o644)
}

func loadPackages(path string) ([]pkg, error) {
	memoMu.Lock()
	defer memoMu.Unlock()
	if pk, ok := memo[path]; ok {
		return pk, nil
	}
	b, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pk []pkg
	if err := json.Unmarshal(b, &pk); err != nil {
		return nil, err
	}
	memo[path] = pk
	return pk, nil
}
