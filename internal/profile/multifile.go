package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/store"
)

// MergeAsk is the choice to attach a Nexus file to an entry already taken from that page, or to keep it separate.
type MergeAsk struct {
	EntryKey   string `json:"entryKey"`
	Label      string `json:"label"`
	DefaultAdd bool   `json:"defaultAdd"`
}

// DefaultMerge is true for OPTIONAL and MISCELLANEOUS Nexus file categories.
func DefaultMerge(category string) bool {
	switch strings.ToUpper(strings.TrimSpace(category)) {
	case "OPTIONAL", "MISCELLANEOUS":
		return true
	default:
		return false
	}
}

// SamePageAsk reports an existing Nexus entry from modID when fileID is a different file on that page.
func SamePageAsk(p Profile, modID, fileID int, category string) (MergeAsk, bool) {
	if modID <= 0 || fileID <= 0 {
		return MergeAsk{}, false
	}
	incoming := store.NexusKey(modID, fileID)
	for _, e := range p.Entries {
		if e.Source.Kind != KindNexus || e.Source.ModID != modID {
			continue
		}
		if e.Source.FileID == fileID || e.Key == incoming || slices.Contains(e.ExtraStoreKeys, incoming) {
			return MergeAsk{}, false
		}
		return MergeAsk{EntryKey: e.Key, Label: entryLabel(e), DefaultAdd: DefaultMerge(category)}, true
	}
	return MergeAsk{}, false
}

// NewestFromPage is the highest Nexus file id the profile holds from modID's page, 0 when it holds none.
func NewestFromPage(p Profile, modID int) int {
	newest := 0
	for _, e := range p.Entries {
		if e.Source.Kind == KindNexus && e.Source.ModID == modID {
			newest = max(newest, e.Source.FileID)
		}
	}
	return newest
}

// AddExtra copies extraKey into mods/<entryKey>/<extraKey>/ and records it on the entry.
func (s *Store) AddExtra(game, id, entryKey, extraKey string, source Source) (Profile, error) {
	p, err := s.updateMods(game, id, func(p *Profile, dir string) error {
		return s.addExtraLocked(game, p, dir, entryKey, extraKey, source)
	})
	if err != nil {
		return Profile{}, err
	}
	if err := s.RecordModsSnapshot(game, id); err != nil {
		return Profile{}, err
	}
	return p, s.items.Touch(game, extraKey)
}

func (s *Store) addExtraLocked(game string, p *Profile, dir, entryKey, extraKey string, source Source) error {
	ei := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == entryKey })
	if ei < 0 {
		return fmt.Errorf("%q is not in this profile", entryKey)
	}
	e := &p.Entries[ei]
	if extraKey == "" || extraKey == e.Key || slices.Contains(e.ExtraStoreKeys, extraKey) {
		return &DuplicateError{Key: extraKey, Label: entryLabel(*e)}
	}
	entryDir := liveEntryDir(filepath.Join(dir, "mods"), e.Key)
	if err := s.copyExtraInto(game, p.ID, entryDir, extraKey, source.fomodMap()); err != nil {
		return err
	}
	e.ExtraStoreKeys = append(append([]string{}, e.ExtraStoreKeys...), extraKey)
	return s.refreshEntryMods(e, entryDir)
}

// UpdateExtra replaces one extra store item on an entry with another.
func (s *Store) UpdateExtra(game, id, entryKey, oldExtra, newExtra string) (Profile, error) {
	p, err := s.updateMods(game, id, func(p *Profile, dir string) error {
		ei := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == entryKey })
		if ei < 0 {
			return fmt.Errorf("%q is not in this profile", entryKey)
		}
		e := &p.Entries[ei]
		xi := slices.Index(e.ExtraStoreKeys, oldExtra)
		if xi < 0 {
			return fmt.Errorf("%q is not an extra file of this entry", oldExtra)
		}
		if newExtra == "" || newExtra == e.Key || newExtra == oldExtra {
			return fmt.Errorf("%q is not a new extra store key", newExtra)
		}
		if slices.Contains(e.ExtraStoreKeys, newExtra) {
			return &DuplicateError{Key: newExtra, Label: entryLabel(*e)}
		}
		entryDir := liveEntryDir(filepath.Join(dir, "mods"), e.Key)
		e.PreviousExtraStoreKeys = growPreviousExtras(*e)
		e.PreviousExtraStoreKeys[xi] = oldExtra
		if err := s.fillOneExtraUpdate(game, p.ID, entryDir, entryDir, oldExtra, newExtra, *e); err != nil {
			return err
		}
		if oldExtra != newExtra {
			if err := os.RemoveAll(filepath.Join(entryDir, oldExtra)); err != nil {
				return err
			}
		}
		e.ExtraStoreKeys[xi] = newExtra
		return s.refreshEntryMods(e, entryDir)
	})
	if err != nil {
		return Profile{}, err
	}
	if err := s.RecordModsSnapshot(game, id); err != nil {
		return Profile{}, err
	}
	return p, s.items.Touch(game, newExtra)
}

// UpdateMultiFile swaps the entry's primary store item and every extra as one unit.
func (s *Store) UpdateMultiFile(game, id, oldKey, newKey string, source *Source) (Profile, error) {
	return s.moveTo(game, id, oldKey, newKey, source)
}

