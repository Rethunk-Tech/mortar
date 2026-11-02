package installer

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader/thunderpack"
)

var _ = Register(zipped{})

// DriverZipped is the id of the driver that places a mod as one zip file in the mods target.
const DriverZipped = "zipped"

// zipped lays out the mods of a loader that reads only .zip files from its mods folder (GodotModLoader): the
// archive, whatever site it came from, is one file named for its store item.
type zipped struct{}

func (zipped) ID() string { return DriverZipped }

func (zipped) Detect(_ Archive, g Game) bool { return thunderpack.Zipped(g.Loaders) }

func (zipped) Layout(a Archive, g Game, _ Choices) (Layout, error) {
	l := Layout{Files: []File{{Src: ".", Target: TargetMods, Rel: a.Key + ".zip", Zip: true}}}
	return l, validate(l, g)
}

// ZipDir writes the files below src as a zip at dst. Names are sorted and stamped with no time, so the same tree
// always gives the same bytes and a placed copy can be compared by hash.
func ZipDir(src, dst string) error {
	var names []string
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err == nil && !skip(filepath.ToSlash(rel)) {
			names = append(names, filepath.ToSlash(rel))
		}
		return err
	})
	if err != nil {
		return err
	}
	slices.Sort(names)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			return err
		}
		b, err := safefs.ReadFile(filepath.Join(src, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	return safefs.WriteFile(dst, buf.Bytes(), 0o600)
}
