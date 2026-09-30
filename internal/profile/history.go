package profile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/store"
)

const (
	historyFile     = "history.json"
	maxHistory      = 200
	historyAdded    = "added"
	historyRemoved  = "removed"
	historyUpdated  = "updated"
	historyEnabled  = "enabled"
	historyDisabled = "disabled"
	historyPinned   = "pinned"
	historyImported = "imported"
	historyRestored = "restored"
	historyReverted = "reverted"
	historyBulk     = "bulk"
)

// HistoryEvent is one change to a profile's mod set, with the entry list after that change.
type HistoryEvent struct {
	ID      string    `json:"id"`
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Label   string    `json:"label"`
	Count   int       `json:"count,omitempty"`
	From    string    `json:"from,omitempty"`
	To      string    `json:"to,omitempty"`
	Entries []Entry   `json:"entries"`
}

type historyFileData struct {
	Events []HistoryEvent `json:"events"`
}

// History returns this profile's change events, newest first.
func (s *Store) History(game, id string) ([]HistoryEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.read(game, id); err != nil {
		return nil, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return nil, err
	}
	events, err := readHistory(dir)
	if err != nil {
		return nil, err
	}
	out := make([]HistoryEvent, len(events))
	for i, e := range events {
		out[len(events)-1-i] = e
	}
	return out, nil
}

// MissingKeys are store keys a revert needs that are not in the store.
type MissingKeys struct {
	Keys []string
}

func (e *MissingKeys) Error() string {
	return "missing from the store: " + strings.Join(e.Keys, ", ")
}

// Revert restores the profile's entries to the snapshot stored with eventID.
func (s *Store) Revert(game, id, eventID string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	events, err := readHistory(dir)
	if err != nil {
		return Profile{}, err
	}
	var target *HistoryEvent
	for i := range events {
		if events[i].ID == eventID {
			target = &events[i]
			break
		}
	}
	if target == nil {
		return Profile{}, fmt.Errorf("history event %s not found", eventID)
	}
	missing, err := s.missingStoreKeys(game, target.Entries)
	if err != nil {
		return Profile{}, err
	}
	if len(missing) > 0 {
		return Profile{}, &MissingKeys{Keys: missing}
	}
	label := "Reverted to " + target.At.UTC().Format(time.RFC3339)
	snap := cloneEntries(target.Entries)
	return s.updateLockedAs(game, id, historyReverted, label, func(p *Profile, profDir string) error {
		return s.applyEntrySnapshot(game, p, profDir, snap)
	})
}

func (s *Store) applyEntrySnapshot(game string, p *Profile, dir string, entries []Entry) error {
	modsDir := filepath.Join(dir, "mods")
	staging := modsDir + ".new"
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return err
	}
	p.Entries = []Entry{}
	keys := make([]string, 0, len(entries)*2)
	for _, e := range entries {
		if err := s.place(game, staging, e); err != nil {
			_ = os.RemoveAll(staging)
			return err
		}
		p.Entries = append(p.Entries, e)
		keys = append(keys, e.Key)
		if e.PreviousKey != "" {
			keys = append(keys, e.PreviousKey)
		}
	}
	old := modsDir + ".old"
	_ = os.RemoveAll(old)
	if _, err := os.Stat(modsDir); err == nil {
		if err := os.Rename(modsDir, old); err != nil {
			_ = os.RemoveAll(staging)
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := os.Rename(staging, modsDir); err != nil {
		if _, statErr := os.Stat(old); statErr == nil {
			_ = os.Rename(old, modsDir)
		}
		_ = os.RemoveAll(staging)
		return err
	}
	if s.items != nil && len(keys) > 0 {
		if err := s.items.Touch(game, keys...); err != nil {
			restoreModsOld(dir)
			return err
		}
	}
	return nil
}

func restoreModsOld(dir string) {
	modsDir := filepath.Join(dir, "mods")
	old := modsDir + ".old"
	if _, err := os.Stat(old); err != nil {
		return
	}
	_ = os.RemoveAll(modsDir)
	_ = os.Rename(old, modsDir)
}

func (s *Store) missingStoreKeys(game string, entries []Entry) ([]string, error) {
	seen := map[string]struct{}{}
	var missing []string
	add := func(key string) error {
		if key == "" {
			return nil
		}
		if _, ok := seen[key]; ok {
			return nil
		}
		seen[key] = struct{}{}
		if s.items == nil {
			missing = append(missing, key)
			return nil
		}
		if _, err := s.items.Path(game, key); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				missing = append(missing, key)
				return nil
			}
			return err
		}
		return nil
	}
	for _, e := range entries {
		if err := add(e.Key); err != nil {
			return nil, err
		}
		if err := add(e.PreviousKey); err != nil {
			return nil, err
		}
	}
	return missing, nil
}

func (s *Store) updateLockedAs(game, id, kind, label string, fn func(p *Profile, dir string) error) (Profile, error) {
	s.historyKind = kind
	s.historyLabel = label
	return s.updateLocked(game, id, fn)
}

func (s *Store) setHistoryQuiet(id string, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.historyQuietIDs == nil {
		s.historyQuietIDs = map[string]int{}
	}
	if on {
		s.historyQuietIDs[id]++
		return
	}
	s.historyQuietIDs[id]--
	if s.historyQuietIDs[id] <= 0 {
		delete(s.historyQuietIDs, id)
	}
}

