// Package bundles stores named sets of mods for each game.
package bundles

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	gamepkg "github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/profile"
	modstore "github.com/Rethunk-AI/mortar/internal/store"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

const maxName = 60

var idPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

// Mod is one profile mod captured by a bundle.
type Mod struct {
	UniqueID string         `json:"uniqueId"`
	Name     string         `json:"name"`
	EntryKey string         `json:"entryKey"`
	Source   profile.Source `json:"source"`
}

// Bundle is a named set of mods for one game.
type Bundle struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Mods []Mod  `json:"mods"`
}

// ApplyResult is the result of adding a bundle to a profile.
type ApplyResult struct {
	Profile profile.Profile `json:"profile"`
	Added   int             `json:"added"`
	Missing []string        `json:"missing"`
}

// Service exposes bundles to the frontend.
type Service struct {
	profiles *profile.Store
	root     string
	mu       sync.Mutex
}

// NewService opens the bundle store below the application's data directory.
func NewService(profiles *profile.Store, dataDir string) *Service {
	return &Service{profiles: profiles, root: filepath.Join(dataDir, "bundles")}
}

func (s *Service) file(gameID string) (string, error) {
	if !gamepkg.Valid(gameID) {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", gameID))
	}
	return filepath.Join(s.root, gameID+".json"), nil
}

func bundleName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("bundle name is empty")
	}
	if utf8.RuneCountInString(name) > maxName {
		return "", fmt.Errorf("bundle name is longer than %d characters", maxName)
	}
	return name, nil
}

func bundleID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid bundle id %q", id)
	}
	return nil
}

func (s *Service) readLocked(gameID string) ([]Bundle, error) {
	path, err := s.file(gameID)
	if err != nil {
		return nil, err
	}
	var bundles []Bundle
	found, err := datadir.ReadJSON(path, &bundles)
	if err != nil {
		return nil, fmt.Errorf("read bundles for %s: %w", gameID, err)
	}
	if !found {
		return []Bundle{}, nil
	}
	if bundles == nil {
		bundles = []Bundle{}
	}
	for i := range bundles {
		if bundles[i].Mods == nil {
			bundles[i].Mods = []Mod{}
		}
	}
	return bundles, nil
}

func (s *Service) writeLocked(gameID string, bundles []Bundle) error {
	path, err := s.file(gameID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(path, bundles)
}

func findBundle(bundles []Bundle, id string) (int, error) {
	if err := bundleID(id); err != nil {
		return -1, err
	}
	i := slices.IndexFunc(bundles, func(b Bundle) bool { return b.ID == id })
	if i < 0 {
		return -1, usererr.Wrap(usererr.NotFound, fmt.Errorf("bundle %q was not found", id))
	}
	return i, nil
}

func newID(bundles []Bundle) (string, error) {
	used := make(map[string]struct{}, len(bundles))
	for _, b := range bundles {
		used[b.ID] = struct{}{}
	}
	for {
		var raw [8]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return "", err
		}
		id := hex.EncodeToString(raw[:])
		if _, ok := used[id]; !ok {
			return id, nil
		}
	}
}

func historyBatchID(bundleID string) string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return bundleID + "-apply"
	}
	return bundleID + "-" + hex.EncodeToString(raw[:])
}

func profileFor(profiles []profile.Profile, id string) (profile.Profile, error) {
	for _, p := range profiles {
		if p.Error == "" && p.ID == id {
			return p, nil
		}
	}
	return profile.Profile{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("profile %q was not found", id))
}

