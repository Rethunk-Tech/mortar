package profile

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/store"
)

const configFile = "config.json"

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
		if i := slices.IndexFunc(e.Mods, func(m EntryMod) bool { return sameID(m.UniqueID, uniqueID) }); i >= 0 {
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
		if sameID(f.UniqueID, uniqueID) {
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
