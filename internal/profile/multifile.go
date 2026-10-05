package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/store"
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

// IncomingFile is a Nexus file arriving for a profile, with what the queue could learn about it.
type IncomingFile struct {
	ModID, FileID int
	Category      string
	// Files is the mod page's file list, nil when it could not be fetched.
	Files []nexus.File
	// ModIDs are the mods the file's archive holds, nil when it could not be read.
	ModIDs []mod.ID
}

// SamePageAsk reports an existing Nexus entry from in's page when in is a different file on that page. A file that
// updates an entry instead returns that entry's file id as updates, and no ask: the install replaces it.
func SamePageAsk(p Profile, in IncomingFile) (ask MergeAsk, updates int, ok bool) {
	if in.ModID <= 0 || in.FileID <= 0 {
		return MergeAsk{}, 0, false
	}
	incoming := store.NexusKey(in.ModID, in.FileID)
	var page []Entry
	for _, e := range p.Entries {
		if e.Source.Kind != KindNexus || e.Source.ModID != in.ModID || e.IsOverlay() {
			continue
		}
		if e.Source.FileID == in.FileID || e.Key == incoming || slices.Contains(e.ExtraStoreKeys, incoming) {
			return MergeAsk{}, 0, false
		}
		page = append(page, e)
	}
	if len(page) == 0 {
		return MergeAsk{}, 0, false
	}
	if e, ok := updatedEntry(page, in); ok {
		return MergeAsk{}, e.Source.FileID, false
	}
	target := page[0]
	if i := slices.IndexFunc(page, func(e Entry) bool { return sharesMods(e, in.ModIDs) }); i >= 0 {
		target = page[i]
	}
	return MergeAsk{EntryKey: target.Key, Label: entryLabel(target), DefaultAdd: DefaultMerge(in.Category)}, 0, true
}

// updatedEntry finds the entry in is a newer version of. The author's file_updates chain from an entry's file to in
// decides first, even when the mod was renamed inside, unless the page carries several main files: those are separate
// mods, and only the mods inside in tell which one it updates. Otherwise the mods inside in decide when they are
// known: exactly an entry's mods is its update, anything else is another file. Unknown (a FOMOD or a listing cut
// short), a retired main file is updated when it is the page's only main file the profile holds.
func updatedEntry(page []Entry, in IncomingFile) (Entry, bool) {
	pageMains := 0
	for _, f := range in.Files {
		if strings.EqualFold(f.Category, "MAIN") {
			pageMains++
		}
	}
	if len(in.ModIDs) == 0 || pageMains <= 1 {
		if i := slices.IndexFunc(page, func(e Entry) bool { return chainedTo(e.Source.FileID, in) }); i >= 0 {
			return page[i], true
		}
	}
	if len(in.ModIDs) > 0 {
		i := slices.IndexFunc(page, func(e Entry) bool {
			return len(e.Mods) == len(in.ModIDs) && !slices.ContainsFunc(e.Mods, func(m Component) bool {
				return !slices.ContainsFunc(in.ModIDs, func(id mod.ID) bool { return mod.Equal(id, m.ID) })
			})
		})
		if i < 0 {
			return Entry{}, false
		}
		return page[i], true
	}
	if !strings.EqualFold(in.Category, "MAIN") {
		return Entry{}, false
	}
	var mains []Entry
	for _, e := range page {
		if e.Source.Category == "" || strings.EqualFold(e.Source.Category, "MAIN") {
			mains = append(mains, e)
		}
	}
	if len(mains) != 1 {
		return Entry{}, false
	}
	for _, f := range in.Files {
		if f.FileID == mains[0].Source.FileID && (strings.EqualFold(f.Category, "OLD_VERSION") || strings.EqualFold(f.Category, "ARCHIVED")) {
			return mains[0], true
		}
	}
	return Entry{}, false
}

// chainedTo reports whether the author's file_updates chain leads from fileID to in.
func chainedTo(fileID int, in IncomingFile) bool {
	seen := map[int]bool{}
	for id := fileID; id != 0 && !seen[id]; {
		seen[id] = true
		next := 0
		for _, f := range in.Files {
			if f.FileID == id {
				next = f.ReplacedBy
			}
		}
		if next == in.FileID {
			return true
		}
		id = next
	}
	return false
}

