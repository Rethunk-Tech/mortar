package launchsvc

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// cacheClearing is the profile's cacheClearing setting: auto clears the game's caches when the mod set changed.
func (s *Service) cacheClearing(gameID, profileID, installID string) string {
	return settings.ResolveAt(s.settings.Get(), "cacheClearing", settings.Scope{Game: gameID, Install: s.pinOf(gameID, profileID, installID), Profile: profileID}, launchOverrides(s.profiles, gameID, profileID))
}

// clearCaches deletes the game's rebuildable caches (the catalog's caches list) on the first launch after the profile's
// mod set changed, and records the set it cleared for. It never fails the launch: a problem is logged, and the set is
// then not recorded, so the next launch tries again.
func (s *Service) clearCaches(gameID, profileID, installID string) {
	info, ok := components.Game(gameID)
	if !ok || len(info.Caches) == 0 || s.profiles == nil || profileID == "" {
		return
	}
	if s.cacheClearing(gameID, profileID, installID) == settings.CacheClearingOff {
		return
	}
	hash, err := s.profiles.ModSetHash(gameID, profileID)
	if err != nil {
		log.Printf("launch: %s caches: read the mod set: %v", gameID, err)
		return
	}
	if hash == s.profiles.CacheSet(gameID, profileID) {
		log.Printf("launch: %s mod set unchanged, nothing cleared", gameID)
		return
	}
	cleared, failed := 0, false
	for _, c := range info.Caches {
		root, err := game.PathFor(s.home, s.settings.Get(), gameID, s.pinOf(gameID, profileID, installID), c.Role)
		if err != nil {
			log.Printf("launch: %s cache %s: %v", gameID, c.Path, err)
			failed = true
			continue
		}
		n, err := clearCache(root, c.Path)
		cleared += n
		if err != nil {
			log.Printf("launch: %s cache %s: %v", gameID, c.Path, err)
			failed = true
		}
	}
	log.Printf("launch: %s cleared %d cache files", gameID, cleared)
	if failed {
		return
	}
	if err := s.profiles.SetCacheSet(gameID, profileID, hash); err != nil {
		log.Printf("launch: %s caches: record the mod set: %v", gameID, err)
	}
}

// clearCache deletes the file rel names below root, or the contents of the folder it names, and returns how many
// files went. A path that leaves root, by `..` or through a symlink, is refused; a missing path is nothing to do; the
// root and a named folder themselves stay.
func clearCache(root, rel string) (int, error) {
	if rel == "" || strings.Contains(rel, `\`) || !filepath.IsLocal(filepath.FromSlash(rel)) {
		return 0, fmt.Errorf("%q is not a path below its folder", rel)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	target := filepath.Join(realRoot, filepath.FromSlash(rel))
	real, err := filepath.EvalSymlinks(filepath.Dir(target))
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if real != filepath.Dir(target) {
		return 0, fmt.Errorf("%q leaves %s through a link", rel, root)
	}
	info, err := os.Lstat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return 0, fmt.Errorf("%q is a link", rel)
	case info.Mode().IsRegular():
		if err := os.Remove(target); err != nil {
			return 0, err
		}
		return 1, nil
	case info.IsDir():
		return clearFolder(target)
	}
	return 0, nil
}

// clearFolder deletes what is inside dir: files and links (never followed) and the folders below, not dir.
func clearFolder(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	var errs []error
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			sub, err := clearFolder(p)
			n += sub
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if err := os.Remove(p); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := os.Remove(p); err != nil {
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}