func snapshot(p profile.Profile, uniqueIDs []string) ([]Mod, error) {
	if len(uniqueIDs) == 0 {
		return nil, errors.New("select at least one mod")
	}
	byID := map[string]Mod{}
	for _, e := range p.Entries {
		for _, m := range e.Mods {
			id := strings.ToLower(m.UniqueID)
			if _, exists := byID[id]; exists {
				continue
			}
			name := m.Name
			if name == "" {
				name = m.UniqueID
			}
			byID[id] = Mod{UniqueID: m.UniqueID, Name: name, EntryKey: e.Key, Source: e.Source}
		}
	}
	out := make([]Mod, 0, len(uniqueIDs))
	seen := map[string]struct{}{}
	for _, id := range uniqueIDs {
		key := strings.ToLower(strings.TrimSpace(id))
		if key == "" {
			return nil, errors.New("mod id is empty")
		}
		if _, exists := seen[key]; exists {
			continue
		}
		mod, ok := byID[key]
		if !ok {
			return nil, fmt.Errorf("mod %q is not in this profile", id)
		}
		seen[key] = struct{}{}
		out = append(out, mod)
	}
	if len(out) == 0 {
		return nil, errors.New("select at least one mod")
	}
	return out, nil
}

func (s *Service) modsFromProfile(gameID, profileID string, uniqueIDs []string) ([]Mod, error) {
	profiles, err := s.profiles.List(gameID)
	if err != nil {
		return nil, err
	}
	p, err := profileFor(profiles, profileID)
	if err != nil {
		return nil, err
	}
	return snapshot(p, uniqueIDs)
}

// List returns all bundles for a game.
func (s *Service) List(gameID string) ([]Bundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readLocked(gameID)
}

