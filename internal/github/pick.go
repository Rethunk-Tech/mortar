package github

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
)

var (
	sourceAsset   = regexp.MustCompile(`(?i)(^|[^a-z])(src|sources?|source[ ._-]?code)([^a-z]|$)`)
	checksumAsset = regexp.MustCompile(`(?i)(checksums?|sha(1|256|512)(sums?)?|md5(sums?)?)([^a-z]|$)`)
	versionRun    = regexp.MustCompile(`v?\d+(?:[._]\d+)+|v\d+`)
	nonWord       = regexp.MustCompile(`[^a-z]+`)
)

// platformWords are the words an asset name uses for each GOOS.
var platformWords = map[string][]string{
	"windows": {"windows", "win", "win32", "win64"},
	"linux":   {"linux"},
	"darwin":  {"macos", "mac", "osx", "darwin"},
}

func words(name string) []string {
	return strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return (r < 'a' || r > 'z') && (r < '0' || r > '9') })
}

// forOtherPlatform reports an asset that names a platform and none of them is goos.
func forOtherPlatform(name, goos string) bool {
	w := words(strings.TrimSuffix(strings.ToLower(name), strings.ToLower(path.Ext(name))))
	named := false
	for platform, list := range platformWords {
		if slices.ContainsFunc(list, func(p string) bool { return slices.Contains(w, p) }) {
			if platform == goos {
				return false
			}
			named = true
		}
	}
	return named
}

// Installable narrows a release's archive assets to the ones a player of goos would install: source archives,
// checksum files and builds for other platforms drop out. A filter that would leave nothing is skipped, since an
// author who only ships, say, a Windows-named zip still means it for every platform.
func Installable(assets []Asset, goos string) []Asset {
	out := slices.Clone(assets)
	for _, drop := range []func(Asset) bool{
		func(a Asset) bool { return sourceAsset.MatchString(a.Name) || checksumAsset.MatchString(a.Name) },
		func(a Asset) bool { return forOtherPlatform(a.Name, goos) },
	} {
		if kept := slices.DeleteFunc(slices.Clone(out), drop); len(kept) > 0 {
			out = kept
		}
	}
	return out
}

// Shape is an asset's name with its extension and every version number removed, so the same download keeps its
// shape from one release to the next ("MyMod-1.2.0-linux.zip" and "MyMod-v1.3-linux.zip" are both "mymod linux").
func Shape(name string) string {
	name = strings.TrimSuffix(strings.ToLower(name), strings.ToLower(path.Ext(name)))
	return strings.TrimSpace(nonWord.ReplaceAllString(versionRun.ReplaceAllString(name, " "), " "))
}

// peekChunk is how much one ranged read fetches; a zip's central directory usually fits in the first one.
const peekChunk = 64 << 10

// rangeReader reads a remote file of size bytes through get, a ranged request, keeping the last chunk it fetched.
type rangeReader struct {
	get      func(start, end int64) ([]byte, error)
	size     int64
	at       int64
	buf      []byte
	requests int
}

func (r *rangeReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= r.size {
		return 0, io.EOF
	}
	if off < r.at || off+int64(len(p)) > r.at+int64(len(r.buf)) {
		r.requests++
		if r.requests > 8 {
			return 0, errors.New("the archive's file list is too large to read remotely")
		}
		// A zip's directory is read from its end backwards, so a chunk ending at the file's end serves most reads.
		n := max(int64(len(p)), peekChunk)
		start := min(off, max(0, r.size-n))
		b, err := r.get(start, min(r.size, start+n))
		if err != nil {
			return 0, err
		}
		r.at, r.buf = start, b
	}
	if off < r.at || off-r.at > int64(len(r.buf)) {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, r.buf[off-r.at:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// ZipNames lists the entry names of a zip asset without downloading it, reading only its central directory through
// a few ranged requests. Other archive kinds answer nothing.
func ZipNames(ctx context.Context, hc *http.Client, a Asset) ([]string, error) {
	if !strings.EqualFold(path.Ext(a.Name), ".zip") || a.Size <= 0 {
		return nil, errors.New("not a zip")
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()
	get := func(start, end int64) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end-1))
		resp, err := hc.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusPartialContent {
			return nil, fmt.Errorf("ranged read answered %s", resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, end-start))
	}
	zr, err := zip.NewReader(&rangeReader{get: get, size: a.Size}, a.Size)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(zr.File))
	for i, f := range zr.File {
		names[i] = f.Name
	}
	return names, nil
}
