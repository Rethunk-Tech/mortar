// Package savedgroups stores the library of mod groups saved from profiles, for each game.
package savedgroups

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/ids"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	modstore "github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const maxName = 60

var idPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

// Mod is one profile mod captured by a saved group.
type Mod struct {
	ID       mod.ID         `json:"id"`
	Name     string         `json:"name"`
	EntryKey string         `json:"entryKey"`
	Source   profile.Source `json:"source"`
	// Implicit is the install reason the mod's entry had, which applying the saved group gives a new entry.
	Implicit bool `json:"implicit,omitempty"`
	// OverlayOf is set on an optional file without a manifest: the entry key of the mod it is laid over, with
	// OverlayFrom and OverlayTo as recorded on the profile entry. Its id is empty.
	OverlayOf   string `json:"overlayOf,omitempty"`
	OverlayFrom string `json:"overlayFrom,omitempty"`
	OverlayTo   string `json:"overlayTo,omitempty"`
}

// SavedGroup is a named set of mods for one game, saved from a profile's group.
type SavedGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Mods []Mod  `json:"mods"`
}

// ApplyResult is the result of adding a saved group to a profile.
type ApplyResult struct {
	Profile profile.Profile `json:"profile"`
	Added   int             `json:"added"`
	Missing []string        `json:"missing"`
	// MissingMods are the mods behind Missing with their sources, so the caller can queue their downloads.
	MissingMods []Mod `json:"missingMods"`
	// Group is the name of the profile group the apply created, which differs from the saved group's name when the
	// profile already had one by that name; empty when none was created.
	Group string `json:"group"`
}

// Service exposes saved groups to the frontend.
type Service struct {
	profiles *profile.Store
	root     string
	mu       sync.Mutex
}

// NewService opens the saved-group store below the application's data directory.
func NewService(profiles *profile.Store, dataDir string) *Service {
	return &Service{profiles: profiles, root: filepath.Join(dataDir, "saved-groups")}
}

func groupName(name string) (string, error) {
	name, err := profile.CleanGroupName(name)
	if err != nil {
		return "", err
	}
	if utf8.RuneCountInString(name) > maxName {
		return "", fmt.Errorf("saved group name is longer than %d characters", maxName)
	}
	return name, nil
}

func groupID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid saved group id %q", id)
	}
	return nil
}

func (s *Service) readLocked(gameID string) ([]SavedGroup, error) {
	path, err := gamepkg.File(s.root, gameID)
	if err != nil {
		return nil, err
	}
	var groups []SavedGroup
	found, err := datadir.ReadJSON(path, &groups)
	if err != nil {
		return nil, fmt.Errorf("read saved groups for %s: %w", gameID, err)
	}
	if !found {
		return []SavedGroup{}, nil
	}
	if groups == nil {
		groups = []SavedGroup{}
	}
	for i := range groups {
		if groups[i].Mods == nil {
			groups[i].Mods = []Mod{}
		}
	}
	return groups, nil
}

func (s *Service) writeLocked(gameID string, groups []SavedGroup) error {
	path, err := gamepkg.File(s.root, gameID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(path, groups)
}

func findGroup(groups []SavedGroup, id string) (int, error) {
	if err := groupID(id); err != nil {
		return -1, err
	}
	i := slices.IndexFunc(groups, func(g SavedGroup) bool { return g.ID == id })
	if i < 0 {
		return -1, usererr.Wrap(usererr.NotFound, fmt.Errorf("saved group %q was not found", id))
	}
	return i, nil
}

func nameIndex(groups []SavedGroup, name string) int {
	return slices.IndexFunc(groups, func(g SavedGroup) bool { return strings.EqualFold(g.Name, name) })
}

// freeName rejects a name another saved group already has, apart from the group at skip (-1 for none).
func freeName(groups []SavedGroup, name string, skip int) error {
	if i := nameIndex(groups, name); i >= 0 && i != skip {
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("a saved group named %q already exists", name))
	}
	return nil
}

func newID(groups []SavedGroup) string {
	for {
		id := ids.New()
		if !slices.ContainsFunc(groups, func(g SavedGroup) bool { return g.ID == id }) {
			return id
		}
	}
}

func historyBatchID(label string) string { return label + "-" + ids.New() }

// key identifies a mod within a saved group: its id, or for an optional file without one, its entry.
func (m Mod) key() string { return cmp.Or(m.ID.Fold(), m.EntryKey) }

// profileModIDs are the ids of every mod a profile holds apart from the game's own bundled ones.
func profileModIDs(p profile.Profile) []mod.ID {
	var out []mod.ID
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		for _, m := range e.Mods {
			out = append(out, m.ID)
		}
	}
	return out
}

