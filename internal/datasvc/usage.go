package datasvc

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// Progress is a size walk in progress.
type Progress struct {
	Measuring  bool   `json:"measuring"`
	Copying    bool   `json:"copying"`
	Bytes      int64  `json:"bytes"`
	TotalBytes int64  `json:"totalBytes"`
	Files      int    `json:"files"`
	TotalFiles int    `json:"totalFiles"`
	Path       string `json:"path"`
}

// ProfileSize is one profile's mods folder, not the shared store.
type ProfileSize struct {
	Game string `json:"game"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Usage is disk use of Mortar's data folder.
type Usage struct {
	Path     string        `json:"path"`
	Profiles []ProfileSize `json:"profiles"`
	Store    int64         `json:"store"`
	Cache    int64         `json:"cache"`
	Backups  int64         `json:"backups"`
	Trash    int64         `json:"trash"`
	Total    int64         `json:"total"`
}

type profileMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Measure walks root without following symlinks and reports each bucket's size.
func Measure(root string, report func(Progress)) (Usage, error) {
	u := Usage{Path: root, Profiles: []ProfileSize{}}
	mods := map[string]int64{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			return fs.SkipDir
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if info, infoErr := d.Info(); infoErr == nil {
			n := info.Size()
			u.Total += n
			rel := path
			if r, relErr := filepath.Rel(root, path); relErr == nil {
				rel = r
			}
			slash := filepath.ToSlash(rel)
			switch {
			case hasPrefix(slash, "store/"):
				u.Store += n
			case hasPrefix(slash, "cache/"):
				u.Cache += n
			case hasPrefix(slash, "backups/"):
				u.Backups += n
			case hasPrefix(slash, "trash/"):
				u.Trash += n
			default:
				if game, id, ok := profileMods(slash); ok {
					mods[game+"/"+id] += n
				}
			}
			if report != nil {
				report(Progress{Measuring: true, Bytes: u.Total, Path: slash})
			}
		}
		return nil
	})
	if err != nil {
		return Usage{}, err
	}
	profilesRoot := filepath.Join(root, "profiles")
	games, err := os.ReadDir(profilesRoot)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Usage{}, err
	}
	for _, g := range games {
		if !g.IsDir() || g.Type()&fs.ModeSymlink != 0 {
			continue
		}
		ids, err := os.ReadDir(filepath.Join(profilesRoot, g.Name()))
		if err != nil {
			continue
		}
		for _, d := range ids {
			if !d.IsDir() || d.Type()&fs.ModeSymlink != 0 {
				continue
			}
			name := d.Name()
			meta := readProfile(filepath.Join(profilesRoot, g.Name(), d.Name(), "profile.json"))
			if meta.Name != "" {
				name = meta.Name
			}
			key := g.Name() + "/" + d.Name()
			u.Profiles = append(u.Profiles, ProfileSize{
				Game: g.Name(),
				ID:   d.Name(),
				Name: name,
				Size: mods[key],
			})
		}
	}
	return u, nil
}

func hasPrefix(slash, prefix string) bool {
	return slash == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(slash, prefix)
}

func profileMods(slash string) (game, id string, ok bool) {
	rest, found := strings.CutPrefix(slash, "profiles/")
	if !found {
		return "", "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) < 4 || parts[2] != "mods" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func readProfile(path string) profileMeta {
	b, err := fsx.ReadFile(path)
	if err != nil {
		return profileMeta{}
	}
	var m profileMeta
	_ = json.Unmarshal(b, &m)
	return m
}
