// Package pack reads other managers' mod lists (profile codes, profile folders, modpack packages) into a Draft.
// Reading writes nothing: turning a Draft into a profile belongs to the installer.
package pack

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Input is what a user handed Mortar: a file or folder path, a link, or pasted text. A format reads the field it
// understands.
type Input struct {
	Path, URL, Text string
}

// Ref names one package in a source's own terms: a Thunderstore "Namespace-Name", a Nexus "modId[/fileId]", a
// GitHub "owner/repo@tag/asset".
type Ref struct {
	Source   string
	Native   string
	Version  string
	Disabled bool
}

// LoaderPin is a mod loader and the version a pack was made with.
type LoaderPin struct {
	ID, Version string
}

// File is a file a pack carries, at a path relative to the profile.
type File struct {
	Path string
	Data []byte
}

// Draft is a pack's contents, before anything is installed.
type Draft struct {
	Name, Game, GameVersion string
	Loaders                 []LoaderPin
	Packages                []Ref
	Loose                   []File
	Configs                 []File
}

// Format is one pack format.
type Format interface {
	ID() string
	Detect(in Input) bool
	Parse(ctx context.Context, in Input) (Draft, error)
}

// ErrUnknown means no format recognised the input.
var ErrUnknown = errors.New("not a known mod pack")

// Read parses in with the first format that detects it.
func Read(ctx context.Context, in Input, formats ...Format) (Draft, error) {
	for _, f := range formats {
		if f.Detect(in) {
			return f.Parse(ctx, in)
		}
	}
	return Draft{}, ErrUnknown
}

const (
	maxFile    = 16 << 20
	maxEntries = 5000
	// maxTotal bounds a whole pack's unpacked size, so many small-looking entries cannot add up to a zip bomb.
	maxTotal = 256 << 20
	// maxInput bounds a pack read from disk or pasted.
	maxInput = 128 << 20
)

// nativeID is a Thunderstore package name, Namespace-Name, whose parts are the site's own word characters.
var nativeID = regexp.MustCompile(`^\w+-\w+$`)

// versionNumber is a major.minor.patch version.
var versionNumber = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// readZip returns the zip's files by cleaned slash path; a path that leaves the archive is refused.
func readZip(data []byte) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	if len(zr.File) > maxEntries {
		return nil, fmt.Errorf("archive holds %d entries", len(zr.File))
	}
	out := map[string][]byte{}
	total := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := path.Clean(strings.ReplaceAll(f.Name, `\`, "/"))
		if name == "." || path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, ":") {
			return nil, fmt.Errorf("archive path %q leaves the archive", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, maxFile+1))
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		if len(b) > maxFile {
			return nil, fmt.Errorf("archive file %q is too large", f.Name)
		}
		if total += len(b); total > maxTotal {
			return nil, errors.New("archive unpacks to too much data")
		}
		out[name] = b
	}
	return out, nil
}

// filesUnder returns the files below prefix ("config/") as File values, ordered by path.
func filesUnder(files map[string][]byte, prefix string) []File {
	var out []File
	for name, b := range files {
		if strings.HasPrefix(name, prefix) {
			out = append(out, File{Path: name, Data: b})
		}
	}
	slices.SortFunc(out, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	return out
}