func snapshot(p profile.Profile, uniqueIDs []mod.ID) ([]Mod, error) {
	if len(uniqueIDs) == 0 {
		return nil, errors.New("select at least one mod")
	}
	byID := map[string]Mod{}
	for _, e := range p.Entries {
		for _, m := range e.Mods {
			id := m.ID.Fold()
			if _, exists := byID[id]; exists {
				continue
			}
			name := m.Name
			name = cmp.Or(name, m.ID.Local())
			byID[id] = Mod{ID: m.ID, Name: name, EntryKey: e.StoreKey(), Source: e.Source, Implicit: e.Implicit}
		}
	}
	out := make([]Mod, 0, len(uniqueIDs))
	seen := map[string]struct{}{}
	for _, id := range uniqueIDs {
		key := id.Fold()
		if key == "" {
			return nil, errors.New("mod id is empty")
		}
		if _, exists := seen[key]; exists {
			continue
		}
		im, ok := byID[key]
		if !ok {
			return nil, fmt.Errorf("mod %q is not in this profile", id)
		}
		seen[key] = struct{}{}
		out = append(out, im)
	}
	if len(out) == 0 {
		return nil, errors.New("select at least one mod")
	}
	return withOverlays(p, out), nil
}

// withOverlays adds the optional files laid over the chosen mods' entries, after them, so applying the saved group
// puts each main file in first.
func withOverlays(p profile.Profile, mods []Mod) []Mod {
	for _, e := range p.Entries {
		if !e.IsOverlay() || !slices.ContainsFunc(mods, func(m Mod) bool { return m.EntryKey == e.OverlayOf && m.ID != "" }) {
			continue
		}
		if slices.ContainsFunc(mods, func(m Mod) bool { return m.EntryKey == e.Key }) {
			continue
		}
		mods = append(mods, Mod{
			Name: e.Source.Name, EntryKey: e.Key, Source: e.Source, Implicit: e.Implicit,
			OverlayOf: e.OverlayOf, OverlayFrom: e.OverlayFrom, OverlayTo: e.OverlayTo,
		})
	}
	return mods
}

// modsFromProfile snapshots the named mods of a profile, or every mod of it when none are named.
func (s *Service) modsFromProfile(gameID, profileID string, uniqueIDs []mod.ID) ([]Mod, error) {
	p, err := s.profiles.Get(gameID, profileID)
	if err != nil {
		return nil, err
	}
	if len(uniqueIDs) == 0 {
		uniqueIDs = profileModIDs(p)
	}
	return snapshot(p, uniqueIDs)
}

// List returns all saved groups for a game.
func (s *Service) List(gameID string) ([]SavedGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readLocked(gameID)
}

// ReferencedStoreKeys lists the store items still needed by saved groups.
//
//wails:ignore
func (s *Service) ReferencedStoreKeys() (map[string][]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string][]string{}
	files, err := os.ReadDir(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		gameID := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		if !gamepkg.Valid(gameID) {
			continue
		}
		list, err := s.readLocked(gameID)
		if err != nil {
			return nil, err
		}
		for _, group := range list {
			for _, im := range group.Mods {
				if im.EntryKey != "" {
					out[gameID] = append(out[gameID], im.EntryKey)
				}
			}
		}
	}
	for gameID, keys := range out {
		seen := map[string]struct{}{}
		unique := make([]string, 0, len(keys))
		for _, key := range keys {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			unique = append(unique, key)
		}
		out[gameID] = unique
	}
	return out, nil
}

