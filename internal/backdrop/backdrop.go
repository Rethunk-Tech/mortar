// Package backdrop serves the wallpaper drawn behind the window. The image mode serves the user's image,
// else Fedora's system wallpaper, else the copy bundled with Mortar; the desktop mode tries the user's
// desktop wallpaper first; the solid mode serves nothing.
package backdrop

import (
	"bytes"
	_ "embed" // bundled wallpaper
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// Path is the URL the frontend loads the backdrop from.
const Path = "/backdrop"

// SystemDefault is Fedora 44's default wallpaper, used in place of the bundled copy when present.
const SystemDefault = "/usr/share/backgrounds/f44/default/f44-01-night.jxl"

//go:embed f44-01-night.jpg
var bundled []byte

// contentType returns the image type for path's extension, empty when Mortar will not serve it.
// JPEG XL is served on Linux only: WebKitGTK decodes it, WebView2 does not.
func contentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".jxl":
		if runtime.GOOS == "linux" {
			return "image/jxl"
		}
	}
	return ""
}

func open(path string) (*os.File, string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, "", fmt.Errorf("%q is not a clean absolute path", path)
	}
	typ := contentType(path)
	if typ == "" {
		return nil, "", fmt.Errorf("%q is not a PNG, JPEG, WebP or JPEG XL image", path)
	}
	f, err := fsx.Open(path)
	if err != nil {
		return nil, "", err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = f.Close()
		return nil, "", errors.Join(err, fmt.Errorf("%q is not a regular file", path))
	}
	return f, typ, nil
}

// Check reports why path cannot be served as the backdrop.
func Check(path string) error {
	f, _, err := open(path)
	if err != nil {
		return err
	}
	return f.Close()
}

// candidates lists the files to try, in order, for the current settings.
func candidates(cur settings.Settings, system string, desktop func() string) []string {
	if cur.Background == settings.BackgroundDesktop {
		return []string{desktop(), cur.BackgroundImage, system}
	}
	return []string{cur.BackgroundImage, system}
}

// Middleware serves GET /backdrop for the current settings: the first servable candidate, else the bundled
// copy. Solid mode has no backdrop.
func Middleware(current func() settings.Settings, system string, desktop func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != Path {
				next.ServeHTTP(w, r)
				return
			}
			cur := current()
			// The page names the mode it just chose, which can reach here before the settings write does. The image
			// path is still read from settings only, never from the URL.
			if m := r.URL.Query().Get("mode"); slices.Contains([]string{settings.BackgroundImage, settings.BackgroundDesktop, settings.BackgroundSolid}, m) {
				cur.Background = m
			}
			if r.Method != http.MethodGet || cur.Background == settings.BackgroundSolid {
				http.NotFound(w, r)
				return
			}
			// The file behind the one URL changes with the mode and the desktop, and an older file's date would win a
			// Last-Modified revalidation, so the local file is served uncached and undated.
			w.Header().Set("Cache-Control", "no-store")
			for _, path := range candidates(cur, system, desktop) {
				if path == "" {
					continue
				}
				f, typ, err := open(path)
				if err != nil {
					continue
				}
				defer func() { _ = f.Close() }()
				w.Header().Set("Content-Type", typ)
				http.ServeContent(w, r, "", time.Time{}, f)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(bundled))
		})
	}
}
