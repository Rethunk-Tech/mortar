package datadir

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// ProfileDir is one <profiles>/<game>/<id> folder.
type ProfileDir struct {
	Game, ID, Dir string
}

// ProfileDirs lists the profile folders under profilesRoot, skipping files and symlinks at both levels. A missing
// root is an empty list.
func ProfileDirs(profilesRoot string) ([]ProfileDir, error) {
	games, err := os.ReadDir(filepath.Clean(profilesRoot))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	var out []ProfileDir
	for _, g := range games {
		if !plainDir(g) {
			continue
		}
		ids, err := os.ReadDir(filepath.Join(profilesRoot, g.Name()))
		if err != nil {
			continue
		}
		for _, d := range ids {
			if plainDir(d) {
				out = append(out, ProfileDir{Game: g.Name(), ID: d.Name(), Dir: filepath.Join(profilesRoot, g.Name(), d.Name())})
			}
		}
	}
	return out, nil
}

func plainDir(e fs.DirEntry) bool { return e.IsDir() && e.Type()&fs.ModeSymlink == 0 }