func (s *Store) recordSnapshot(game, id, kind, label string, count int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.read(game, id)
	if err != nil {
		return err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return err
	}
	ev := HistoryEvent{Kind: kind, Label: label, Count: count}
	return appendHistory(dir, ev, p.Entries)
}

func readHistory(dir string) ([]HistoryEvent, error) {
	b, err := fsx.ReadFile(filepath.Join(dir, historyFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []HistoryEvent{}, nil
		}
		return nil, err
	}
	var data historyFileData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}
	if data.Events == nil {
		return []HistoryEvent{}, nil
	}
	return data.Events, nil
}

func writeHistory(dir string, events []HistoryEvent) error {
	if len(events) > maxHistory {
		events = events[len(events)-maxHistory:]
	}
	return datadir.WriteJSON(filepath.Join(dir, historyFile), historyFileData{Events: events})
}

func cloneEntries(in []Entry) []Entry {
	out := make([]Entry, len(in))
	for i, e := range in {
		e.Mods = append([]EntryMod(nil), e.Mods...)
		e.Disabled = append([]string(nil), e.Disabled...)
		e.Tags = append([]string(nil), e.Tags...)
		out[i] = e
	}
	return out
}

func entriesEqual(a, b []Entry) bool {
	return reflect.DeepEqual(a, b)
}

func recordHistory(dir string, before, after []Entry, kind, label string) error {
	if entriesEqual(before, after) {
		return nil
	}
	ev := classifyHistory(before, after)
	if kind != "" {
		ev.Kind = kind
	}
	if label != "" {
		ev.Label = label
	}
	return appendHistory(dir, ev, after)
}

func appendHistory(dir string, ev HistoryEvent, after []Entry) error {
	ev.At = time.Now().UTC().Truncate(time.Second)
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return err
	}
	ev.ID = hex.EncodeToString(raw[:])
	ev.Entries = cloneEntries(after)
	events, err := readHistory(dir)
	if err != nil {
		return err
	}
	return writeHistory(dir, append(events, ev))
}

func classifyHistory(before, after []Entry) HistoryEvent {
	bMap := indexEntries(before)
	aMap := indexEntries(after)
	var added, removed, updated, enabled, disabled, pinned int
	var from, to, name, pinLabel string
	seen := map[string]struct{}{}
	for id, ae := range aMap {
		seen[id] = struct{}{}
		be, ok := bMap[id]
		if !ok {
			added++
			name = entryName(ae)
			continue
		}
		switch {
		case be.Key != ae.Key || entryVersion(be) != entryVersion(ae):
			updated++
			from = entryVersion(be)
			to = entryVersion(ae)
			name = entryName(ae)
		case !equalStrings(be.Disabled, ae.Disabled):
			if len(ae.Disabled) > len(be.Disabled) {
				disabled++
			} else {
				enabled++
			}
			name = entryName(ae)
		case be.Pinned != ae.Pinned:
			pinned++
			name = entryName(ae)
			if ae.Pinned {
				pinLabel = "Pinned " + name
			} else {
				pinLabel = "Unpinned " + name
			}
		}
	}
	for id := range bMap {
		if _, ok := seen[id]; ok {
			continue
		}
		if _, ok := aMap[id]; !ok {
			removed++
			name = entryName(bMap[id])
		}
	}
	count := added + removed + updated + enabled + disabled + pinned
	switch {
	case count > 1:
		return HistoryEvent{Kind: historyBulk, Label: fmt.Sprintf("Changed %d mods", count), Count: count}
	case added == 1:
		return HistoryEvent{Kind: historyAdded, Label: "Added " + name, Count: 1}
	case removed == 1:
		return HistoryEvent{Kind: historyRemoved, Label: "Removed " + name, Count: 1}
	case updated == 1:
		return HistoryEvent{Kind: historyUpdated, Label: fmt.Sprintf("Updated %s from %s to %s", name, from, to), Count: 1, From: from, To: to}
	case enabled == 1:
		return HistoryEvent{Kind: historyEnabled, Label: "Enabled " + name, Count: 1}
	case disabled == 1:
		return HistoryEvent{Kind: historyDisabled, Label: "Disabled " + name, Count: 1}
	case pinned == 1:
		return HistoryEvent{Kind: historyPinned, Label: pinLabel, Count: 1}
	default:
		return HistoryEvent{Kind: historyBulk, Label: "Changed mods", Count: 1}
	}
}

func indexEntries(es []Entry) map[string]Entry {
	m := make(map[string]Entry, len(es))
	for _, e := range es {
		m[entryIdentity(e)] = e
	}
	return m
}

func entryIdentity(e Entry) string {
	if len(e.Mods) == 0 {
		return e.Key
	}
	return strings.ToLower(e.Mods[0].UniqueID)
}

func entryName(e Entry) string {
	if len(e.Mods) > 0 && e.Mods[0].Name != "" {
		return e.Mods[0].Name
	}
	if len(e.Mods) > 0 {
		return e.Mods[0].UniqueID
	}
	return e.Key
}

func entryVersion(e Entry) string {
	if len(e.Mods) > 0 {
		return e.Mods[0].Version
	}
	return e.Key
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
