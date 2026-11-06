package datasvc

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

const (
	detailsTTL    = 24 * time.Hour
	categoriesTTL = 30 * 24 * time.Hour
	datasetTTL    = 30 * 24 * time.Hour
	// signedListTTL drops a signed per-game list (known-broken mods, log suppressions) or a profile's last problem check not rewritten for a month;
	// the next open fetches it again.
	signedListTTL = 30 * 24 * time.Hour
	changelogsTTL = 4 * time.Hour
	collectionTTL = 15 * time.Minute
	standingTTL   = 24 * time.Hour
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

// add appends one removable path and counts its size.
func (p *Preview) add(kind, label, rel string, size int64) {
	p.Items = append(p.Items, Item{Kind: kind, Label: label, Rel: rel, Size: size})
	p.Total += size
}

// Select lists store items no referenced key names, cache files past their expiry, and leftover temp folders.
func Select(root string, items *store.Store, referenced map[string][]string, now time.Time, nameOf func(string, string) string) (Preview, error) {
	out := Preview{Items: []Item{}}
	refs, err := items.Unreferenced(referenced)
	if err != nil {
		return Preview{}, err
	}
	for _, r := range refs {
		var n int64
		if dir, err := items.Dir(r.Game, r.Key); err == nil {
			n = dirSize(dir)
		}
		label := r.Game + "/" + r.Key
		if nameOf != nil {
			if named := nameOf(r.Game, r.Key); named != "" {
				label = named
			}
		}
		out.add("store", label, filepath.ToSlash(filepath.Join("store", r.Game, r.Key)), n)
	}
	err = selectExpiredCache(root, now, &out)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Preview{}, err
	}
	if err := selectTemp(root, now, &out); err != nil {
		return Preview{}, err
	}
	return out, nil
}

// selectExpiredCache adds the cache files older than their time to live.
func selectExpiredCache(root string, now time.Time, out *Preview) error {
	return filepath.WalkDir(filepath.Clean(filepath.Join(root, "cache")), func(path string, d fs.DirEntry, err error) error {
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
			out.add("cache", slash, slash, info.Size())
		}
		return nil
	})
}

// selectTemp adds the stale temp files and the temp folders that are empty or Mortar's own scratch.
func selectTemp(root string, now time.Time, out *Preview) error {
	return filepath.WalkDir(filepath.Clean(root), func(path string, d fs.DirEntry, err error) error {
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
		if d.IsDir() && (slash == "profiles" || slash == "store") {
			return fs.SkipDir
		}
		name := d.Name()
		if d.Type().IsRegular() && tempName(name) && fileAge(path, now) > backup.MinGap {
			if info, infoErr := d.Info(); infoErr == nil {
				out.add("temp", slash, slash, info.Size())
			}
			return nil
		}
		if !d.IsDir() || !tempName(name) {
			return nil
		}
		empty, emptyErr := dirEmpty(path)
		if emptyErr != nil {
			return emptyErr
		}
		if !empty && !strings.HasPrefix(name, ".tmp-") && !strings.HasPrefix(name, ".tmp_") {
			return nil
		}
		out.add("temp", slash, slash, dirSize(path))
		return fs.SkipDir
	})
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
		_ = safefs.RemoveAll(filepath.Clean(abs))
	}
	return errors.Join(items.Remove(refs), items.PruneDangling(keep))
}

// cacheTTL is how long the cache file rel may sit; a retired file has none.
func cacheTTL(rel string) (time.Duration, bool) {
	if meta.Retired(rel) {
		return 0, true
	}
	switch {
	case strings.HasPrefix(rel, meta.NexusDetailsPrefix), strings.HasPrefix(rel, meta.NexusPagePrefix):
		return detailsTTL, true
	case strings.HasPrefix(rel, meta.NexusCategoriesPrefix):
		return categoriesTTL, true
	case strings.HasPrefix(rel, meta.NexusChangelogsPrefix):
		return changelogsTTL, true
	case strings.HasPrefix(rel, meta.StandingPrefix):
		return standingTTL, true
	case strings.HasPrefix(rel, meta.CollectionPrefix):
		return collectionTTL, true
	case strings.HasPrefix(rel, meta.DatasetPrefix):
		return datasetTTL, true
	case strings.HasPrefix(rel, meta.KnownBrokenPrefix), strings.HasPrefix(rel, meta.LogSuppressionsPrefix), strings.HasPrefix(rel, meta.ProblemsLastPrefix):
		return signedListTTL, true
	case rel == "smapi-updates.json":
		return updatesTTL, true
	case strings.HasSuffix(rel, "-releases-v2.json"):
		return releasesTTL, true
	default:
		return 0, false
	}
}

// fetchedAt is the "fetched" time of a cache file, read by scanning keys so a large payload is skipped, not parsed
// into memory.
func fetchedAt(path string) (time.Time, bool) {
	f, err := safefs.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(bufio.NewReader(f))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return time.Time{}, false
	}
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return time.Time{}, false
		}
		if k == "fetched" {
			var at time.Time
			if dec.Decode(&at) != nil || at.IsZero() {
				return time.Time{}, false
			}
			return at, true
		}
		var skip json.RawMessage
		if dec.Decode(&skip) != nil {
			return time.Time{}, false
		}
	}
	return time.Time{}, false
}

func fileAge(path string, now time.Time) time.Duration {
	if at, ok := fetchedAt(path); ok {
		return now.Sub(at)
	}
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return 0
	}
	return now.Sub(info.ModTime())
}

func dirSize(dir string) int64 {
	n, _ := safefs.Size(dir)
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
	if !safefs.UnderRoot(absRoot, abs) {
		return "", fs.ErrInvalid
	}
	return abs, nil
}
