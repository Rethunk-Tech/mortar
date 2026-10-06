package profile

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

const configFile = "config.json"

// ConfigPath is the mod's config.json, only if that file sits in the profile's mod folder.
func (s *Store) ConfigPath(game, id, key string, uniqueID mod.ID) (string, error) {
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
	// Gmcm is true when the bridge captured the mod's in-game settings menu, which Mortar can then edit.
	Gmcm bool `json:"gmcm"`
}

// FindMod is the entry and mod holding uniqueID; an empty key searches every entry.
func (p Profile) FindMod(key string, uniqueID mod.ID) (Entry, Component, bool) {
	for _, e := range p.Entries {
		if key != "" && e.Key != key {
			continue
		}
		if i := slices.IndexFunc(e.Mods, func(m Component) bool { return mod.Equal(m.ID, uniqueID) }); i >= 0 {
			return e, e.Mods[i], true
		}
	}
	return Entry{}, Component{}, false
}

// ModState reads the mod's rollback target and the state of its config.json.
func (s *Store) ModState(game, id, key string, uniqueID mod.ID) (ModState, error) {
	p, err := s.read(game, id)
	if err != nil {
		return ModState{}, err
	}
	e, m, ok := p.FindMod(key, uniqueID)
	if !ok {
		return ModState{}, errors.New("no such mod in this profile")
	}
	st := ModState{PreviousVersion: s.previousVersion(game, e, m.ID), Config: ConfigNone}
	if dir, err := s.ProfileDir(game, id); err == nil {
		st.Gmcm = hasGmcmCapture(dir, m.ID)
	}
	folder, err := s.ModFolder(game, id, e.Key, m.ID)
	if errors.Is(err, ErrNoModFolder) {
		return st, nil
	}
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

func (s *Store) previousVersion(game string, e Entry, uniqueID mod.ID) string {
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
		if mod.Equal(f.ModID(), uniqueID) {
			return f.Version
		}
	}
	return ""
}

// ResetConfig deletes the mod's config.json so the mod writes a fresh one the next time it runs.
func (s *Store) ResetConfig(game, id, key string, uniqueID mod.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	folder, err := s.modFolderLocked(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(folder, configFile)); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return s.editConfigLocked(game, id, "Reset "+uniqueID.Local()+" settings", func() error {
		return os.Remove(filepath.Join(folder, configFile))
	})
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
func (s *Store) ReadConfig(game, id, key string, uniqueID mod.ID) (string, error) {
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

// WriteConfig replaces the mod's config.json atomically after checking JSON and the path, and records a history event.
func (s *Store) WriteConfig(game, id, key string, uniqueID mod.ID, contents string) error {
	return s.writeConfig(game, id, key, uniqueID, contents, "Edited "+uniqueID.Local()+" settings")
}

func (s *Store) writeConfig(game, id, key string, uniqueID mod.ID, contents, label string) error {
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
	return s.editConfigLocked(game, id, label, func() error {
		return datadir.WriteFile(path, rewritten, 0o600)
	})
}

// editConfigLocked runs a config.json change between two history events: one capturing the file as it was, when the
// newest event does not already, and one capturing the result, so the edit can be undone.
func (s *Store) editConfigLocked(game, id, label string, edit func() error) error {
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return err
	}
	if err := s.captureBeforeEdit(dir, p); err != nil {
		return err
	}
	if err := edit(); err != nil {
		return err
	}
	s.historyKind, s.historyLabel = historyConfigEdit, label
	_, err = s.updateLocked(game, id, func(*Profile, string) error { return nil })
	return err
}

// captureBeforeEdit appends a "Before this change" event unless the newest event already holds every config as it is.
func (s *Store) captureBeforeEdit(dir string, p Profile) error {
	data, err := readHistory(dir)
	if err != nil {
		return err
	}
	if n := len(data.Events); n > 0 {
		head := data.Events[n-1]
		if snap, ok := snapshotEntries(&data, head.SnapshotID); ok && entriesEqual(snap, p.Entries) {
			idx, _ := historyConfigIndex(dir, p.Entries)
			if captured, err := readSnapshotIndex(dir, head.SnapshotID); err == nil && reflect.DeepEqual(idx, captured) {
				return nil
			}
		}
	}
	_, err = appendHistory(dir, HistoryEvent{Kind: historyRestored, Label: "Before this change", Count: 1}, p.Entries, s.historyKeep())
	return err
}
