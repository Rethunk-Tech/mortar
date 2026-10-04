// Package github reads a repository's public releases without sign-in (60 calls an hour per IP), so lookups are
// cached on disk and a stale answer stands in when GitHub cannot be reached or the limit is reached.
package github

import (
	"context"
	"crypto/md5" // #nosec G501 -- Content-MD5 is MD5 by RFC 1864
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

const (
	apiBase    = "https://api.github.com"
	cacheTTL   = time.Hour
	apiTimeout = 8 * time.Second
	maxList    = 8 << 20
)

// downloadIdle is how long a GitHub asset body may sit idle before Download cancels the copy.
var downloadIdle = 30 * time.Second

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

// RateLimited returns a *RateLimitError when resp is GitHub's limit answer, else nil.
func RateLimited(resp *http.Response) error {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return nil
	}
	if resp.Header.Get("X-Ratelimit-Remaining") != "0" && resp.Header.Get("Retry-After") == "" {
		return nil
	}
	e := &RateLimitError{}
	if secs, err := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64); err == nil {
		e.Reset = time.Unix(secs, 0)
	} else if secs, err := strconv.ParseInt(resp.Header.Get("Retry-After"), 10, 64); err == nil {
		e.Reset = time.Now().Add(time.Duration(secs) * time.Second)
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
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := RateLimited(resp); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub's releases API answered %s", resp.Status)
	}
	var all []Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxList)).Decode(&all); err != nil {
		return nil, fmt.Errorf("read GitHub's releases response: %w", err)
	}
	return all, nil
}

// Progress reports bytes received so far and the expected total (0 when unknown).
type Progress func(done, total int64)

const downloadRestarts = 3

// ErrLinkExpired is a CDN or host refusing this URL; Nexus then needs a fresh DownloadLinks call.
var ErrLinkExpired = errors.New("the download link has expired")

type resumeMeta struct {
	ExpectedSize int64  `json:"expectedSize"`
	ETag         string `json:"etag"`
	URL          string `json:"url"`
	Hash         string `json:"hash"`
	MD5          string `json:"md5,omitempty"`
}

// ResumeSuffix ends the sidecar that records a partial download so a later Download can resume it.
const ResumeSuffix = ".resume.json"

// ResumeSidecar is the sidecar path of the download at path.
func ResumeSidecar(path string) string { return path + ResumeSuffix }

func loadResume(path string) resumeMeta {
	var m resumeMeta
	if _, err := datadir.ReadJSON(ResumeSidecar(path), &m); err != nil {
		return resumeMeta{}
	}
	return m
}

func saveResume(path string, m resumeMeta) {
	_ = datadir.WriteJSON(ResumeSidecar(path), m)
}

func dropDownload(path string) {
	_ = os.Remove(path)
	_ = os.Remove(ResumeSidecar(path))
}

// Download streams url to dest, appending with Range when a partial and the server allow it, and
// failing when the body exceeds limit bytes.
func Download(ctx context.Context, hc *http.Client, url, dest string, limit int64, progress Progress) error {
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Minute}
	}
	for n := 0; n <= downloadRestarts; n++ {
		err, retry := downloadOnce(ctx, hc, url, dest, limit, progress)
		if retry {
			dropDownload(dest)
			continue
		}
		return err
	}
	return fmt.Errorf("the download server sent a range Mortar could not resume")
}

