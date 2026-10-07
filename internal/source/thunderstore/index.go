package thunderstore

import (
	"bufio"
	"bytes"
	"cmp"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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
	Deps   []string `json:"deps,omitempty"` // as stored on disk; a loaded listing keeps ids instead
	ids    []uint32
}

// pkg is what Mortar keeps of one listing: the fields search shows and install needs, latest version first.
type pkg struct {
	Owner     string `json:"owner"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Icon      string `json:"icon"`
	URL       string `json:"url"`
	Updated   string `json:"updated"`
	Created   string `json:"created"`
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
	tab        *depTable
}

// wirePackage is a v1 package listing as the site serves it.
type wirePackage struct {
	Name           string   `json:"name"`
	Owner          string   `json:"owner"`
	PackageURL     string   `json:"package_url"`
	DateUpdated    string   `json:"date_updated"`
	DateCreated    string   `json:"date_created"`
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
	buildLocksMu sync.Mutex
	buildLocks   = map[string]*sync.Mutex{}
)

func buildLock(key string) *sync.Mutex {
	buildLocksMu.Lock()
	defer buildLocksMu.Unlock()
	if buildLocks[key] == nil {
		buildLocks[key] = &sync.Mutex{}
	}
	return buildLocks[key]
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
	base := cmp.Or(d.URL, BaseURL)
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

// open starts a GET and returns its body, which the caller reads before calling done.
func (d Driver) open(ctx context.Context, url, ua string) (body io.Reader, done func(), err error) {
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	req.Header.Set("User-Agent", ua)
	client := cmp.Or(d.HTTP, http.DefaultClient)
	if client.CheckRedirect == nil {
		secured := *client
		secured.CheckRedirect = noDowngrade
		client = &secured
	}
	resp, err := client.Do(req)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	done = func() { _ = resp.Body.Close(); cancel() }
	if resp.StatusCode == http.StatusNotFound {
		done()
		return nil, nil, errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		done()
		return nil, nil, fmt.Errorf("thunderstore answered %s", resp.Status)
	}
	return io.LimitReader(resp.Body, source.MaxBody), done, nil
}

func (d Driver) get(ctx context.Context, url, ua string) ([]byte, error) {
	body, done, err := d.open(ctx, url, ua)
	if err != nil {
		return nil, err
	}
	defer done()
	return io.ReadAll(body)
}

// streamPackages decodes a package-list chunk one package at a time, so only the package being read is held.
func expectArray(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if tok != json.Delim('[') {
		return errors.New("package list is not a JSON array")
	}
	return nil
}

func (d Driver) streamPackages(ctx context.Context, url, ua string, fn func(*wirePackage) error) error {
	body, done, err := d.open(ctx, url, ua)
	if err != nil {
		return err
	}
	defer done()
	br := bufio.NewReader(body)
	var r io.Reader = br
	if magic, _ := br.Peek(2); len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		zr, err := gzip.NewReader(br)
		if err != nil {
			return err
		}
		r = io.LimitReader(zr, source.MaxBody*4)
	}
	dec := json.NewDecoder(r)
	if err := expectArray(dec); err != nil {
		return err
	}
	var w wirePackage
	for dec.More() {
		w = wirePackage{}
		if err := dec.Decode(&w); err != nil {
			return err
		}
		if err := fn(&w); err != nil {
			return err
		}
	}
	_, err = dec.Token()
	return err
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
	base := cmp.Or(d.URL, BaseURL)
	return base + "/c/" + key + "/api/v1/package-listing-index/"
}

// packages returns the community's listing, from the cache while it is under an hour old or the index blob is
// unchanged, else rebuilt from the chunks.
func (d Driver) packages(ctx context.Context, key, ua string) ([]pkg, error) {
	if !communityKey.MatchString(key) {
		return nil, fmt.Errorf("%q is not a Thunderstore community key", key)
	}
	// One build per community: a second caller waits, then finds the listing the first one wrote.
	mu := buildLock(key)
	mu.Lock()
	defer mu.Unlock()
	dir := filepath.Join(d.cacheRoot(), "thunderstore")
	metaPath := filepath.Join(dir, key+".meta.json")
	var meta cacheMeta
	if b, err := fsx.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(b, &meta)
	}
	// The schema tag keeps a listing built before creation dates were kept from being reused.
	pkgPath := func(hash string) string { return filepath.Join(dir, key+"-c2-"+hash+".json") }
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

// build downloads every chunk and writes the slimmed listing to path as it goes, so neither a chunk nor the whole
// listing is held in memory.
func (d Driver) build(ctx context.Context, chunks []string, path, ua string) (err error) {
	tmp, err := fsx.Create(path + ".tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(path + ".tmp")
		}
	}()
	bw := bufio.NewWriter(tmp)
	enc := json.NewEncoder(bw)
	if err = bw.WriteByte('['); err != nil {
		return err
	}
	first := true
	for _, u := range chunks {
		err = d.streamPackages(ctx, u, ua, func(w *wirePackage) error {
			if len(w.Versions) == 0 {
				return nil
			}
			if !first {
				if err := bw.WriteByte(','); err != nil {
					return err
				}
			}
			first = false
			return enc.Encode(slim(w))
		})
		if err != nil {
			return err
		}
	}
	if err = bw.WriteByte(']'); err != nil {
		return err
	}
	if err = bw.Flush(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return fsx.Rename(path+".tmp", path)
}

// slim keeps the fields of a wire package that search and install use.
func slim(w *wirePackage) pkg {
	p := pkg{
		Owner: w.Owner, Name: w.Name, URL: w.PackageURL, Updated: w.DateUpdated, Created: w.DateCreated, Rating: w.RatingScore,
		Hidden: w.IsDeprecated, Categories: w.Categories, Adult: w.HasNSFWContent, Summary: w.Versions[0].Description, Icon: w.Versions[0].Icon,
		Repo: source.GitHubRepo(w.Versions[0].WebsiteURL),
	}
	p.Versions = make([]version, len(w.Versions))
	for i, v := range w.Versions {
		p.Downloads += v.Downloads
		p.Versions[i] = version{Number: v.VersionNumber, Size: v.FileSize, Deps: v.Dependencies}
	}
	return p
}

func loadPackages(key, path string) ([]pkg, error) {
	memoMu.Lock()
	defer memoMu.Unlock()
	if m, ok := memo[key]; ok && m.path == path {
		return m.pk, nil
	}
	f, err := fsx.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(bufio.NewReader(f))
	if err := expectArray(dec); err != nil {
		return nil, err
	}
	var pk []pkg
	c := newDepCompactor()
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			return nil, err
		}
		c.add(&p)
		pk = append(pk, p)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	memo[key] = memoEntry{path, pk}
	return pk, nil
}

// depTable holds each distinct dependency string of a listing once; versions refer to them by index.
type depTable struct{ names []string }

// depCompactor moves every version's dependency strings into one shared table: most of a listing's bytes are the
// same "Owner-Name-1.2.3" pins named again by every version. Identical dependency lists, version numbers and the other
// strings that repeat across packages (owners, categories, repositories) are held once.
type depCompactor struct {
	tab   *depTable
	index map[string]uint32
	lists map[string][]uint32
	words map[string]string
	key   []byte
}

func newDepCompactor() *depCompactor {
	return &depCompactor{tab: &depTable{}, index: map[string]uint32{}, lists: map[string][]uint32{}, words: map[string]string{}}
}

// word returns the one copy of s held for the listing.
func (c *depCompactor) word(s string) string {
	if w, ok := c.words[s]; ok {
		return w
	}
	c.words[s] = s
	return s
}

func (c *depCompactor) add(p *pkg) {
	p.tab = c.tab
	// The decoder grows slices by doubling; the listing keeps them at their length.
	p.Versions = append(make([]version, 0, len(p.Versions)), p.Versions...)
	p.Categories = append(make([]string, 0, len(p.Categories)), p.Categories...)
	p.Owner, p.Repo = c.word(p.Owner), c.word(p.Repo)
	for i, cat := range p.Categories {
		p.Categories[i] = c.word(cat)
	}
	for j := range p.Versions {
		v := &p.Versions[j]
		v.Number = c.word(v.Number)
		if len(v.Deps) == 0 {
			continue
		}
		c.key = c.key[:0]
		for _, d := range v.Deps {
			id, ok := c.index[d]
			if !ok {
				id = uint32(len(c.tab.names) & math.MaxUint32)
				c.tab.names = append(c.tab.names, d)
				c.index[d] = id
			}
			c.key = binary.LittleEndian.AppendUint32(c.key, id)
		}
		ids, ok := c.lists[string(c.key)]
		if !ok {
			ids = make([]uint32, len(v.Deps))
			for k := range ids {
				ids[k] = binary.LittleEndian.Uint32(c.key[k*4:])
			}
			c.lists[string(c.key)] = ids
		}
		v.ids, v.Deps = ids, nil
	}
}

// depsOf is the dependency strings of one of p's versions.
func (p pkg) depsOf(v version) []string {
	if v.ids == nil {
		return v.Deps
	}
	out := make([]string, len(v.ids))
	for i, id := range v.ids {
		out[i] = p.tab.names[id]
	}
	return out
}