func (s *Store) fillExtrasUpdate(game, id, modsDir, oldEntryKey, tmp string, e, ne Entry) error {
	oldDir := liveEntryDir(modsDir, oldEntryKey)
	for i, newKey := range ne.ExtraStoreKeys {
		oldProfKey := newKey
		if i < len(e.ExtraStoreKeys) {
			oldProfKey = e.ExtraStoreKeys[i]
		}
		if err := s.fillOneExtraUpdate(game, id, oldDir, tmp, oldProfKey, newKey, e); err != nil {
			return err
		}
	}
	for _, stale := range e.ExtraStoreKeys {
		if slices.Contains(ne.ExtraStoreKeys, stale) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(tmp, stale))
	}
	return nil
}

func (s *Store) fillOneExtraUpdate(game, id, oldDir, tmp, oldProfKey, newKey string, e Entry) error {
	extraSrc, extraTmp, err := s.layoutItem(game, id, newKey, nil)
	if extraTmp != "" {
		defer func() { _ = os.RemoveAll(extraTmp) }()
	}
	if err != nil {
		return err
	}
	found, err := manifest.Scan(extraSrc)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		return &NoModError{Key: newKey}
	}
	scratch, err := os.MkdirTemp(tmp, tempPrefix)
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	if err := datadir.MaterializeTree(extraSrc, scratch); err != nil {
		return err
	}
	profRoot := filepath.Join(oldDir, oldProfKey)
	oldPrefix := filepath.ToSlash(oldProfKey) + "/"
	for _, om := range e.Mods {
		oldFolder := filepath.ToSlash(om.Folder)
		if !strings.HasPrefix(oldFolder, oldPrefix) {
			continue
		}
		if !slices.ContainsFunc(found, func(m manifest.Mod) bool { return sameID(m.UniqueID, om.UniqueID) }) {
			continue
		}
		rel := strings.TrimPrefix(oldFolder, oldPrefix)
		cur := filepath.Join(profRoot, filepath.FromSlash(rel))
		if !exists(cur) {
			continue
		}
		configOnly := deleteOldVersion(found, om.UniqueID)
		err = carryOverWalk(cur, filepath.Join(extraSrc, filepath.FromSlash(rel)), filepath.Join(scratch, filepath.FromSlash(rel)), configOnly)
		if err != nil {
			return err
		}
	}
	dest := filepath.Join(tmp, newKey)
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	return os.Rename(scratch, dest)
}

// InstallNexusExtra unpacks a Nexus archive into the store and adds it as an extra file of entryKey.
func (s *Store) InstallNexusExtra(game, id, entryKey, path string, source Source) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	key := store.NexusKey(source.ModID, source.FileID)
	if err := s.items.AddArchiveKey(game, key, path); err != nil {
		return InstallResult{}, installError(err)
	}
	p, err := s.AddExtra(game, id, entryKey, key, source)
	if ask, ok := s.installQuestion(game, id, key, source, err); ok {
		return ask, nil
	}
	if err != nil {
		return InstallResult{}, installError(err)
	}
	res := InstallResult{Profile: p, Added: []string{}}
	for _, e := range p.Entries {
		if e.Key != entryKey {
			continue
		}
		for _, m := range e.Mods {
			res.Added = append(res.Added, m.Name)
		}
	}
	return res, nil
}

func growPreviousExtras(e Entry) []string {
	out := append([]string(nil), e.PreviousExtraStoreKeys...)
	for len(out) < len(e.ExtraStoreKeys) {
		out = append(out, e.ExtraStoreKeys[len(out)])
	}
	return out
}

func (s *Store) copyExtraInto(game, id, entryDir, extraKey string, choices map[string]map[string][]string) error {
	src, tmp, err := s.layoutItem(game, id, extraKey, choices)
	if tmp != "" {
		defer func() { _ = os.RemoveAll(tmp) }()
	}
	if err != nil {
		return err
	}
	found, err := manifest.Scan(src)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		return &NoModError{Key: extraKey}
	}
	dest := filepath.Join(entryDir, extraKey)
	if err := os.MkdirAll(entryDir, 0o700); err != nil {
		return err
	}
	scratch, err := os.MkdirTemp(entryDir, tempPrefix)
	if err != nil {
		return err
	}
	if err := datadir.MaterializeTree(src, scratch); err != nil {
		return errors.Join(err, os.RemoveAll(scratch))
	}
	if err := os.Rename(scratch, dest); err != nil {
		return errors.Join(err, os.RemoveAll(scratch))
	}
	return nil
}

func (s *Store) refreshEntryMods(e *Entry, entryDir string) error {
	found, err := manifest.Scan(entryDir)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		return &NoModError{Key: e.Key}
	}
	prev := e.Mods
	e.Mods = entryMods(found)
	kept := e.Disabled[:0]
	for _, id := range e.Disabled {
		if slices.ContainsFunc(e.Mods, func(m EntryMod) bool { return sameID(m.UniqueID, id) }) {
			kept = append(kept, id)
		}
	}
	e.Disabled = kept
	startEnabled := true
	if s.NewModsEnabled != nil {
		startEnabled = s.NewModsEnabled()
	}
	if !startEnabled {
		for _, m := range e.Mods {
			if slices.ContainsFunc(prev, func(old EntryMod) bool { return sameID(old.UniqueID, m.UniqueID) }) {
				continue
			}
			if !hasID(e.Disabled, m.UniqueID) {
				e.Disabled = append(e.Disabled, m.UniqueID)
			}
		}
	}
	return nil
}

func liveEntryDir(modsDir, key string) string {
	dotted := filepath.Join(modsDir, "."+key)
	if exists(dotted) && !exists(filepath.Join(modsDir, key)) {
		return dotted
	}
	return filepath.Join(modsDir, key)
}