func (s *Service) mutate(gameID string, fn func([]SavedGroup) ([]SavedGroup, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	groups, err := s.readLocked(gameID)
	if err != nil {
		return err
	}
	groups, err = fn(groups)
	if err != nil {
		return err
	}
	return s.writeLocked(gameID, groups)
}

func (s *Service) update(gameID string, fn func([]SavedGroup) ([]SavedGroup, SavedGroup, error)) (SavedGroup, error) {
	var out SavedGroup
	err := s.mutate(gameID, func(groups []SavedGroup) ([]SavedGroup, error) {
		next, g, err := fn(groups)
		out = g
		return next, err
	})
	return out, err
}

// Create saves a group by snapshotting selected mods from a profile, or all of its mods when none are named.
func (s *Service) Create(gameID, name, profileID string, uniqueIDs []mod.ID) (SavedGroup, error) {
	name, err := groupName(name)
	if err != nil {
		return SavedGroup{}, err
	}
	return s.update(gameID, func(groups []SavedGroup) ([]SavedGroup, SavedGroup, error) {
		if err := freeName(groups, name, -1); err != nil {
			return nil, SavedGroup{}, err
		}
		mods, err := s.modsFromProfile(gameID, profileID, uniqueIDs)
		if err != nil {
			return nil, SavedGroup{}, err
		}
		g := SavedGroup{ID: newID(groups), Name: name, Mods: mods}
		return append(groups, g), g, nil
	})
}

// SaveGroup saves a profile's group to the library under name, snapshotting the mods of its entries. A saved group
// of that name is an error unless replace is set, which swaps its mods and keeps its id.
func (s *Service) SaveGroup(gameID, profileID, profileGroup, name string, replace bool) (SavedGroup, error) {
	name, err := groupName(name)
	if err != nil {
		return SavedGroup{}, err
	}
	p, err := s.profiles.Get(gameID, profileID)
	if err != nil {
		return SavedGroup{}, err
	}
	mods, err := snapshotGroup(p, profileGroup)
	if err != nil {
		return SavedGroup{}, err
	}
	return s.update(gameID, func(groups []SavedGroup) ([]SavedGroup, SavedGroup, error) {
		if i := nameIndex(groups, name); i >= 0 {
			if !replace {
				return nil, SavedGroup{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("a saved group named %q already exists", name))
			}
			groups[i].Name, groups[i].Mods = name, mods
			return groups, groups[i], nil
		}
		g := SavedGroup{ID: newID(groups), Name: name, Mods: mods}
		return append(groups, g), g, nil
	})
}

// snapshotGroup captures the mods of the entries a profile's group, and the groups inside it, hold.
func snapshotGroup(p profile.Profile, name string) ([]Mod, error) {
	var inside []profile.Group
	for _, g := range p.Groups {
		if profile.InGroup(g.Name, name) {
			inside = append(inside, g)
		}
	}
	if len(inside) == 0 {
		return nil, usererr.Wrap(usererr.NotFound, fmt.Errorf("group %q not found", name))
	}
	var modIDs []mod.ID
	for _, e := range p.Entries {
		if e.Source.Bundled() || !slices.ContainsFunc(inside, func(g profile.Group) bool { return slices.Contains(g.Keys, e.Key) }) {
			continue
		}
		for _, m := range e.Mods {
			modIDs = append(modIDs, m.ID)
		}
	}
	if len(modIDs) == 0 {
		return nil, usererr.Wrap(usererr.Invalid, fmt.Errorf("group %q has no mods to save", name))
	}
	return snapshot(p, modIDs)
}

// Rename changes a saved group's name.
func (s *Service) Rename(gameID, id, name string) (SavedGroup, error) {
	name, err := groupName(name)
	if err != nil {
		return SavedGroup{}, err
	}
	return s.update(gameID, func(groups []SavedGroup) ([]SavedGroup, SavedGroup, error) {
		i, err := findGroup(groups, id)
		if err != nil {
			return nil, SavedGroup{}, err
		}
		if err := freeName(groups, name, i); err != nil {
			return nil, SavedGroup{}, err
		}
		groups[i].Name = name
		return groups, groups[i], nil
	})
}

// Delete removes a saved group.
func (s *Service) Delete(gameID, id string) error {
	return s.mutate(gameID, func(groups []SavedGroup) ([]SavedGroup, error) {
		i, err := findGroup(groups, id)
		if err != nil {
			return nil, err
		}
		return slices.Delete(groups, i, i+1), nil
	})
}

// AddMods snapshots selected mods from a profile into a saved group, or all of its mods when none are named.
func (s *Service) AddMods(gameID, id, profileID string, uniqueIDs []mod.ID) (SavedGroup, error) {
	return s.update(gameID, func(groups []SavedGroup) ([]SavedGroup, SavedGroup, error) {
		i, err := findGroup(groups, id)
		if err != nil {
			return nil, SavedGroup{}, err
		}
		mods, err := s.modsFromProfile(gameID, profileID, uniqueIDs)
		if err != nil {
			return nil, SavedGroup{}, err
		}
		known := make(map[string]struct{}, len(groups[i].Mods))
		for _, im := range groups[i].Mods {
			known[im.key()] = struct{}{}
		}
		for _, im := range mods {
			if _, exists := known[im.key()]; exists {
				continue
			}
			groups[i].Mods = append(groups[i].Mods, im)
			known[im.key()] = struct{}{}
		}
		return groups, groups[i], nil
	})
}

// RemoveMods drops the mods with the given ids from a saved group, and the optional files laid over them. A saved
// group keeps at least one mod: delete it to empty it.
func (s *Service) RemoveMods(gameID, id string, uniqueIDs []mod.ID) (SavedGroup, error) {
	return s.update(gameID, func(groups []SavedGroup) ([]SavedGroup, SavedGroup, error) {
		i, err := findGroup(groups, id)
		if err != nil {
			return nil, SavedGroup{}, err
		}
		drop := make(map[string]struct{}, len(uniqueIDs))
		for _, id := range uniqueIDs {
			drop[id.Fold()] = struct{}{}
		}
		kept := slices.DeleteFunc(slices.Clone(groups[i].Mods), func(m Mod) bool {
			_, gone := drop[m.ID.Fold()]
			return m.OverlayOf == "" && gone
		})
		mains := map[string]struct{}{}
		for _, m := range kept {
			if m.OverlayOf == "" {
				mains[m.EntryKey] = struct{}{}
			}
		}
		kept = slices.DeleteFunc(kept, func(m Mod) bool {
			_, held := mains[m.OverlayOf]
			return m.OverlayOf != "" && !held
		})
		if len(kept) == 0 {
			return nil, SavedGroup{}, errors.New("a saved group needs at least one mod; delete it instead")
		}
		groups[i].Mods = kept
		return groups, groups[i], nil
	})
}

type entryMods struct {
	key    string
	source profile.Source
	mods   []Mod
}

// Apply adds a saved group's store entries to a profile without downloading anything, and creates a group of the
// same name in the profile holding every one of its mods the profile now has.
func (s *Service) Apply(gameID, id, profileID string) (ApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	groups, err := s.readLocked(gameID)
	if err != nil {
		return ApplyResult{}, err
	}
	i, err := findGroup(groups, id)
	if err != nil {
		return ApplyResult{}, err
	}
	return s.applyLocked(gameID, id, groups[i].Name, groups[i].Mods, profileID)
}

// ApplyMods adds the store entries behind mods to a profile as Apply does, without creating a group; label names the
// history batch.
//
//wails:ignore
func (s *Service) ApplyMods(gameID, label string, mods []Mod, profileID string) (ApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyLocked(gameID, label, "", mods, profileID)
}

// applyLocked adds mods' entries to the profile in one history batch and, when group is not empty, creates it there.
func (s *Service) applyLocked(gameID, label, group string, mods []Mod, profileID string) (ApplyResult, error) {
	current, err := s.profiles.Get(gameID, profileID)
	if err != nil {
		return ApplyResult{}, err
	}
	if s.profiles.Running != nil && s.profiles.Running(gameID, profileID) {
		return ApplyResult{}, &profile.RunningError{Game: gameID}
	}
	batchID := historyBatchID(label)
	if err := s.profiles.OpenHistoryBatch(gameID, profileID, batchID); err != nil {
		return ApplyResult{}, err
	}
	defer func() { _ = s.profiles.CloseHistoryBatch(gameID, profileID) }()
	byEntry := make([]entryMods, 0, len(mods))
	entryAt := map[string]int{}
	for _, im := range mods {
		at, ok := entryAt[im.EntryKey]
		if !ok {
			entryAt[im.EntryKey] = len(byEntry)
			src := im.Source.WithImplicit(im.Implicit)
			if im.OverlayOf != "" {
				src = src.WithOverlay(im.OverlayFrom, im.OverlayTo)
			}
			byEntry = append(byEntry, entryMods{key: im.EntryKey, source: src, mods: []Mod{im}})
			continue
		}
		byEntry[at].mods = append(byEntry[at].mods, im)
	}
	result := ApplyResult{Profile: current, Missing: []string{}, MissingMods: []Mod{}}
	held := map[string]struct{}{}
	for _, entry := range byEntry {
		next, err := s.profiles.AddEntry(gameID, profileID, entry.key, entry.source)
		if err == nil {
			result.Profile = next
			result.Added += len(entry.mods)
			held[entry.key] = struct{}{}
			continue
		}
		if _, ok := errors.AsType[*profile.DuplicateError](err); ok {
			held[entry.key] = struct{}{}
			continue
		}
		if errors.Is(err, modstore.ErrNotFound) {
			for _, im := range entry.mods {
				result.Missing = append(result.Missing, im.Name)
				result.MissingMods = append(result.MissingMods, im)
			}
			continue
		}
		return ApplyResult{}, err
	}
	if group == "" {
		return result, nil
	}
	var keys []string
	for _, e := range result.Profile.Entries {
		if _, ok := held[e.StoreKey()]; ok {
			keys = append(keys, e.Key)
		}
	}
	if len(keys) == 0 {
		return result, nil
	}
	if result.Added == 0 {
		// Nothing changed in the batch, so the group would join no history event; it gets its own.
		_ = s.profiles.CloseHistoryBatch(gameID, profileID)
	}
	next, name, err := s.profiles.AddGroup(gameID, profileID, group, keys)
	if err != nil {
		return ApplyResult{}, err
	}
	result.Profile, result.Group = next, name
	return result, nil
}
