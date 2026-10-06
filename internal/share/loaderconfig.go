package share

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// loaderPrefix is where a .mortar file keeps the loader's own config files, by their path in the profile.
const loaderPrefix = "loader/"

// loaderConfigExts are the text formats a loader's config folder holds; anything else there (a cache, a binary) stays home.
var loaderConfigExts = []string{".cfg", ".json", ".toml", ".ini", ".yml", ".yaml", ".txt"}

// LoaderConfig is one file of the loader's config folders, by slash path relative to the profile's folder.
type LoaderConfig struct {
	Path string `json:"path"`
	Data []byte `json:"data"`
}

// LoaderConfigRoots are the profile-relative config folders of the loader the profile runs (loaderID, "" for the
// game's primary), or none when it keeps its settings per mod.
func LoaderConfigRoots(gameID, loaderID string) []string {
	l, ok := gamereg.LoaderOf(gameID, loaderID)
	if !ok {
		return nil
	}
	if c, ok := l.(loader.WithConfig); ok {
		return c.ConfigDirs()
	}
	return nil
}

// UnderRoots reports whether the slash path p lies inside one of roots.
func UnderRoots(p string, roots []string) bool {
	return slices.ContainsFunc(roots, func(r string) bool {
		return strings.HasPrefix(strings.ToLower(p), strings.ToLower(strings.TrimSuffix(r, "/")+"/"))
	})
}

// validLoaderConfigPath accepts a clean relative slash path of plain segments ending in a text config extension.
func validLoaderConfigPath(p string) bool {
	if len(p) > maxRelPath || path.Clean(p) != p || !slices.Contains(loaderConfigExts, strings.ToLower(path.Ext(p))) {
		return false
	}
	for s := range strings.SplitSeq(p, "/") {
		if !validSegment(s) {
			return false
		}
	}
	return true
}

// companionState is the loader's companion state file, which holds a live token and never leaves the computer.
func companionState(gameID, loaderID string) string {
	l, ok := gamereg.LoaderOf(gameID, loaderID)
	if !ok {
		return ""
	}
	if c, ok := l.(loader.WithCompanion); ok {
		return c.Companion().StateFile
	}
	return ""
}

// readLoaderConfigs collects the text config files under the loader's config roots in profileDir; files over the
// size cap, not UTF-8, or with unusual names are skipped and returned as paths.
func readLoaderConfigs(profileDir, gameID, loaderID string) (found []LoaderConfig, skipped []string, err error) {
	state := companionState(gameID, loaderID)
	for _, root := range LoaderConfigRoots(gameID, loaderID) {
		err := filepath.WalkDir(filepath.Join(profileDir, filepath.FromSlash(root)), func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, fs.ErrNotExist) {
					return nil
				}
				return walkErr
			}
			if !d.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(profileDir, p)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if strings.EqualFold(rel, state) || !slices.Contains(loaderConfigExts, strings.ToLower(path.Ext(rel))) {
				return nil
			}
			data, err := readCapped(p)
			if errors.Is(err, errOverCap) || (err == nil && (!utf8.Valid(data) || !validLoaderConfigPath(rel))) {
				skipped = append(skipped, rel)
				return nil
			}
			if err != nil {
				return err
			}
			found = append(found, LoaderConfig{Path: rel, Data: data})
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	return found, skipped, nil
}