func sharesMods(e Entry, ids []mod.ID) bool {
	return slices.ContainsFunc(e.Mods, func(m Component) bool {
		return slices.ContainsFunc(ids, func(id mod.ID) bool { return mod.Equal(id, m.ID) })
	})
}

// NewestFromPage is the highest Nexus file id the profile holds from modID's page, 0 when it holds none. When
// current is an optional file the profile holds, it is current: that file's own versions are what count.
func NewestFromPage(p Profile, modID, current int) int {
	newest := 0
	if current > 0 && slices.ContainsFunc(p.Entries, func(e Entry) bool {
		return e.IsOverlay() && e.Source.ModID == modID && e.Source.FileID == current
	}) {
		return current
	}
	for _, e := range p.Entries {
		if e.Source.Kind == KindNexus && e.Source.ModID == modID && !e.IsOverlay() {
			newest = max(newest, e.Source.FileID)
		}
	}
	return newest
}

// AddExtra copies extraKey into mods/<entryKey>/<extraKey>/ and records it on the entry.
func (s *Store) AddExtra(game, id, entryKey, extraKey string, source Source) (Profile, error) {
	p, err := s.updateModsRecorded(game, id, func(p *Profile, dir string) error {
		return s.addExtraLocked(game, p, dir, entryKey, extraKey, source)
	})
	if err != nil {
		return Profile{}, err
	}
	return p, s.items.Touch(game, extraKey)
}

// SplitExtra turns extraKey into its own profile entry, as a separate install of that store item would.
func (s *Store) SplitExtra(game, id, entryKey, extraKey string) (Profile, error) {
	p, err := s.updateModsRecorded(game, id, func(p *Profile, dir string) error {
		ei, err := requireEntry(p.Entries, entryKey)
		if err != nil {
			return err
		}
		e := &p.Entries[ei]
		xi := slices.Index(e.ExtraStoreKeys, extraKey)
		if xi < 0 {
			return fmt.Errorf("%q is not an extra file of this entry", extraKey)
		}
		disabled := extraDisabled(*e, extraKey)
		source := extraSource(*e, extraKey)
		e.ExtraStoreKeys = slices.Delete(e.ExtraStoreKeys, xi, xi+1)
		if xi < len(e.PreviousExtraStoreKeys) {
			e.PreviousExtraStoreKeys = slices.Delete(e.PreviousExtraStoreKeys, xi, xi+1)
		}
		entryDir := liveEntryDir(filepath.Join(dir, "mods"), e.Key)
		if err := fsx.RemoveAll(filepath.Join(entryDir, extraKey)); err != nil {
			return err
		}
		if err := s.refreshEntryMods(e, entryDir); err != nil {
			return err
		}
		was := s.NewModsEnabled
		s.NewModsEnabled = func() bool { return true }
		defer func() { s.NewModsEnabled = was }()
		_, err = s.addTo(game, p, dir, extraKey, source, disabled)
		return err
	})
	if err != nil {
		return Profile{}, err
	}
	return p, s.items.Touch(game, extraKey)
}

// CombineEntries attaches otherKey as an extra file of targetKey when both are from the same Nexus page.
func (s *Store) CombineEntries(game, id, targetKey, otherKey string) (Profile, error) {
	p, err := s.updateModsRecorded(game, id, func(p *Profile, dir string) error {
		if targetKey == "" || otherKey == "" || targetKey == otherKey {
			return fmt.Errorf("cannot combine %q with %q", targetKey, otherKey)
		}
		ti, err := requireEntry(p.Entries, targetKey)
		if err != nil {
			return err
		}
		oi, err := requireEntry(p.Entries, otherKey)
		if err != nil {
			return err
		}
		target, other := p.Entries[ti], p.Entries[oi]
		if target.IsOverlay() || other.IsOverlay() || len(overlaysOf(p.Entries, otherKey)) > 0 {
			return fmt.Errorf("optional files laid over a mod cannot be combined")
		}
		if len(other.ExtraStoreKeys) > 0 {
			return fmt.Errorf("%q has extra files of its own", otherKey)
		}
		if target.Source.Kind != KindNexus || other.Source.Kind != KindNexus || target.Source.ModID != other.Source.ModID || target.Source.ModID <= 0 {
			return fmt.Errorf("those mods are not from the same Nexus page")
		}
		if err := s.removeFrom(game, p, dir, otherKey); err != nil {
			return err
		}
		return s.addExtraLocked(game, p, dir, targetKey, otherKey, other.Source.WithFomod(other.Fomod).WithDisabled(other.Disabled))
	})
	if err != nil {
		return Profile{}, err
	}
	return p, s.items.Touch(game, otherKey)
}

