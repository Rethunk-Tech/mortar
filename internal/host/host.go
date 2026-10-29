// Package host downloads a mod file from a URL. A host knows one family of URLs: GitHub release assets, which can
// use the user's gh login, or any other HTTPS address.
package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/bytefmt"
	"github.com/Rethunk-Tech/mortar/internal/winname"
)

func fileName(u *url.URL) string {
	n := path.Base(u.Path)
	if n == "." || n == ".." || n == "/" || n == "" {
		return "download"
	}
	if clean := winname.Clean(n); clean != "" {
		return clean
	}
	return "download"
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
	hc = httpkit.WithRedirect(hc, httpkit.NoDowngrade)
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
		return "", fmt.Errorf("larger than %s", bytefmt.Format(limit))
	}
	dst := filepath.Join(dstDir, fileName(u))
	err = safefs.WriteStreamAtomic(dst, 0o600, func(w io.Writer) error {
		_, err := io.Copy(w, safefs.CapReader(resp.Body, limit))
		if errors.Is(err, safefs.ErrTooLarge) {
			err = fmt.Errorf("larger than %s", bytefmt.Format(limit))
		}
		return err
	})
	if err != nil {
		return "", err
	}
	return dst, nil
}
