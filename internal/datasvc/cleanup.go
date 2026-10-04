package datasvc

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

const (
	detailsTTL    = 24 * time.Hour
	categoriesTTL = 30 * 24 * time.Hour
	datasetTTL    = 30 * 24 * time.Hour
	releasesTTL   = time.Hour
	updatesTTL    = time.Hour
)

// Item is one path Clean up would remove.
type Item struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Rel   string `json:"rel"`
	Size  int64  `json:"size"`
}

// Preview is the unused set with a total size.
type Preview struct {
	Items []Item `json:"items"`
	Total int64  `json:"total"`
}

type fetchedFile struct {
	Fetched time.Time `json:"fetched"`
}

// Select lists store items no referenced key names, cache files past their expiry, and leftover temp folders.
func Select(root string, items *store.Store, referenced map[string][]string, now time.Time, nameOf func(string, string) string) (Preview, error) {
	out := Preview{Items: []Item{}}
	refs, err := items.Unreferenced(referenced)
	if err != nil {
		return Preview{}, err
	}
	for _, r := range refs {
		rel := filepath.ToSlash(filepath.Join("store", r.Game, r.Key))
		n := dirSize(filepath.Join(root, filepath.FromSlash(rel)))
		label := r.Game + "/" + r.Key
		if nameOf != nil {
			if named := nameOf(r.Game, r.Key); named != "" {
				label = named
			}
		}
		out.Items = append(out.Items, Item{Kind: "store", Label: label, Rel: rel, Size: n})
		out.Total += n
	}
	cacheRoot := filepath.Join(root, "cache")
	err = filepath.WalkDir(filepath.Clean(cacheRoot), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fs.SkipDir
			}
			return err
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
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		slash := filepath.ToSlash(rel)
		ttl, ok := cacheTTL(strings.TrimPrefix(slash, "cache/"))
		if !ok || fileAge(path, now) <= ttl {
			return nil
		}
		if info, infoErr := d.Info(); infoErr == nil {
			out.Items = append(out.Items, Item{Kind: "cache", Label: slash, Rel: slash, Size: info.Size()})
			out.Total += info.Size()
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Preview{}, err
	}
	err = filepath.WalkDir(filepath.Clean(root), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		slash := filepath.ToSlash(rel)
		name := d.Name()
		if d.Type().IsRegular() && tempName(name) {
			if info, infoErr := d.Info(); infoErr == nil {
				out.Items = append(out.Items, Item{Kind: "temp", Label: slash, Rel: slash, Size: info.Size()})
				out.Total += info.Size()
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if !tempName(name) {
			return nil
		}
		empty, emptyErr := dirEmpty(path)
		if emptyErr != nil {
			return emptyErr
		}
		if !empty && !strings.HasPrefix(name, ".tmp-") && !strings.HasPrefix(name, ".tmp_") {
			return nil
		}
		n := dirSize(path)
		out.Items = append(out.Items, Item{Kind: "temp", Label: slash, Rel: slash, Size: n})
		out.Total += n
		return fs.SkipDir
	})
	if err != nil {
		return Preview{}, err
	}
	return out, nil
}

func tempName(name string) bool {
	return strings.HasPrefix(name, ".tmp-") || strings.HasPrefix(name, ".tmp_") || strings.HasSuffix(name, ".tmp")
}

// Apply deletes the listed relative paths except store keys still in keep, and drops matching index entries.
func Apply(root string, items *store.Store, preview Preview, keep map[string][]string) error {
	named := map[string]map[string]bool{}
	for g, keys := range keep {
		named[g] = map[string]bool{}
		for _, k := range keys {
			named[g][k] = true
		}
	}
	var refs []store.Ref
	for _, it := range preview.Items {
		if it.Kind == "store" {
			game, key, ok := strings.Cut(strings.TrimPrefix(it.Rel, "store/"), "/")
			if ok && named[game][key] {
				continue
			}
			if ok {
				refs = append(refs, store.Ref{Game: game, Key: key})
			}
		}
		abs, confErr := confined(root, it.Rel)
		if confErr != nil {
			continue
		}
		_ = fsx.RemoveAll(filepath.Clean(abs))
	}
	return items.Remove(refs)
}

func cacheTTL(rel string) (time.Duration, bool) {
	switch {
	case strings.HasPrefix(rel, "nexus/details-"):
		return detailsTTL, true
	case strings.HasPrefix(rel, "nexus/categories-"):
		return categoriesTTL, true
	case strings.HasPrefix(rel, "dataset-"):
		return datasetTTL, true
	case rel == "smapi-updates.json":
		return updatesTTL, true
	case strings.HasSuffix(rel, "-releases.json"):
		return releasesTTL, true
	default:
		return 0, false
	}
}

func fileAge(path string, now time.Time) time.Duration {
	if b, err := fsx.ReadFile(path); err == nil {
		var wrap fetchedFile
		if json.Unmarshal(b, &wrap) == nil && !wrap.Fetched.IsZero() {
			return now.Sub(wrap.Fetched)
		}
	}
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return 0
	}
	return now.Sub(info.ModTime())
}

func dirSize(dir string) int64 {
	n, _ := datadir.Size(dir)
	return n
}

func dirEmpty(dir string) (bool, error) {
	ents, err := os.ReadDir(filepath.Clean(dir))
	if err != nil {
		return false, err
	}
	return len(ents) == 0, nil
}

func confined(root, rel string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	abs := filepath.Join(absRoot, filepath.FromSlash(rel))
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	if !datadir.UnderRoot(absRoot, abs) {
		return "", fs.ErrInvalid
	}
	return abs, nil
}
