package thunderstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	// Repo is the GitHub repository the newest version's website links, as "owner/repo".
	Repo     string    `json:"repo,omitempty"`
	Hidden   bool      `json:"hidden,omitempty"`
	Adult    bool      `json:"adult,omitempty"`
	Versions []version `json:"versions"`
	// Categories are the package's site categories.
	Categories []string `json:"categories,omitempty"`
}

// wirePackage is a v1 package listing as the site serves it.
type wirePackage struct {
	Name           string   `json:"name"`
	Owner          string   `json:"owner"`
	PackageURL     string   `json:"package_url"`
	DateUpdated    string   `json:"date_updated"`
	RatingScore    int      `json:"rating_score"`
	IsDeprecated   bool     `json:"is_deprecated"`
	HasNSFWContent bool     `json:"has_nsfw_content"`
	Categories     []string `json:"categories"`
	Versions       []struct {
		Description   string   `json:"description"`
		Icon          string   `json:"icon"`
		VersionNumber string   `json:"version_number"`
		Dependencies  []string `json:"dependencies"`
		Downloads     int      `json:"downloads"`
		WebsiteURL    string   `json:"website_url"`
		FileSize      int64    `json:"file_size"`
	} `json:"versions"`
}

// cacheMeta records which index blob the cached package file was built from.
type cacheMeta struct {
	Hash    string    `json:"hash"`
	Fetched time.Time `json:"fetched"`
}

type memoEntry struct {
	path string
	pk   []pkg
}

var (
	memoMu sync.Mutex
	// memo holds one listing per community key, the one built from the blob at path.
	memo = map[string]memoEntry{}
)

// noDowngrade refuses a redirect from https to another scheme.
func noDowngrade(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("too many redirects")
	}
	if via[0].URL.Scheme == "https" && req.URL.Scheme != "https" {
		return fmt.Errorf("refusing a redirect to %s://", req.URL.Scheme)
	}
	return nil
}

const (
	maxRedirects = 10
	maxChunks    = 500
)

var communityKey = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// chunkAllowed reports whether the index may point Mortar at raw: Thunderstore's own https hosts, or the host the
// index itself came from.
func (d Driver) chunkAllowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	base := d.URL
	if base == "" {
		base = BaseURL
	}
	if b, err := url.Parse(base); err == nil && u.Scheme == b.Scheme && u.Host == b.Host {
		return true
	}
	host := u.Hostname()
	return u.Scheme == "https" && (host == "thunderstore.io" || strings.HasSuffix(host, ".thunderstore.io"))
}

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
	if client.CheckRedirect == nil {
		secured := *client
		secured.CheckRedirect = noDowngrade
		client = &secured
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
	if !communityKey.MatchString(key) {
		return nil, fmt.Errorf("%q is not a Thunderstore community key", key)
	}
	dir := filepath.Join(d.cacheRoot(), "thunderstore")
	metaPath := filepath.Join(dir, key+".meta.json")
	var meta cacheMeta
	if b, err := fsx.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(b, &meta)
	}
	// The schema tag keeps a listing built before categories were kept from being reused.
	pkgPath := func(hash string) string { return filepath.Join(dir, key+"-c1-"+hash+".json") }
	if meta.Hash != "" && d.now().Sub(meta.Fetched) < refreshAfter {
		if pk, err := loadPackages(key, pkgPath(meta.Hash)); err == nil {
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
			if pk, lerr := loadPackages(key, pkgPath(meta.Hash)); lerr == nil {
				return pk, nil
			}
		}
		return nil, err
	}
	if len(chunks) > maxChunks {
		return nil, fmt.Errorf("index lists %d chunks", len(chunks))
	}
	for _, c := range chunks {
		if !d.chunkAllowed(c) {
			return nil, fmt.Errorf("index names a chunk outside Thunderstore: %s", c)
		}
	}
	if _, err := os.Stat(pkgPath(hash)); err != nil {
		if err := d.build(ctx, chunks, pkgPath(hash), ua); err != nil {
			return nil, err
		}
	}
	if meta.Hash != "" && hash != meta.Hash {
		_ = os.Remove(pkgPath(meta.Hash))
	}
	nm, err := json.Marshal(cacheMeta{Hash: hash, Fetched: d.now()})
	if err != nil {
		return nil, err
	}
	if err := datadir.WriteFile(metaPath, nm, 0o644); err != nil {
		return nil, err
	}
	return loadPackages(key, pkgPath(hash))
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
				Hidden: w.IsDeprecated, Categories: w.Categories, Adult: w.HasNSFWContent, Summary: w.Versions[0].Description, Icon: w.Versions[0].Icon,
				Repo: source.GitHubRepo(w.Versions[0].WebsiteURL),
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

func loadPackages(key, path string) ([]pkg, error) {
	memoMu.Lock()
	defer memoMu.Unlock()
	if m, ok := memo[key]; ok && m.path == path {
		return m.pk, nil
	}
	b, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pk []pkg
	if err := json.Unmarshal(b, &pk); err != nil {
		return nil, err
	}
	memo[key] = memoEntry{path, pk}
	return pk, nil
}
