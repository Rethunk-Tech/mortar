package datasvc

import (
	"encoding/json"
	"errors"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
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

// GameUsage is one game's profiles, store items, save backups, and cache entries.
type GameUsage struct {
	Game string `json:"game"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Usage is disk use of Mortar's data folder.
type Usage struct {
	Path             string        `json:"path"`
	Profiles         []ProfileSize `json:"profiles"`
	Games            []GameUsage   `json:"games"`
	Store            int64         `json:"store"`
	Cache            int64         `json:"cache"`
	Backups          int64         `json:"backups"`
	Trash            int64         `json:"trash"`
	Total            int64         `json:"total"`
	SharedSaved      int64         `json:"sharedSaved"`
	SharedSavedKnown bool          `json:"sharedSavedKnown"`
}

// CacheInfo is Mortar's cache folder.
type CacheInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// EntrySize is one store key's size on disk.
type EntrySize struct {
	Game string `json:"game"`
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

type profileMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Measure walks root without following symlinks and reports each bucket's exclusive size.
func Measure(root string, report func(Progress)) (Usage, error) {
	u := Usage{Path: root, Profiles: []ProfileSize{}, Games: []GameUsage{}}
	mods := map[string]int64{}
	direct := map[string]int64{}
	backupGame := map[string]int64{}
	cacheSeg := map[[2]string]int64{}
	share := newShareAcc()
	root = filepath.Clean(root)
	account := func(path string, info os.FileInfo) {
		share.addFollowed(info)
		n, known := fileExclusive(path, info, share)
		if known {
			share.allocated += n
		}
		u.Total += n
		rel := path
		if r, relErr := filepath.Rel(root, path); relErr == nil {
			rel = r
		}
		slash := filepath.ToSlash(rel)
		switch {
		case hasPrefix(slash, "store/"):
			u.Store += n
			if game := nestedGame(slash, "store/"); game != "" {
				direct[game] += n
			}
		case hasPrefix(slash, "cache/"):
			u.Cache += n
			cacheSeg[cacheSegs(slash)] += n
		case hasPrefix(slash, "backups/"):
			u.Backups += n
			if game := nestedGame(slash, "backups/"); game != "" {
				backupGame[game] += n
			}
		case hasPrefix(slash, "trash/"):
			u.Trash += n
		default:
			if game := nestedGame(slash, "profiles/"); game != "" {
				direct[game] += n
			}
			if game, id, ok := profileMods(slash); ok {
				mods[game+"/"+id] += n
			}
		}
		if report != nil {
			report(Progress{Measuring: true, Bytes: u.Total, Path: slash})
		}
	}
	walk := func(path string, skipNamed bool) error {
		st, err := os.Lstat(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if !st.IsDir() {
			return nil
		}
		return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if p == path {
					return err
				}
				return fs.SkipDir
			}
			if skipNamed && p != root {
				rel, relErr := filepath.Rel(root, p)
				if relErr == nil && filepath.Dir(rel) == "." {
					switch d.Name() {
					case "store", "profiles", "backups", "cache", "trash":
						return fs.SkipDir
					}
				}
			}
			if d.Type()&fs.ModeSymlink != 0 {
				if d.IsDir() {
					return fs.SkipDir
				}
				if info, infoErr := os.Stat(filepath.Clean(p)); infoErr == nil && info.Mode().IsRegular() {
					if resolved, resErr := filepath.EvalSymlinks(p); resErr == nil && datadir.UnderRoot(root, resolved) {
						share.addFollowed(info)
					}
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			if info, infoErr := d.Info(); infoErr == nil {
				account(p, info)
			}
			return nil
		})
	}
	for _, name := range []string{"store", "profiles", "backups", "cache", "trash"} {
		if err := walk(filepath.Join(root, name), false); err != nil {
			return Usage{}, err
		}
	}
	err := walk(root, true)
	if err != nil {
		return Usage{}, err
	}
	u.SharedSaved, u.SharedSavedKnown = share.saved()
	dirs, err := datadir.ProfileDirs(filepath.Join(root, "profiles"))
	if err != nil {
		return Usage{}, err
	}
	for _, d := range dirs {
		name := d.ID
		if meta := readProfile(filepath.Join(d.Dir, "profile.json")); meta.Name != "" {
			name = meta.Name
		}
		u.Profiles = append(u.Profiles, ProfileSize{Game: d.Game, ID: d.ID, Name: name, Size: mods[d.Game+"/"+d.ID]})
	}
	known := map[string]struct{}{}
	sizes := map[string]int64{}
	for id, n := range direct {
		known[id] = struct{}{}
		sizes[id] = n
	}
	for id, n := range backupGame {
		known[id] = struct{}{}
		sizes[id] += n
	}
	for segs, n := range cacheSeg {
		switch {
		case segs[0] != "":
			if _, ok := known[segs[0]]; ok {
				sizes[segs[0]] += n
				continue
			}
			fallthrough
		default:
			if _, ok := known[segs[1]]; ok {
				sizes[segs[1]] += n
			}
		}
	}
	for id, n := range sizes {
		name := id
		if g := game.Find(id); g != nil {
			name = g.Name()
		}
		u.Games = append(u.Games, GameUsage{Game: id, Name: name, Size: n})
	}
	return u, nil
}

func nestedGame(slash, prefix string) string {
	rest, ok := strings.CutPrefix(slash, prefix)
	if !ok {
		return ""
	}
	game, more, found := strings.Cut(rest, "/")
	// Dot-prefixed folders are Mortar's own indexes (store/.manifests), not games.
	if !found || game == "" || more == "" || strings.HasPrefix(game, ".") {
		return ""
	}
	return game
}

func cacheSegs(slash string) [2]string {
	rest, ok := strings.CutPrefix(slash, "cache/")
	if !ok {
		return [2]string{}
	}
	first, rest, found := strings.Cut(rest, "/")
	if !found {
		return [2]string{first, ""}
	}
	second, _, _ := strings.Cut(rest, "/")
	return [2]string{first, second}
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

// ModUse is one store item's disk use.
type ModUse struct {
	Game        string `json:"game"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	Profiles    int    `json:"profiles"`
	ProfileSize int64  `json:"profileSize"`
	LastUsed    string `json:"lastUsed"`
}

// ModUsage is store items with sizes, who uses them, and a store-size total.
type ModUsage struct {
	Total int64    `json:"total"`
	Items []ModUse `json:"items"`
}

type profileEntries struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Entries []struct {
		Key            string   `json:"key"`
		ExtraStoreKeys []string `json:"extraStoreKeys"`
		Source         struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"source"`
	} `json:"entries"`
}

// MeasureMods reports each store folder's size, live-profile use, and copy sizes under profiles/.
func MeasureMods(root string) (ModUsage, error) {
	out := ModUsage{Items: []ModUse{}}
	names, uses, copies := profileUse(root)
	entries, err := store.OpenAt(filepath.Join(root, "store")).Entries()
	if err != nil {
		return ModUsage{}, err
	}
	for _, e := range entries {
		id := e.Game + "/" + e.Key
		n := dirSize(e.Dir)
		name := names[id]
		if name == "" {
			name = e.Key
		}
		lastUsed := ""
		if !e.LastUsed.IsZero() {
			lastUsed = e.LastUsed.UTC().Format(time.RFC3339)
		}
		out.Items = append(out.Items, ModUse{
			Game: e.Game, Key: e.Key, Name: name, Size: n,
			Profiles: uses[id], ProfileSize: copies[id], LastUsed: lastUsed,
		})
		out.Total += n
	}
	return out, nil
}

func profileUse(root string) (names map[string]string, uses map[string]int, copies map[string]int64) {
	names = map[string]string{}
	uses = map[string]int{}
	copies = map[string]int64{}
	dirs, _ := datadir.ProfileDirs(filepath.Join(root, "profiles"))
	for _, d := range dirs {
		p := readProfileEntries(filepath.Join(d.Dir, "profile.json"))
		mods := filepath.Join(d.Dir, "mods")
		seen := map[string]bool{}
		for _, e := range p.Entries {
			keys := append([]string{e.Key}, e.ExtraStoreKeys...)
			for _, key := range keys {
				if key == "" {
					continue
				}
				id := d.Game + "/" + key
				if e.Source.Name != "" && names[id] == "" {
					if e.Source.Version == "" {
						names[id] = e.Source.Name
					} else {
						names[id] = e.Source.Name + " " + e.Source.Version
					}
				}
				if !seen[id] {
					uses[id]++
					seen[id] = true
				}
				if key == e.Key {
					copies[id] += dirSize(filepath.Join(mods, key)) + dirSize(filepath.Join(mods, "."+key))
				} else {
					copies[id] += dirSize(filepath.Join(mods, e.Key, key))
				}
			}
		}
	}
	return names, uses, copies
}

func readProfileEntries(path string) profileEntries {
	b, err := fsx.ReadFile(path)
	if err != nil {
		return profileEntries{}
	}
	var p profileEntries
	_ = json.Unmarshal(b, &p)
	return p
}

func clearCache(dir string) error {
	ents, err := os.ReadDir(filepath.Clean(dir))
	if errors.Is(err, fs.ErrNotExist) {
		return os.MkdirAll(filepath.Clean(dir), 0o700)
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range ents {
		errs = append(errs, fsx.RemoveAll(filepath.Clean(filepath.Join(dir, e.Name()))))
	}
	return errors.Join(errs...)
}

func usageFingerprint(root string) string {
	var n int64
	for _, rel := range []string{filepath.Join("store", "index.json"), "profiles", "store"} {
		info, err := os.Stat(filepath.Clean(filepath.Join(root, rel)))
		if err != nil {
			continue
		}
		n += info.ModTime().UnixNano()
	}
	return strconv.FormatInt(n, 10)
}
