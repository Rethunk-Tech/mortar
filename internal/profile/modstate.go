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

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
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
	return configPathIn(dir)
}

func configPathIn(dir string) (string, error) {
	cfg, err := configInMod(dir)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(cfg)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("config.json is missing: %w", fs.ErrNotExist)
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
	mine, found, err := safefs.ReadFileIfExists(filepath.Join(folder, configFile))
	if err != nil {
		return ModState{}, err
	}
	if !found {
		return st, nil
	}
	src, err := s.items.Path(game, e.StoreKey())
	if errors.Is(err, store.ErrNotFound) {
		st.Config = ConfigGenerated
		return st, nil
	}
	if err != nil {
		return ModState{}, err
	}
	shipped, found, err := safefs.ReadFileIfExists(filepath.Join(src, filepath.FromSlash(m.Folder), configFile))
	switch {
	case err != nil:
		return ModState{}, err
	case !found:
		st.Config = ConfigGenerated
	default:
		st.Config = configState(mine, shipped)
	}
	return st, nil
}

// configState compares the profile's config.json with the shipped one as both read in the panel, where a rewritten
// file that only differs in spacing is still the shipped default.
func configState(mine, shipped []byte) string {
	m, merr := rewriteConfigJSON(mine)
	s, serr := rewriteConfigJSON(shipped)
	if merr == nil && serr == nil && bytes.Equal(m, s) {
		return ConfigDefault
	}
	if bytes.Equal(mine, shipped) {
		return ConfigDefault
	}
	return ConfigChanged
}

func (s *Store) previousVersion(game string, e Entry, uniqueID mod.ID) string {
	if e.PreviousKey == "" {
		return ""
	}
	src, err := s.items.Path(game, e.PreviousKey)
	if err != nil {
		return ""
	}
	if found, err := smapimanifest.Scan(src); err == nil {
		for _, f := range found {
			if mod.Equal(f.ModID(), uniqueID) {
				return f.Version
			}
		}
	}
	// A package mod (a Thunderstore one) has no SMAPI manifest; the source it was installed from carries its version.
	if e.PreviousSource != nil {
		return e.PreviousSource.Version
	}
	return ""
}

// ResetConfig deletes the mod's config.json so the mod writes a fresh one the next time it runs.
func (s *Store) ResetConfig(game, id, key string, uniqueID mod.ID) error {
	if err := s.lockUnlocked(game, id); err != nil {
		return err
	}
	defer s.mu.Unlock()
	folder, err := s.modFolderLocked(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(folder, configFile)); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return s.editConfigLocked(game, id, HistoryEvent{Change: ChangeConfigReset, Name: uniqueID.Local()}, func() error {
		return safefs.Remove(filepath.Join(folder, configFile))
	})
}

// configInMod joins configFile onto the mod folder and refuses a folder with a link in it.
func configInMod(modDir string) (string, error) {
	root := filepath.Clean(modDir)
	cfg := filepath.Join(root, configFile)
	out, err := filepath.Rel(root, cfg)
	if err != nil || filepath.IsAbs(out) || strings.HasPrefix(out, "..") {
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
	return readConfigAt(path)
}

func readConfigAt(path string) (string, error) {
	b, err := safefs.ReadFile(path)
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
	return s.writeConfig(game, id, key, uniqueID, contents, HistoryEvent{Change: ChangeConfigEdited, Name: uniqueID.Local()})
}

func (s *Store) writeConfig(game, id, key string, uniqueID mod.ID, contents string, note HistoryEvent) error {
	if err := s.lockUnlocked(game, id); err != nil {
		return err
	}
	defer s.mu.Unlock()
	folder, err := s.modFolderLocked(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	path, err := configInMod(folder)
	if err != nil {
		return err
	}
	rewritten, err := rewriteConfigJSON([]byte(contents))
	if err != nil {
		return fmt.Errorf("config.json is not valid JSON: %w", err)
	}
	return s.editConfigLocked(game, id, note, func() error {
		return safefs.WriteFileAtomic(path, rewritten, 0o600)
	})
}

// editConfigLocked runs a config.json change between two history events: one capturing the file as it was, when the
// newest event does not already, and one capturing the result, so the edit can be undone.
func (s *Store) editConfigLocked(game, id string, note HistoryEvent, edit func() error) error {
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return err
	}
	if err := s.captureBeforeEdit(dir, p); err != nil {
		return err
	}
	if modNamed(note.Change) {
		note.Name = modDisplayName(p.Entries, note.Name)
	}
	if err := edit(); err != nil {
		return err
	}
	s.historyKind, s.historyNote = historyConfigEdit, note
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
		if snap, ok := snapshotEntries(&data, head.SnapshotID); ok && reflect.DeepEqual(snap, p.Entries) {
			idx, _ := historyConfigIndex(dir, p.Entries)
			if captured, err := readSnapshotIndex(dir, head.SnapshotID); err == nil && reflect.DeepEqual(idx, captured) {
				return nil
			}
		}
	}
	_, err = appendHistory(dir, HistoryEvent{Kind: historyRestored, Change: ChangeBeforeEdit, Count: 1}, p.Entries, s.historyKeep())
	return err
}