func downloadOnce(ctx context.Context, hc *http.Client, url, dest string, limit int64, progress Progress) (err error, retry bool) {
	meta := loadResume(dest)
	if meta.URL == "" {
		meta.URL = url
	}
	offset := int64(0)
	if st, err := os.Stat(dest); err == nil {
		offset = st.Size()
	}
	if meta.ExpectedSize > 0 && offset > meta.ExpectedSize {
		dropDownload(dest)
		offset = 0
		meta = resumeMeta{URL: url}
	}
	if meta.ExpectedSize > 0 && offset == meta.ExpectedSize && offset > 0 {
		return verifyDownload(dest, meta), false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err, false
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err, false
	}
	defer func() { _ = resp.Body.Close() }()
	if err := RateLimited(resp); err != nil {
		return err, false
	}

	switch resp.StatusCode {
	case http.StatusOK:
		if offset > 0 {
			dropDownload(dest)
			offset = 0
		}
	case http.StatusPartialContent:
		start, total, ok := parseContentRange(resp.Header.Get("Content-Range"))
		if !ok || start != offset {
			return nil, true
		}
		if total > 0 {
			meta.ExpectedSize = total
		}
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusGone:
		return ErrLinkExpired, false
	case http.StatusRequestedRangeNotSatisfiable:
		return nil, true
	default:
		return fmt.Errorf("server answered %s", resp.Status), false
	}

	tooLarge := fmt.Errorf("larger than %d MiB", limit>>20)
	if etag := resp.Header.Get("ETag"); etag != "" {
		if meta.ETag != "" && meta.ETag != etag && offset > 0 && resp.StatusCode == http.StatusPartialContent {
			return nil, true
		}
		meta.ETag = etag
	}
	if h := hashFrom(resp.Header); h != "" {
		meta.Hash = h
	}
	if v := strings.TrimSpace(resp.Header.Get("Content-MD5")); v != "" {
		meta.MD5 = v
	}
	meta.URL = url
	if resp.StatusCode == http.StatusOK && resp.ContentLength > 0 {
		meta.ExpectedSize = resp.ContentLength
	} else if resp.StatusCode == http.StatusPartialContent && meta.ExpectedSize == 0 && resp.ContentLength > 0 {
		meta.ExpectedSize = offset + resp.ContentLength
	}
	if meta.ExpectedSize > limit {
		return tooLarge, false
	}
	if resp.ContentLength > limit {
		return tooLarge, false
	}

	var f *os.File
	if offset > 0 {
		f, err = fsx.OpenFile(dest, os.O_WRONLY|os.O_APPEND, 0o600)
	} else {
		f, err = fsx.Create(dest)
	}
	if err != nil {
		return err, false
	}
	defer func() { _ = f.Close() }()
	saveResume(dest, meta)

	var w io.Writer = f
	if progress != nil {
		total := meta.ExpectedSize
		w = &progressWriter{w: f, done: offset, total: total, fn: progress}
	}
	n, err := copyIdle(ctx, w, io.LimitReader(resp.Body, limit+1-offset), downloadIdle)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err(), false
		}
		return err, false
	}
	if offset+n > limit {
		return tooLarge, false
	}
	if err := f.Close(); err != nil {
		return err, false
	}
	st, err := os.Stat(dest)
	if err != nil {
		return err, false
	}
	if meta.ExpectedSize > 0 && st.Size() != meta.ExpectedSize {
		dropDownload(dest)
		return fmt.Errorf("the download size is %d bytes, not %d", st.Size(), meta.ExpectedSize), false
	}
	return verifyDownload(dest, meta), false
}

