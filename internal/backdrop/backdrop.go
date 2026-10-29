// Package backdrop serves the wallpaper drawn behind the window. The image mode serves the user's image,
// else Fedora's system wallpaper, else the copy bundled with Mortar; the desktop mode tries the user's
// desktop wallpaper first; the solid mode serves nothing.
package backdrop

import (
	"bytes"
	"crypto/sha256"
	_ "embed" // bundled wallpaper
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/modpic"
	"github.com/Rethunk-Tech/mortar/internal/settings"
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
	case ".gif":
		return "image/gif"
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
		return nil, "", fmt.Errorf("%q is not a PNG, JPEG, GIF, WebP or JPEG XL image", path)
	}
	f, err := fsx.OpenRegular(path)
	if errors.Is(err, fsx.ErrNotRegular) {
		err = fmt.Errorf("%q is not a regular file", path)
	}
	if err != nil {
		return nil, "", err
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
// copy. Solid mode has no backdrop. Shrunk copies are kept in cacheDir (none when empty).
func Middleware(current func() settings.Settings, system string, desktop func() string, cacheDir string) func(http.Handler) http.Handler {
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
			// The file behind the one URL changes with the mode and the desktop, and an older file's date would win an
			// If-Modified-Since revalidation, so every answer carries an ETag naming the file, width and date, which
			// the browser checks first.
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Vary", "Accept")
			jxl := !negotiates || strings.Contains(r.Header.Get("Accept"), "image/jxl")
			for _, path := range candidates(cur, system, desktop) {
				if path == "" {
					continue
				}
				f, typ, err := open(path)
				if err != nil {
					continue
				}
				// JPEG XL cannot be shrunk here, so a page asking for its width gets the bundled JPEG of the same default
				// image, shrunk, rather than decoding the system copy's 4096 pixels square.
				if typ == "image/jxl" && (!jxl || (path == system && r.URL.Query().Get("w") != "")) {
					_ = f.Close()
					continue
				}
				defer func() { _ = f.Close() }()
				var mod time.Time
				id := path
				if info, err := f.Stat(); err == nil {
					mod = info.ModTime()
					id += "\x00" + strconv.FormatInt(info.Size(), 10) + "\x00" + strconv.FormatInt(mod.UnixNano(), 10)
				}
				serve(w, r, f, typ, id, mod, cacheDir)
				return
			}
			serve(w, r, bytes.NewReader(bundled), "image/jpeg", "bundled", time.Time{}, cacheDir)
		})
	}
}

// Widths the page may ask for: below the first the window is too small to matter, above the last no screen needs more.
const (
	minWidthPx = 640
	maxWidthPx = 2560
)

// serve writes the image as it is, or shrunk to the width the page asked for with ?w=, so the browser does not decode
// a 4K wallpaper to fill a smaller window. JPEG XL cannot be decoded here and an animation would lose its frames, so
// both are served whole; the page freezes an animation itself when motion is reduced. id names the source's version;
// a shrunk copy is kept in cacheDir under it, so a start after the first does not shrink again.
func serve(w http.ResponseWriter, r *http.Request, src io.ReadSeeker, typ, id string, mod time.Time, cacheDir string) {
	query := r.URL.Query().Get("w")
	sum := sha256.Sum256([]byte(id + "\x00" + typ + "\x00" + query))
	name := hex.EncodeToString(sum[:])
	w.Header().Set("ETag", `"`+name[:32]+`"`)
	if px, err := strconv.Atoi(query); err == nil && px >= minWidthPx && px <= maxWidthPx && typ != "image/jxl" {
		cached := filepath.Join(cacheDir, name)
		if cacheDir != "" {
			if b, err := fsx.ReadFile(cached); err == nil {
				w.Header().Set("Content-Type", http.DetectContentType(b))
				http.ServeContent(w, r, "", mod, bytes.NewReader(b))
				return
			}
		}
		if b, err := modpic.ReadCapped(src, maxSourceBytes); err == nil && !animated(b, typ) {
			if out, outTyp, err := modpic.Shrink(b, typ, px); err == nil {
				if cacheDir != "" && len(out) < len(b) {
					keep(cacheDir, cached, out)
				}
				w.Header().Set("Content-Type", outTyp)
				http.ServeContent(w, r, "", mod, bytes.NewReader(out))
				return
			}
		}
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			http.Error(w, "the backdrop could not be read", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", typ)
	http.ServeContent(w, r, "", mod, src)
}

// keptCopies bounds the shrunk copies on disk: a few widths of the current image, and older images' leftovers.
const keptCopies = 8

// keep writes one shrunk copy and drops the oldest beyond keptCopies.
func keep(dir, path string, b []byte) {
	if os.MkdirAll(dir, 0o700) != nil || datadir.WriteFileAtomic(path, b, 0o600) != nil {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) <= keptCopies {
		return
	}
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		ai, aerr := a.Info()
		bi, berr := b.Info()
		if aerr != nil || berr != nil {
			return 0
		}
		return ai.ModTime().Compare(bi.ModTime())
	})
	for _, e := range entries[:len(entries)-keptCopies] {
		_ = fsx.Remove(filepath.Join(dir, e.Name()))
	}
}

const maxSourceBytes = 64 << 20

// animated reports an image that may hold frames: any GIF, a WebP whose VP8X header sets the animation flag, and an
// APNG, whose acTL chunk comes before the first IDAT.
func animated(b []byte, typ string) bool {
	switch typ {
	case "image/gif":
		return true
	case "image/webp":
		return len(b) > 20 && string(b[12:16]) == "VP8X" && b[20]&0x02 != 0
	case "image/png":
		head, _, _ := bytes.Cut(b, []byte("IDAT"))
		return bytes.Contains(head, []byte("acTL"))
	}
	return false
}
