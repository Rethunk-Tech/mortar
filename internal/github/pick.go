package github

import (
	"archive/zip"
	"context"
	"errors"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit/download"
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
	zr, err := zip.NewReader(download.NewRangeReader(ctx, hc, a.URL, a.Size), a.Size)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(zr.File))
	for i, f := range zr.File {
		names[i] = f.Name
	}
	return names, nil
}