func parseContentRange(h string) (start, total int64, ok bool) {
	h = strings.TrimSpace(strings.TrimPrefix(h, "bytes"))
	rangePart, totalPart, found := strings.Cut(strings.TrimSpace(h), "/")
	if !found {
		return 0, 0, false
	}
	from, _, ok := strings.Cut(rangePart, "-")
	if !ok {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(from, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if totalPart != "*" {
		total, err = strconv.ParseInt(totalPart, 10, 64)
		if err != nil {
			return 0, 0, false
		}
	}
	return start, total, true
}

func hashFrom(h http.Header) string {
	if v := strings.TrimSpace(h.Get("X-Checksum-Sha256")); v != "" {
		return "sha256:" + strings.ToLower(strings.TrimPrefix(v, "sha256:"))
	}
	return ""
}

func verifyDownload(path string, meta resumeMeta) error {
	if meta.Hash == "" && meta.MD5 == "" {
		return nil
	}
	var wantMD5 []byte
	if meta.MD5 != "" {
		b, err := base64.StdEncoding.DecodeString(meta.MD5)
		if err != nil {
			dropDownload(path)
			return fmt.Errorf("Content-MD5: %w", err)
		}
		wantMD5 = b
	}
	f, err := fsx.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	sha := sha256.New()
	sum := md5.New() // #nosec G401 -- Content-MD5 is MD5 by RFC 1864
	if _, err := io.Copy(io.MultiWriter(sha, sum), f); err != nil {
		return err
	}
	if kind, want, _ := strings.Cut(meta.Hash, ":"); kind == "sha256" && !strings.EqualFold(want, hex.EncodeToString(sha.Sum(nil))) {
		dropDownload(path)
		return errors.New("the download did not match its checksum")
	}
	if meta.MD5 != "" && (len(wantMD5) != md5.Size || string(sum.Sum(nil)) != string(wantMD5)) {
		dropDownload(path)
		return errors.New("the download did not match its Content-MD5 checksum")
	}
	return nil
}

func copyIdle(ctx context.Context, dst io.Writer, src io.Reader, idle time.Duration) (int64, error) {
	if idle <= 0 {
		return io.Copy(dst, src)
	}
	return io.Copy(dst, &idleReader{done: ctx.Done(), cause: ctx.Err, r: src, idle: idle})
}

type idleReader struct {
	done  <-chan struct{}
	cause func() error
	r     io.Reader
	idle  time.Duration
	err   error
}

func (r *idleReader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	if err := r.cause(); err != nil {
		r.err = err
		return 0, err
	}
	buf := make([]byte, len(p))
	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1)
	go func() {
		n, err := r.r.Read(buf)
		ch <- result{n, err}
	}()
	t := time.NewTimer(r.idle)
	defer t.Stop()
	select {
	case <-r.done:
		err := r.cause()
		if err == nil {
			err = context.Canceled
		}
		r.err = err
		return 0, r.err
	case <-t.C:
		r.err = context.DeadlineExceeded
		return 0, r.err
	case got := <-ch:
		return copy(p, buf[:got.n]), got.err
	}
}

type progressWriter struct {
	w           io.Writer
	done, total int64
	fn          Progress
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.done += int64(n)
	p.fn(p.done, p.total)
	return n, err
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

type cacheEntry struct {
	Fetched  time.Time `json:"fetched"`
	Releases []Release `json:"releases"`
}

func (c *Client) cacheDir() (string, error) {
	if c.CacheDir != "" {
		return c.CacheDir, nil
	}
	d, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "cache"), nil
}

// Releases lists owner/repo's releases, newest first. An answer younger than an hour comes from the cache and a
// stale one stands in when the lookup fails, rate limit included.
func (c *Client) Releases(ctx context.Context, owner, repo string) ([]Release, error) {
	base := c.APIBase
	if base == "" {
		base = apiBase
	}
	dir, err := c.cacheDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, Key(owner, repo, "", "")+"-releases.json")
	var old cacheEntry
	if b, err := fsx.ReadFile(path); err == nil && json.Unmarshal(b, &old) != nil {
		old = cacheEntry{}
	}
	if !old.Fetched.IsZero() && c.now().Sub(old.Fetched) < cacheTTL {
		return old.Releases, nil
	}
	all, err := FetchReleases(ctx, c.HTTP, base+"/repos/"+owner+"/"+repo+"/releases?per_page=30")
	if err != nil {
		if !old.Fetched.IsZero() {
			return old.Releases, nil
		}
		return nil, err
	}
	// A cache that cannot be written only costs a refetch.
	if os.MkdirAll(dir, 0o700) == nil {
		_ = datadir.WriteJSON(path, cacheEntry{Fetched: c.now(), Releases: all})
	}
	return all, nil
}

// ErrNoRelease and ErrNoArchive say why Select found nothing to install.
var (
	ErrNoRelease = errors.New("GitHub lists no matching stable release")
	ErrNoArchive = errors.New("the release has no zip, rar or 7z asset")
)

func tagVersion(tag string) string { return strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V") }

var archiveTypes = map[string]bool{
	"application/zip": true, "application/x-zip-compressed": true, "application/x-rar-compressed": true,
	"application/vnd.rar": true, "application/x-7z-compressed": true, "application/octet-stream": true,
}

// IsArchive reports whether a is a zip, rar or 7z by name, and by content type where GitHub set one.
func IsArchive(a Asset) bool {
	switch strings.ToLower(filepath.Ext(a.Name)) {
	case ".zip", ".rar", ".7z":
		return a.ContentType == "" || archiveTypes[strings.ToLower(a.ContentType)]
	}
	return false
}

// Select picks the newest non-prerelease, non-draft release whose tag is version (any when version is empty)
// and returns it with its archive assets: one when the release ships one, several when the user must choose.
func Select(all []Release, version string) (Release, []Asset, error) {
	for _, r := range all {
		if r.Draft || r.Prerelease {
			continue
		}
		if version != "" {
			if cmp, ok := meta.CompareVersions(tagVersion(r.Tag), tagVersion(version)); !ok || cmp != 0 {
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

// Download saves asset to a temp file and returns its path; the caller removes it. The size cap is the archive
// extraction cap.
func (c *Client) Download(ctx context.Context, asset Asset, progress Progress) (string, error) {
	f, err := os.CreateTemp("", "mortar-github-*"+filepath.Ext(asset.Name))
	if err != nil {
		return "", err
	}
	path := f.Name()
	_ = f.Close()
	if err := Download(ctx, c.HTTP, asset.URL, path, archive.DefaultMaxTotalBytes, progress); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("download %s: %w", asset.Name, err)
	}
	return path, nil
}
