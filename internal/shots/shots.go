// Package shots lists a game's screenshots: the pictures in the folder the catalog's screenshots path names, and those
// Steam keeps for the game.
package shots

import (
	"cmp"
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	ga "github.com/Rethunk-Tech/go-archive"
	"github.com/Rethunk-Tech/go-safefs"
)

// Screenshot sources.
const (
	SourceGame  = "game"
	SourceSteam = "steam"
)

// Shot is one screenshot, its file Name in the folder Source names; At is when it was written, in Unix milliseconds.
type Shot struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	At     int64  `json:"at"`
	Size   int64  `json:"size"`
}

// Gallery is a game's screenshots, newest first, and the sources whose folder exists.
type Gallery struct {
	Shots   []Shot   `json:"shots"`
	Sources []string `json:"sources"`
}

var pictureExts = []string{".png", ".jpg", ".jpeg", ".webp", ".bmp"}

// IsPicture reports whether name is a bare file name in a picture format the app's webview shows.
func IsPicture(name string) bool {
	return ga.WindowsSegment(name) && !strings.HasPrefix(name, ".") &&
		slices.Contains(pictureExts, strings.ToLower(filepath.Ext(name)))
}

// List reads the pictures directly in each source's folder; a missing folder holds none.
func List(dirs map[string]string) (Gallery, error) {
	out := Gallery{Shots: []Shot{}, Sources: []string{}}
	var errs []error
	for _, source := range slices.Sorted(maps.Keys(dirs)) {
		ents, found, err := safefs.ReadDirIfExists(dirs[source])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !found {
			continue
		}
		out.Sources = append(out.Sources, source)
		for _, e := range ents {
			if !e.Type().IsRegular() || !IsPicture(e.Name()) {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			out.Shots = append(out.Shots, Shot{Source: source, Name: e.Name(), At: info.ModTime().UnixMilli(), Size: info.Size()})
		}
	}
	slices.SortFunc(out.Shots, func(a, b Shot) int { return cmp.Or(cmp.Compare(b.At, a.At), strings.Compare(a.Name, b.Name)) })
	return out, errors.Join(errs...)
}
