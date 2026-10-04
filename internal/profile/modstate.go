package profile

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/store"
)

const configFile = "config.json"

// ConfigPath is the mod's config.json, only if that file sits in the profile's mod folder.
func (s *Store) ConfigPath(game, id, key, uniqueID string) (string, error) {
	dir, err := s.ModFolder(game, id, key, uniqueID)
	if err != nil {
		return "", err
	}
	cfg, err := configInMod(dir, configFile)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(cfg)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("config.json is missing")
	}
	return cfg, nil
}

// Config states, where a store item that is gone counts as shipping none: "none" when the mod has no config.json (yet), "default" when it is byte for byte the one the
// mod ships, "changed" when it differs from the shipped one, and "generated" when the mod ships none, so the
// file is the mod's own doing and nothing says whether the user edited it.
const (
	ConfigNone      = "none"
	ConfigDefault   = "default"
	ConfigChanged   = "changed"
	ConfigGenerated = "generated"
)

// ModState is what the mod detail panel shows beyond the manifest. PreviousVersion is empty when there is
// nothing to roll back to, including when the store no longer holds the previous version.
type ModState struct {
	PreviousVersion string `json:"previousVersion"`
	Config          string `json:"config"`
}

func entryMod(p Profile, key, uniqueID string) (Entry, EntryMod, bool) {
	for _, e := range p.Entries {
		if key != "" && e.Key != key {
			continue
		}
		if i := slices.IndexFunc(e.Mods, func(m EntryMod) bool { return SameID(m.UniqueID, uniqueID) }); i >= 0 {
			return e, e.Mods[i], true
		}
	}
	return Entry{}, EntryMod{}, false
}

// ModState reads the mod's rollback target and the state of its config.json.
func (s *Store) ModState(game, id, key, uniqueID string) (ModState, error) {
	p, err := s.read(game, id)
	if err != nil {
		return ModState{}, err
	}
	e, m, ok := entryMod(p, key, uniqueID)
	if !ok {
		return ModState{}, errors.New("no such mod in this profile")
	}
	st := ModState{PreviousVersion: s.previousVersion(game, e, m.UniqueID), Config: ConfigNone}
	folder, err := s.ModFolder(game, id, e.Key, m.UniqueID)
	if err != nil {
		return ModState{}, err
	}
	mine, err := fsx.ReadFile(filepath.Join(folder, configFile))
	if errors.Is(err, fs.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return ModState{}, err
	}
	src, err := s.items.Path(game, e.Key)
	if errors.Is(err, store.ErrNotFound) {
		st.Config = ConfigGenerated
		return st, nil
	}
	if err != nil {
		return ModState{}, err
	}
	shipped, err := fsx.ReadFile(filepath.Join(src, filepath.FromSlash(m.Folder), configFile))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		st.Config = ConfigGenerated
	case err != nil:
		return ModState{}, err
	case bytes.Equal(mine, shipped):
		st.Config = ConfigDefault
	default:
		st.Config = ConfigChanged
	}
	return st, nil
}

func (s *Store) previousVersion(game string, e Entry, uniqueID string) string {
	if e.PreviousKey == "" {
		return ""
	}
	src, err := s.items.Path(game, e.PreviousKey)
	if err != nil {
		return ""
	}
	found, err := manifest.Scan(src)
	if err != nil {
		return ""
	}
	for _, f := range found {
		if SameID(f.UniqueID, uniqueID) {
			return f.Version
		}
	}
	return ""
}

// ResetConfig deletes the mod's config.json so the mod writes a fresh one the next time it runs.
func (s *Store) ResetConfig(game, id, key, uniqueID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	folder, err := s.modFolderLocked(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(folder, configFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// configInMod joins rel onto the mod folder and refuses anything that leaves it.
func configInMod(modDir, rel string) (string, error) {
	root := filepath.Clean(modDir)
	if rel == "" {
		rel = configFile
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("config.json is not in the mod folder")
	}
	cfg := filepath.Join(root, filepath.FromSlash(rel))
	out, err := filepath.Rel(root, cfg)
	if err != nil || filepath.IsAbs(out) || strings.HasPrefix(out, "..") || out != configFile {
		return "", fmt.Errorf("config.json is not in the mod folder")
	}
	return cfg, nil
}

// ReadConfig returns the mod's config.json text.
func (s *Store) ReadConfig(game, id, key, uniqueID string) (string, error) {
	path, err := s.ConfigPath(game, id, key, uniqueID)
	if err != nil {
		return "", err
	}
	b, err := fsx.ReadFile(path)
	if err != nil {
		return "", err
	}
	normalized, err := rewriteConfigJSON(b)
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}

// WriteConfig replaces the mod's config.json atomically after checking JSON and the path.
func (s *Store) WriteConfig(game, id, key, uniqueID, contents string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	folder, err := s.modFolderLocked(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	path, err := configInMod(folder, configFile)
	if err != nil {
		return err
	}
	rewritten, err := rewriteConfigJSON([]byte(contents))
	if err != nil {
		return fmt.Errorf("config.json is not valid JSON: %w", err)
	}
	return datadir.WriteFile(path, rewritten, 0o600)
}