// ReferencedStoreKeys lists the store items still needed by saved bundles.
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
		for _, bundle := range list {
			for _, mod := range bundle.Mods {
				if mod.EntryKey != "" {
					out[gameID] = append(out[gameID], mod.EntryKey)
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

func (s *Service) mutate(gameID string, fn func([]Bundle) ([]Bundle, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bundles, err := s.readLocked(gameID)
	if err != nil {
		return err
	}
	bundles, err = fn(bundles)
	if err != nil {
		return err
	}
	return s.writeLocked(gameID, bundles)
}

func (s *Service) update(gameID string, fn func([]Bundle) ([]Bundle, Bundle, error)) (Bundle, error) {
	var out Bundle
	err := s.mutate(gameID, func(bundles []Bundle) ([]Bundle, error) {
		next, b, err := fn(bundles)
		out = b
		return next, err
	})
	return out, err
}

// Create makes a bundle by snapshotting selected mods from a profile.
func (s *Service) Create(gameID, name, profileID string, uniqueIDs []string) (Bundle, error) {
	name, err := bundleName(name)
	if err != nil {
		return Bundle{}, err
	}
	return s.update(gameID, func(bundles []Bundle) ([]Bundle, Bundle, error) {
		mods, err := s.modsFromProfile(gameID, profileID, uniqueIDs)
		if err != nil {
			return nil, Bundle{}, err
		}
		id, err := newID(bundles)
		if err != nil {
			return nil, Bundle{}, err
		}
		b := Bundle{ID: id, Name: name, Mods: mods}
		return append(bundles, b), b, nil
	})
}

// Rename changes a bundle's name.
func (s *Service) Rename(gameID, id, name string) (Bundle, error) {
	name, err := bundleName(name)
	if err != nil {
		return Bundle{}, err
	}
	return s.update(gameID, func(bundles []Bundle) ([]Bundle, Bundle, error) {
		i, err := findBundle(bundles, id)
		if err != nil {
			return nil, Bundle{}, err
		}
		bundles[i].Name = name
		return bundles, bundles[i], nil
	})
}

// Delete removes a bundle.
func (s *Service) Delete(gameID, id string) error {
	return s.mutate(gameID, func(bundles []Bundle) ([]Bundle, error) {
		i, err := findBundle(bundles, id)
		if err != nil {
			return nil, err
		}
		return slices.Delete(bundles, i, i+1), nil
	})
}

// AddMods snapshots selected mods from a profile into a bundle.
func (s *Service) AddMods(gameID, bundleID, profileID string, uniqueIDs []string) (Bundle, error) {
	return s.update(gameID, func(bundles []Bundle) ([]Bundle, Bundle, error) {
		i, err := findBundle(bundles, bundleID)
		if err != nil {
			return nil, Bundle{}, err
		}
		mods, err := s.modsFromProfile(gameID, profileID, uniqueIDs)
		if err != nil {
			return nil, Bundle{}, err
		}
		known := make(map[string]struct{}, len(bundles[i].Mods))
		for _, mod := range bundles[i].Mods {
			known[strings.ToLower(mod.UniqueID)] = struct{}{}
		}
		for _, mod := range mods {
			if _, exists := known[strings.ToLower(mod.UniqueID)]; exists {
				continue
			}
			bundles[i].Mods = append(bundles[i].Mods, mod)
			known[strings.ToLower(mod.UniqueID)] = struct{}{}
		}
		return bundles, bundles[i], nil
	})
}

// RemoveMods removes selected unique IDs from a bundle.
func (s *Service) RemoveMods(gameID, bundleID string, uniqueIDs []string) (Bundle, error) {
	return s.update(gameID, func(bundles []Bundle) ([]Bundle, Bundle, error) {
		i, err := findBundle(bundles, bundleID)
		if err != nil {
			return nil, Bundle{}, err
		}
		remove := make(map[string]struct{}, len(uniqueIDs))
		for _, id := range uniqueIDs {
			remove[strings.ToLower(strings.TrimSpace(id))] = struct{}{}
		}
		bundles[i].Mods = slices.DeleteFunc(bundles[i].Mods, func(mod Mod) bool {
			_, ok := remove[strings.ToLower(mod.UniqueID)]
			return ok
		})
		return bundles, bundles[i], nil
	})
}

type entryMods struct {
	key    string
	source profile.Source
	mods   []Mod
}

// Apply adds a bundle's store entries to a profile without downloading anything.
func (s *Service) Apply(gameID, bundleID, profileID string) (ApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bundles, err := s.readLocked(gameID)
	if err != nil {
		return ApplyResult{}, err
	}
	i, err := findBundle(bundles, bundleID)
	if err != nil {
		return ApplyResult{}, err
	}
	profiles, err := s.profiles.List(gameID)
	if err != nil {
		return ApplyResult{}, err
	}
	current, err := profileFor(profiles, profileID)
	if err != nil {
		return ApplyResult{}, err
	}
	if s.profiles.Running != nil && s.profiles.Running(gameID, profileID) {
		return ApplyResult{}, &profile.RunningError{Game: gameID}
	}
	batchID := historyBatchID(bundleID)
	if err := s.profiles.OpenHistoryBatch(gameID, profileID, batchID); err != nil {
		return ApplyResult{}, err
	}
	defer func() { _ = s.profiles.CloseHistoryBatch(gameID, profileID) }()
	groups := make([]entryMods, 0, len(bundles[i].Mods))
	groupAt := map[string]int{}
	for _, mod := range bundles[i].Mods {
		at, ok := groupAt[mod.EntryKey]
		if !ok {
			groupAt[mod.EntryKey] = len(groups)
			groups = append(groups, entryMods{key: mod.EntryKey, source: mod.Source, mods: []Mod{mod}})
			continue
		}
		groups[at].mods = append(groups[at].mods, mod)
	}
	result := ApplyResult{Profile: current, Missing: []string{}}
	for _, group := range groups {
		next, err := s.profiles.AddEntry(gameID, profileID, group.key, group.source)
		if err == nil {
			result.Profile = next
			result.Added += len(group.mods)
			continue
		}
		if _, ok := errors.AsType[*profile.DuplicateError](err); ok {
			continue
		}
		if errors.Is(err, modstore.ErrNotFound) {
			for _, mod := range group.mods {
				result.Missing = append(result.Missing, mod.Name)
			}
			continue
		}
		return ApplyResult{}, err
	}
	return result, nil
}
