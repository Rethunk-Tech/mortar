// Package github reads a repository's public releases (60 calls an hour per IP without sign-in, more with the gh token
// when gh is signed in), so lookups are cached on disk and a stale answer stands in when GitHub cannot be reached or
// the limit is reached.
package github

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/netstate"

	"github.com/Rethunk-Tech/go-semverish"
)

const (
	apiBase    = "https://api.github.com"
	cacheTTL   = time.Hour
	apiTimeout = 8 * time.Second
	maxList    = 8 << 20
)

// Release is one entry of a repository's releases list.
type Release struct {
	Tag        string  `json:"tag_name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Body       string  `json:"body,omitempty"`
	Published  string  `json:"published_at,omitempty"`
	Assets     []Asset `json:"assets"`
}

// Asset is one downloadable file of a release.
type Asset struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	URL         string `json:"browser_download_url"`
	Downloads   int    `json:"download_count"`
	// Digest is "sha256:<hex>" when GitHub computed one for the asset.
	Digest string `json:"digest,omitempty"`
}

// RateLimitError is GitHub's unauthenticated limit; Reset is when calls work again (zero when GitHub did not say).
type RateLimitError struct{ Reset time.Time }

func (e *RateLimitError) Error() string {
	msg := "GitHub's rate limit for unauthenticated requests is used up"
	if !e.Reset.IsZero() {
		msg += "; try again after " + e.Reset.Local().Format("15:04")
	}
	return msg
}

// Rate is the quota GitHub last reported on a response, so Settings can show it without a request of its own.
type Rate struct {
	Known     bool      `json:"known"`
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}

var (
	rateMu   sync.Mutex
	lastRate Rate
)

// CurrentRate is the quota from the most recent GitHub response that carried one.
func CurrentRate() Rate {
	rateMu.Lock()
	defer rateMu.Unlock()
	return lastRate
}

func noteRate(resp *http.Response) {
	limit, e1 := strconv.Atoi(resp.Header.Get("X-Ratelimit-Limit"))
	left, e2 := strconv.Atoi(resp.Header.Get("X-Ratelimit-Remaining"))
	reset, e3 := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64)
	if e1 != nil || e2 != nil || e3 != nil {
		return
	}
	rateMu.Lock()
	lastRate = Rate{Known: true, Limit: limit, Remaining: left, Reset: time.Unix(reset, 0)}
	rateMu.Unlock()
}

// maxRetryAfter bounds a Retry-After in seconds so the Duration arithmetic cannot overflow.
const maxRetryAfter = 365 * 24 * 3600

// RetryAfter is when a Retry-After header (seconds or an HTTP date) says to try again; false when it is absent,
// malformed, negative or zero seconds, or absurdly long.
func RetryAfter(v string) (time.Time, bool) {
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		if n <= 0 || n > maxRetryAfter {
			return time.Time{}, false
		}
		return time.Now().Add(time.Duration(n) * time.Second), true
	}
	t, err := http.ParseTime(v)
	return t, err == nil
}

// RateLimited returns a *RateLimitError when resp is GitHub's limit answer, else nil.
func RateLimited(resp *http.Response) error {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return nil
	}
	if resp.Header.Get("X-Ratelimit-Remaining") != "0" && resp.Header.Get("Retry-After") == "" {
		return nil
	}
	e := &RateLimitError{}
	if t, ok := RetryAfter(resp.Header.Get("Retry-After")); ok {
		e.Reset = t
	} else if secs, err := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64); err == nil {
		e.Reset = time.Unix(secs, 0)
	}
	return e
}

// FetchReleases reads a releases list from url.
func FetchReleases(ctx context.Context, hc *http.Client, url string) ([]Release, error) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	DefaultAuth.Apply(req)
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := noDowngrade(hc).Do(req)
	netstate.Record("github", err)
	if err != nil {
		return nil, fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	noteRate(resp)
	if err := RateLimited(resp); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub's releases API answered %s", resp.Status)
	}
	var all []Release
	if err := json.NewDecoder(safefs.CapReader(resp.Body, maxList)).Decode(&all); err != nil {
		return nil, fmt.Errorf("read GitHub's releases response: %w", err)
	}
	return all, nil
}

// Client looks up and downloads releases. The zero value uses api.github.com and <datadir>/cache.
type Client struct {
	HTTP     *http.Client
	CacheDir string
	// APIBase overrides https://api.github.com, for tests.
	APIBase string
	Now     func() time.Time
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now().UTC()
}

var nonKey = regexp.MustCompile(`[^a-z0-9.]+`)

// maxSlug bounds the readable part of a key, keeping the folder name under the file system's limit.
const maxSlug = 100

// Key is the store key of a GitHub asset: github-<owner>-<repo>-<tag>-<asset>, lower-cased with every other
// character run folded to a dash so it satisfies the store's key rules, then a hash of the exact names, since the
// folding alone maps a-b/c and a/b-c to one key.
func Key(owner, repo, tag, asset string) string {
	slug := nonKey.ReplaceAllString(strings.ToLower(strings.Join([]string{owner, repo, tag, asset}, "-")), "-")
	slug = strings.Trim(slug[:min(len(slug), maxSlug)], "-")
	sum := sha256.Sum256([]byte(strings.ToLower(owner+"/"+repo) + "@" + tag + "/" + asset))
	return "github-" + slug + "-" + hex.EncodeToString(sum[:6])
}

// cacheFormat stamps the release cache and is bumped when Release gains a field an older file lacks: that file is
// fetched again however young, still stands in when the lookup fails, and is never read for the missing field. Format 2
// added asset download counts.
const cacheFormat = 2

// releasesPerPage is how many releases one lookup reads, the newest ones.
const releasesPerPage = 30

// cache is the release cache's handle; the clock is the client's, so tests can set it.
func (c *Client) cache() *diskcache.Cache {
	return &diskcache.Cache{Dir: c.CacheDir, Now: c.now}
}

func releasesName(owner, repo string) string {
	return Key(owner, repo, "", "") + "-releases-v2.json"
}

// CachedReleases lists owner/repo's releases from the stored answer however old it is, never asking GitHub. all is
// whether the list holds every release the repository has, which a full page may not.
func (c *Client) CachedReleases(owner, repo string) (releases []Release, all, ok bool) {
	e, ok := diskcache.Peek[[]Release](c.cache(), releasesName(owner, repo))
	return e.Value, len(e.Value) < releasesPerPage, ok && e.Stamp == strconv.Itoa(cacheFormat)
}

// Releases lists owner/repo's releases, newest first. An answer younger than an hour comes from the cache and a
// stale one stands in when the lookup fails, rate limit included.
func (c *Client) Releases(ctx context.Context, owner, repo string) ([]Release, error) {
	base := cmp.Or(c.APIBase, apiBase)
	return diskcache.CachedStamped(c.cache(), releasesName(owner, repo), strconv.Itoa(cacheFormat), cacheTTL, func() ([]Release, error) {
		return FetchReleases(ctx, c.HTTP, base+"/repos/"+owner+"/"+repo+"/releases?per_page="+strconv.Itoa(releasesPerPage))
	})
}

// ErrNoRelease and ErrNoArchive say why Select found nothing to install.
var (
	ErrNoRelease = errors.New("GitHub lists no matching stable release")
	ErrNoArchive = errors.New("the release has no archive asset (zip, rar, 7z, tar, gz, xz, zst, bz2, lz4 or lzma)")
)

func tagVersion(tag string) string { return strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V") }

// IsArchive reports whether a is an archive Mortar reads by name, and by content type where GitHub set one.
func IsArchive(a Asset) bool {
	if !archive.HasExtension(a.Name) {
		return false
	}
	// GitHub labels an asset it cannot type application/octet-stream, so that counts as an archive here.
	if a.ContentType == "" || archive.IsMediaType(a.ContentType) {
		return true
	}
	mt, _, err := mime.ParseMediaType(a.ContentType)
	return err == nil && mt == "application/octet-stream"
}

// Select picks the newest non-draft release whose tag is version (any when version is empty) and returns it with its
// archive assets: one when the release ships one, several when the user must choose. A prerelease is picked only when
// prereleases are asked for or its tag is named.
func Select(all []Release, version string, prerelease bool) (Release, []Asset, error) {
	for _, r := range all {
		if r.Draft || (r.Prerelease && !prerelease && version == "") {
			continue
		}
		if version != "" {
			if cmp, ok := semverish.Compare(tagVersion(r.Tag), tagVersion(version)); !ok || cmp != 0 {
				continue
			}
		}
		var assets []Asset
		for _, a := range r.Assets {
			if IsArchive(a) {
				assets = append(assets, a)
			}
		}
		if len(assets) == 0 {
			return r, nil, ErrNoArchive
		}
		return r, assets, nil
	}
	return Release{}, nil, ErrNoRelease
}
