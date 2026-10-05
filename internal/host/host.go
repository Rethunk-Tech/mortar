// Package host downloads a mod file from a URL. A host knows one family of URLs: GitHub release assets, which can
// use the user's gh login, or any other HTTPS address.
package host

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/github"
)

// Host fetches the files behind the URLs it matches.
type Host interface {
	ID() string
	Match(rawURL string) bool
	// Fetch saves the file into dstDir and returns its path.
	Fetch(ctx context.Context, rawURL, dstDir string) (string, error)
}

// For returns the host for rawURL: GitHub for its own URLs, else the direct host. It is nil for a URL no host takes.
func For(rawURL string) Host {
	for _, h := range []Host{&GitHub{}, &Direct{}} {
		if h.Match(rawURL) {
			return h
		}
	}
	return nil
}

// httpsOnly refuses a redirect off https, so a server cannot steer a download to a plain-HTTP or local address.
func httpsOnly(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	if req.URL.Scheme != "https" {
		return fmt.Errorf("refusing a redirect to %s://", req.URL.Scheme)
	}
	return nil
}

const maxRedirects = 10

func fileName(u *url.URL) string {
	n := path.Base(u.Path)
	if n == "." || n == ".." || n == "/" || n == "" {
		return "download"
	}
	return filepath.Base(n)
}

func parseHTTPS(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	return u, err == nil && u.Scheme == "https" && u.Host != ""
}

// Direct is a plain HTTPS GET.
type Direct struct {
	HTTP *http.Client
	// MaxBytes caps the body; zero means the archive extraction cap.
	MaxBytes int64
}

func (*Direct) ID() string { return "direct" }

func (*Direct) Match(rawURL string) bool { _, ok := parseHTTPS(rawURL); return ok }

// Fetch writes to a temp file beside the target and renames it into place, so a failed or oversize download leaves
// nothing behind.
func (d *Direct) Fetch(ctx context.Context, rawURL, dstDir string) (string, error) {
	u, ok := parseHTTPS(rawURL)
	if !ok {
		return "", fmt.Errorf("not an https address")
	}
	limit := d.MaxBytes
	if limit <= 0 {
		limit = archive.DefaultMaxTotalBytes
	}
	hc := d.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Minute}
	}
	if hc.CheckRedirect == nil {
		secured := *hc
		secured.CheckRedirect = httpsOnly
		hc = &secured
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach %s: %w", u.Host, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server answered %s", resp.Status)
	}
	if resp.ContentLength > limit {
		return "", fmt.Errorf("larger than %d MiB", limit>>20)
	}
	tmp, err := os.CreateTemp(dstDir, ".download-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	n, err := io.Copy(tmp, io.LimitReader(resp.Body, limit+1))
	if err == nil && n > limit {
		err = fmt.Errorf("larger than %d MiB", limit>>20)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}
	dst := filepath.Join(dstDir, fileName(u))
	if err := fsx.Rename(tmp.Name(), dst); err != nil {
		return "", err
	}
	return dst, nil
}

// GitHub fetches release assets with resume, using the gh login's token when there is one.
type GitHub struct{ HTTP *http.Client }

func (*GitHub) ID() string { return "github" }

func (*GitHub) Match(rawURL string) bool {
	u, ok := parseHTTPS(rawURL)
	return ok && u.Hostname() == "github.com" && strings.Contains(u.Path, "/releases/download/")
}

func (g *GitHub) Fetch(ctx context.Context, rawURL, dstDir string) (string, error) {
	u, ok := parseHTTPS(rawURL)
	if !ok {
		return "", fmt.Errorf("not an https address")
	}
	dst := filepath.Join(dstDir, fileName(u))
	if err := github.Download(ctx, g.HTTP, rawURL, dst, archive.DefaultMaxTotalBytes, nil); err != nil {
		return "", err
	}
	return dst, nil
}
