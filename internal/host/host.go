// Package host downloads a mod file from a URL. A host knows one family of URLs: GitHub release assets, which can
// use the user's gh login, or any other HTTPS address.
package host

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"

	ga "github.com/Rethunk-Tech/go-archive"
	"github.com/Rethunk-Tech/go-httpkit/download"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

func fileName(u *url.URL) string {
	n := path.Base(u.Path)
	if n == "." || n == ".." || n == "/" || n == "" {
		return "download"
	}
	if clean := ga.Clean(n); clean != "" {
		return clean
	}
	return "download"
}

// Direct is a plain HTTPS GET.
type Direct struct {
	HTTP *http.Client
	// MaxBytes caps the body; zero means the archive extraction cap.
	MaxBytes int64
}

// Fetch downloads to dstDir. download.File keeps a partial and a sidecar for resume; Remove on error keeps Fetch's
// contract that a failed or oversize download leaves nothing behind.
func (d *Direct) Fetch(ctx context.Context, rawURL, dstDir string) (string, error) {
	u, ok := source.ParseHTTPSURL(rawURL)
	if !ok {
		return "", fmt.Errorf("not an https address")
	}
	limit := d.MaxBytes
	if limit <= 0 {
		limit = archive.DefaultMaxTotalBytes
	}
	hc := d.HTTP
	dst := filepath.Join(dstDir, fileName(u))
	if err := download.File(ctx, hc, rawURL, dst, download.Options{Limit: limit}); err != nil {
		download.Remove(dst)
		return "", err
	}
	// File leaves the resume sidecar next to a finished dest.
	_ = safefs.Remove(download.Sidecar(dst))
	return dst, nil
}
