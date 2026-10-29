// Package github is Mortar's face on go-github, which reads a repository's public releases (60 calls an hour per IP
// without sign-in, more with the gh token when gh is signed in). It holds what is Mortar's own: the gh login, the
// data-folder cache, reachability, the wording players read, and which assets count as an archive.
package github

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/netstate"
)

type (
	Release = gh.Release
	Asset   = gh.Asset
	Rate    = gh.Rate
	Auth    = gh.Auth
)

// DefaultAuth authorizes every request this package makes to GitHub. Mortar lends requests the token of the player's
// `gh` login; the module sends none until told how to get one.
var DefaultAuth = func() *Auth {
	gh.DefaultAuth.Run = gh.GHCLI
	return gh.DefaultAuth
}()

// MaxChangelog bounds how many releases ReleasesBetween returns.
const MaxChangelog = gh.MaxChangelog

func Key(owner, repo, tag, asset string) string       { return gh.Key(owner, repo, tag, asset) }
func Shape(name string) string                        { return gh.Shape(name) }
func Installable(assets []Asset, goos string) []Asset { return gh.Installable(assets, goos) }
func CurrentRate() Rate                               { return gh.CurrentRate() }
func Observe(resp *http.Response) error               { return gh.Observe(resp) }
func IsGitHubHost(host string) bool                   { return gh.IsGitHubHost(host) }
func ZipNames(ctx context.Context, hc *http.Client, a Asset) ([]string, error) {
	return gh.ZipNames(ctx, hc, a)
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

// Unwrap lets errors.As find the module's *httpkit.RateLimitError behind GitHub's wording.
func (e *RateLimitError) Unwrap() error { return &httpkit.RateLimitError{Reset: e.Reset} }

// ErrNoRelease and ErrNoArchive say why Select found nothing to install.
var (
	ErrNoRelease = errors.New("GitHub lists no matching stable release")
	ErrNoArchive = errors.New("the release has no archive asset (zip, rar, 7z, tar, gz, xz, zst, bz2, lz4 or lzma)")
)

// Client looks up releases. The zero value uses api.github.com and <datadir>/cache.
type Client struct {
	HTTP     *http.Client
	CacheDir string
	// APIBase overrides https://api.github.com, for tests.
	APIBase string
	Now     func() time.Time
}

// prepared is the module's client with Mortar's policies applied.
func (c *Client) prepared() *gh.Client {
	dir := c.CacheDir
	if dir == "" {
		if data, err := datadir.Dir(); err == nil {
			dir = filepath.Join(data, "cache")
		}
	}
	return &gh.Client{HTTP: recording(c.HTTP), CacheDir: dir, APIBase: c.APIBase, Now: c.Now}
}

// recording makes every round trip of hc tell netstate whether GitHub was reached.
func recording(hc *http.Client) *http.Client {
	cp := *cmp.Or(hc, http.DefaultClient)
	cp.Transport = &httpkit.Transport{Base: cp.Transport, OnResult: func(_ *http.Request, _ *http.Response, err error) {
		netstate.RecordTransport("github", err)
	}}
	return &cp
}

// words puts the module's errors in the wording players read. api names the endpoint in a status answer.
func words(err error, api string) error {
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return &RateLimitError{Reset: rl.Reset}
	}
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return fmt.Errorf("GitHub's %s API answered %s", api, se.Status)
	}
	if _, ok := errors.AsType[*url.Error](err); ok {
		return fmt.Errorf("could not reach GitHub: %w", err)
	}
	if errors.Is(err, gh.ErrNoRelease) {
		return ErrNoRelease
	}
	return err
}

// FetchReleases reads a releases list from url.
func FetchReleases(ctx context.Context, hc *http.Client, url string) ([]Release, error) {
	all, err := gh.FetchReleases(ctx, recording(hc), url)
	return all, words(err, "releases")
}

// CachedReleases lists owner/repo's releases from the stored answer however old it is, never asking GitHub.
func (c *Client) CachedReleases(owner, repo string) (releases []Release, all, ok bool) {
	return c.prepared().CachedReleases(owner, repo)
}

// Releases lists owner/repo's releases, newest first.
func (c *Client) Releases(ctx context.Context, owner, repo string) ([]Release, error) {
	all, err := c.prepared().Releases(ctx, owner, repo)
	return all, words(err, "releases")
}

// ReleaseNotes returns the markdown body of owner/repo's release tagged tag.
func (c *Client) ReleaseNotes(ctx context.Context, owner, repo, tag string) (string, error) {
	notes, err := c.prepared().ReleaseNotes(ctx, owner, repo, tag)
	return notes, words(err, "release")
}

// ReleasesBetween returns owner/repo's published releases newer than installed and not newer than latest.
func (c *Client) ReleasesBetween(ctx context.Context, owner, repo, installed, latest string) ([]Release, error) {
	rels, err := c.prepared().ReleasesBetween(ctx, owner, repo, installed, latest)
	return rels, words(err, "releases")
}

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
	r, assets, err := gh.Select(all, version, prerelease, IsArchive)
	switch {
	case errors.Is(err, gh.ErrNoAsset):
		err = ErrNoArchive
	case errors.Is(err, gh.ErrNoRelease):
		err = ErrNoRelease
	}
	return r, assets, err
}