func extraSource(e Entry, extraKey string) Source {
	src := Source{Kind: e.Source.Kind, Name: e.Source.Name, Picture: e.Source.Picture, EndorsementCount: e.Source.EndorsementCount}
	if modID, fileID, ok := store.NexusFile(extraKey); ok {
		src.Kind = KindNexus
		src.ModID, src.FileID = modID, fileID
	}
	return src
}

func extraDisabled(e Entry, extraKey string) []mod.ID {
	prefix := filepath.ToSlash(extraKey) + "/"
	var ids []mod.ID
	for _, m := range e.Mods {
		folder := filepath.ToSlash(m.Folder)
		if folder != extraKey && !strings.HasPrefix(folder, prefix) {
			continue
		}
		if !e.Enabled(m.ID) {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

func (s *Store) addExtraLocked(game string, p *Profile, dir, entryKey, extraKey string, source Source) error {
	ei, err := requireEntry(p.Entries, entryKey)
	if err != nil {
		return err
	}
	e := &p.Entries[ei]
	if e.IsOverlay() {
		return fmt.Errorf("%q is an optional file and holds no mod of its own", entryKey)
	}
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
		_ = fsx.RemoveAll(filepath.Join(tmp, stale))
	}
	return nil
}

func (s *Store) fillOneExtraUpdate(game, id, oldDir, tmp, oldProfKey, newKey string, e Entry) error {
	extraSrc, found, done, err := s.scanItem(game, id, newKey, nil)
	defer done()
	if err != nil {
		return err
	}
	scratch, err := os.MkdirTemp(tmp, tempPrefix)
	if err != nil {
		return err
	}
	defer func() { _ = fsx.RemoveAll(scratch) }()
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
		if !slices.ContainsFunc(found, func(m manifest.Mod) bool { return mod.Equal(m.ModID(), om.ID) }) {
			continue
		}
		rel := strings.TrimPrefix(oldFolder, oldPrefix)
		cur := filepath.Join(profRoot, filepath.FromSlash(rel))
		if !exists(cur) {
			continue
		}
		configOnly := deleteOldVersion(found, om.ID)
		err = carryOverWalk(cur, filepath.Join(extraSrc, filepath.FromSlash(rel)), filepath.Join(scratch, filepath.FromSlash(rel)), configOnly, nil)
		if err != nil {
			return err
		}
	}
	dest := filepath.Join(tmp, newKey)
	if err := fsx.RemoveAll(dest); err != nil {
		return err
	}
	return fsx.Rename(scratch, dest)
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
	if over, err := s.isOverlayItem(game, key, source); err != nil {
		return InstallResult{}, installError(err)
	} else if over {
		return s.installKey(game, id, key, source)
	}
	p, err := s.AddExtra(game, id, entryKey, key, source)
	if ask, ok := s.installQuestion(game, id, key, source, err); ok {
		return ask, nil
	}
	if err != nil {
		return InstallResult{}, installError(err)
	}
	return InstallResult{Profile: p, Added: addedNames(p, entryKey)}, nil
}

func (s *Store) copyExtraInto(game, id, entryDir, extraKey string, choices map[string]map[string][]string) error {
	src, _, done, err := s.scanItem(game, id, extraKey, choices)
	defer done()
	if err != nil {
		return err
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
		return errors.Join(err, fsx.RemoveAll(scratch))
	}
	if err := fsx.Rename(scratch, dest); err != nil {
		return errors.Join(err, fsx.RemoveAll(scratch))
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
		if slices.ContainsFunc(e.Mods, func(m Component) bool { return mod.Equal(m.ID, id) }) {
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
			if slices.ContainsFunc(prev, func(old Component) bool { return mod.Equal(old.ID, m.ID) }) {
				continue
			}
			if e.Enabled(m.ID) {
				e.Disabled = append(e.Disabled, m.ID)
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
