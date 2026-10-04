package profile

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/store"
)

const (
	historyFile      = "history.json"
	snapshotsDir     = "snapshots"
	maxHistory       = 200
	recentHistoryCap = 50
	historyAdded     = "added"
	historyRemoved   = "removed"
	historyUpdated   = "updated"
	historyEnabled   = "enabled"
	historyDisabled  = "disabled"
	historyPinned    = "pinned"
	historyImported  = "imported"
	historyRestored  = "restored"
	historyReverted  = "reverted"
	historyBulk      = "bulk"
)

// HistoryEvent is metadata for one change to a profile's mod set.
type HistoryEvent struct {
	ID         string    `json:"id"`
	At         time.Time `json:"at"`
	Kind       string    `json:"kind"`
	Label      string    `json:"label"`
	Count      int       `json:"count,omitempty"`
	Added      int       `json:"added,omitempty"`
	Removed    int       `json:"removed,omitempty"`
	Updated    int       `json:"updated,omitempty"`
	From       string    `json:"from,omitempty"`
	To         string    `json:"to,omitempty"`
	SnapshotID string    `json:"snapshotId"`
}

type historyFileData struct {
	Events    []HistoryEvent     `json:"events"`
	Snapshots map[string][]Entry `json:"-"`
	dir       string             `json:"-"`
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
	data, err := readHistory(dir)
	if err != nil {
		return nil, err
	}
	out := make([]HistoryEvent, len(data.Events))
	for i, e := range data.Events {
		ev := e
		var before []Entry
		if i > 0 {
			before, _ = snapshotEntries(&data, data.Events[i-1].SnapshotID)
		}
		if after, ok := snapshotEntries(&data, e.SnapshotID); ok {
			ev.Added, ev.Removed, ev.Updated = ModDiffCounts(before, after)
		}
		out[len(data.Events)-1-i] = ev
	}
	return out, nil
}

// RecentEvent is one history event tagged with the profile it belongs to.
type RecentEvent struct {
	ProfileID   string `json:"profileId"`
	ProfileName string `json:"profileName"`
	HistoryEvent
}

// RecentHistory lists the newest change events across this game's usable, non-hidden profiles.
func (s *Store) RecentHistory(game string) ([]RecentEvent, error) {
	return s.recentHistory(game, recentHistoryCap)
}

func (s *Store) recentHistory(game string, limit int) ([]RecentEvent, error) {
	list, err := s.List(game)
	if err != nil {
		return nil, err
	}
	var out []RecentEvent
	for _, p := range list {
		if p.Error != "" || p.Hidden {
			continue
		}
		events, err := s.History(game, p.ID)
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			out = append(out, RecentEvent{ProfileID: p.ID, ProfileName: p.Name, HistoryEvent: ev})
		}
	}
	slices.SortFunc(out, func(a, b RecentEvent) int {
		if c := b.At.Compare(a.At); c != 0 {
			return c
		}
		if a.ProfileName != b.ProfileName {
			if a.ProfileName < b.ProfileName {
				return -1
			}
			return 1
		}
		return strings.Compare(b.ID, a.ID)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	if out == nil {
		out = []RecentEvent{}
	}
	return out, nil
}

// Snapshot returns the entries held by a snapshot hash or history event ID.
func (s *Store) Snapshot(game, id, snapshotID string) ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.read(game, id); err != nil {
		return nil, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return nil, err
	}
	data, err := readHistory(dir)
	if err != nil {
		return nil, err
	}
	entries, ok := snapshotEntries(&data, snapshotID)
	if !ok {
		return nil, fmt.Errorf("history snapshot %s not found", snapshotID)
	}
	return cloneEntries(entries), nil
}

// MissingKeys are store keys a revert needs that are not in the store.
type MissingKeys struct {
	Keys  []string
	Names []string
}

func (e *MissingKeys) Error() string {
	who := e.Names
	if len(who) == 0 {
		who = e.Keys
	}
	return "missing from the store: " + strings.Join(who, ", ")
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
	data, err := readHistory(dir)
	if err != nil {
		return Profile{}, err
	}
	var target *HistoryEvent
	for i := range data.Events {
		if data.Events[i].ID == eventID {
			target = &data.Events[i]
			break
		}
	}
	if target == nil {
		return Profile{}, fmt.Errorf("history event %s not found", eventID)
	}
	snap, ok := snapshotEntries(&data, target.SnapshotID)
	if !ok {
		return Profile{}, fmt.Errorf("history snapshot %s not found", target.SnapshotID)
	}
	missing, err := s.missingStoreKeys(game, snap)
	if err != nil {
		return Profile{}, err
	}
	if len(missing) > 0 {
		return Profile{}, &MissingKeys{Keys: missing, Names: namesForStoreKeys(snap, missing)}
	}
	label := "Reverted to " + target.At.UTC().Format(time.RFC3339)
	snap = cloneEntries(snap)
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
		live := liveEntryDir(modsDir, e.Key)
		if exists(live) {
			old, tmp, err := s.layoutItem(game, p.ID, e.Key, e.Fomod)
			if tmp != "" {
				defer func() { _ = os.RemoveAll(tmp) }()
			}
			if err != nil {
				_ = os.RemoveAll(staging)
				return err
			}
			if err := carryOverWalk(live, old, liveEntryDir(staging, e.Key), false); err != nil {
				_ = os.RemoveAll(staging)
				return err
			}
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

func entryUsesKey(e Entry, want map[string]struct{}) bool {
	if _, ok := want[e.Key]; ok {
		return true
	}
	if _, ok := want[e.PreviousKey]; ok {
		return true
	}
	for _, key := range e.ExtraStoreKeys {
		if _, ok := want[key]; ok {
			return true
		}
	}
	for _, key := range e.PreviousExtraStoreKeys {
		if _, ok := want[key]; ok {
			return true
		}
	}
	return false
}

func namesForStoreKeys(entries []Entry, keys []string) []string {
	want := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		want[key] = struct{}{}
	}
	var names []string
	seen := map[string]struct{}{}
	add := func(label string) {
		if label == "" {
			return
		}
		if _, ok := seen[label]; ok {
			return
		}
		seen[label] = struct{}{}
		names = append(names, label)
	}
	used := map[string]struct{}{}
	for _, e := range entries {
		if !entryUsesKey(e, want) {
			continue
		}
		add(entryLabel(e))
		used[e.Key] = struct{}{}
		used[e.PreviousKey] = struct{}{}
		for _, key := range e.ExtraStoreKeys {
			used[key] = struct{}{}
		}
		for _, key := range e.PreviousExtraStoreKeys {
			used[key] = struct{}{}
		}
	}
	for _, key := range keys {
		if _, ok := used[key]; !ok {
			add(key)
		}
	}
	return names
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
		for _, k := range e.ExtraStoreKeys {
			if err := add(k); err != nil {
				return nil, err
			}
		}
		for _, k := range e.PreviousExtraStoreKeys {
			if err := add(k); err != nil {
				return nil, err
			}
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

type historyBatch struct {
	ID      string
	EventID string
	Before  []Entry
}

func historyBatchKey(game, id string) string {
	return game + "\x00" + id
}

// OpenHistoryBatch makes subsequent changes to this profile part of one history event.
func (s *Store) OpenHistoryBatch(game, id, batchID string) error {
	if batchID == "" {
		return errors.New("history batch id is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.read(game, id)
	if err != nil {
		return err
	}
	key := historyBatchKey(game, id)
	if batch, ok := s.historyBatches[key]; ok && batch.ID == batchID {
		return nil
	}
	if s.historyBatches == nil {
		s.historyBatches = map[string]historyBatch{}
	}
	s.historyBatches[key] = historyBatch{ID: batchID, Before: cloneEntries(p.Entries)}
	return nil
}

// RecordHistoryBatch associates a completed queued install with its bulk event.
func (s *Store) RecordHistoryBatch(game, id, batchID string) error {
	if batchID == "" {
		return s.CloseHistoryBatch(game, id)
	}
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
	data, err := readHistory(dir)
	if err != nil {
		return err
	}
	key := historyBatchKey(game, id)
	batch, ok := s.historyBatches[key]
	if !ok || batch.ID != batchID {
		batch = historyBatch{ID: batchID, Before: cloneEntries(p.Entries)}
	}
	if batch.EventID == "" && len(data.Events) > 0 {
		last := data.Events[len(data.Events)-1]
		if entries, exists := snapshotEntries(&data, last.SnapshotID); exists && entriesEqual(entries, p.Entries) {
			batch.EventID = last.ID
			if len(data.Events) > 1 {
				if previous, exists := snapshotEntries(&data, data.Events[len(data.Events)-2].SnapshotID); exists {
					batch.Before = cloneEntries(previous)
				}
			} else {
				batch.Before = []Entry{}
			}
		}
	}
	if err := s.recordHistoryBatchData(dir, &data, &batch, p.Entries); err != nil {
		return err
	}
	if s.historyBatches == nil {
		s.historyBatches = map[string]historyBatch{}
	}
	s.historyBatches[key] = batch
	return nil
}

// CloseHistoryBatch ends the current bulk event for a profile.
func (s *Store) CloseHistoryBatch(game, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.historyBatches != nil {
		delete(s.historyBatches, historyBatchKey(game, id))
	}
	return nil
}

func (s *Store) recordHistoryBatch(dir string, batch *historyBatch, after []Entry) error {
	data, err := readHistory(dir)
	if err != nil {
		return err
	}
	return s.recordHistoryBatchData(dir, &data, batch, after)
}

func (s *Store) recordHistoryBatchData(dir string, data *historyFileData, batch *historyBatch, after []Entry) error {
	if entriesEqual(batch.Before, after) {
		return nil
	}
	ev := classifyHistory(batch.Before, after)
	ev.Kind = historyBulk
	if ev.Count < 1 {
		ev.Count = 1
	}
	ev.Label = fmt.Sprintf("Changed %d mods", ev.Count)
	if batch.EventID == "" {
		created, err := appendHistory(dir, ev, after, 0)
		if err != nil {
			return err
		}
		batch.EventID = created.ID
		return nil
	}
	index := -1
	for i := range data.Events {
		if data.Events[i].ID == batch.EventID {
			index = i
			ev.ID, ev.At = data.Events[i].ID, data.Events[i].At
			break
		}
	}
	if index < 0 {
		created, err := appendHistory(dir, ev, after, 0)
		if err != nil {
			return err
		}
		batch.EventID = created.ID
		return nil
	}
	snapshot, err := entriesSnapshotID(after)
	if err != nil {
		return err
	}
	if err := writeSnapshotFile(dir, snapshot, after); err != nil {
		return err
	}
	if data.Snapshots == nil {
		data.Snapshots = map[string][]Entry{}
	}
	data.Snapshots[snapshot] = cloneEntries(after)
	ev.SnapshotID = snapshot
	data.Events[index] = ev
	return writeHistory(dir, *data, 0)
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
	_, err = appendHistory(dir, ev, p.Entries, s.historyKeep())
	return err
}

var onHistoryDecode func()

func peekHistory(dir string) (historyFileData, error) {
	data, err := readHistory(dir)
	if err != nil {
		return historyFileData{}, err
	}
	data.dir = dir
	return data, nil
}

func latestSnapshotAt(dir string) ([]Entry, bool) {
	data, err := peekHistory(dir)
	if err != nil {
		return nil, false
	}
	return latestSnapshotOf(data)
}

func readHistory(dir string) (historyFileData, error) {
	path := filepath.Join(dir, historyFile)
	b, err := fsx.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return emptyHistoryAt(dir), nil
		}
		return historyFileData{}, err
	}
	data, err := historyFromBytes(b)
	if err != nil {
		return quarantineHistory(path, err)
	}
	data.dir = dir
	return data, nil
}

func historyFromBytes(b []byte) (historyFileData, error) {
	if onHistoryDecode != nil {
		onHistoryDecode()
	}
	var data historyFileData
	if err := json.Unmarshal(b, &data); err != nil {
		return historyFileData{}, err
	}
	if data.Events == nil {
		data.Events = []HistoryEvent{}
	}
	if data.Snapshots == nil {
		data.Snapshots = map[string][]Entry{}
	}
	return data, nil
}

func emptyHistory() historyFileData {
	return emptyHistoryAt("")
}

func emptyHistoryAt(dir string) historyFileData {
	return historyFileData{Events: []HistoryEvent{}, Snapshots: map[string][]Entry{}, dir: dir}
}

func quarantineHistory(path string, cause error) (historyFileData, error) {
	corrupt := fmt.Sprintf("%s.corrupt-%d", path, time.Now().UnixNano())
	if err := os.Rename(path, corrupt); err != nil {
		return historyFileData{}, errors.Join(fmt.Errorf("read history: %w", cause), fmt.Errorf("quarantine history: %w", err))
	}
	return emptyHistory(), nil
}

func writeHistory(dir string, data historyFileData, keep int) error {
	if keep <= 0 {
		keep = maxHistory
	}
	if len(data.Events) > keep {
		data.Events = data.Events[len(data.Events)-keep:]
	}
	referenced := make(map[string]struct{}, len(data.Events))
	for _, event := range data.Events {
		referenced[event.SnapshotID] = struct{}{}
	}
	for id := range data.Snapshots {
		if _, ok := referenced[id]; !ok {
			delete(data.Snapshots, id)
		}
	}
	if err := datadir.WriteJSON(filepath.Join(dir, historyFile), historyFileData{Events: data.Events}); err != nil {
		return err
	}
	if err := pruneSnapshotFiles(dir, referenced); err != nil {
		return err
	}
	pruneHistoryFiles(dir, referenced)
	return nil
}

func snapshotEntries(data *historyFileData, id string) ([]Entry, bool) {
	if data == nil || id == "" {
		return nil, false
	}
	if entries, ok := data.Snapshots[id]; ok {
		return entries, true
	}
	hash := id
	for _, event := range data.Events {
		if event.ID == id {
			hash = event.SnapshotID
			if entries, ok := data.Snapshots[hash]; ok {
				return entries, true
			}
			break
		}
	}
	dir := data.dir
	entries, ok := readSnapshotFile(dir, hash)
	if !ok {
		return nil, false
	}
	if data.Snapshots == nil {
		data.Snapshots = map[string][]Entry{}
	}
	data.Snapshots[hash] = entries
	return entries, true
}

func snapshotFilePath(dir, id string) (string, bool) {
	if dir == "" || id == "" || len(id) > 64 {
		return "", false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", false
		}
	}
	return filepath.Join(dir, snapshotsDir, id+".json"), true
}

func readSnapshotFile(dir, id string) ([]Entry, bool) {
	path, ok := snapshotFilePath(dir, id)
	if !ok {
		return nil, false
	}
	b, err := fsx.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var entries []Entry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, false
	}
	return entries, true
}

func writeSnapshotFile(dir, id string, entries []Entry) error {
	path, ok := snapshotFilePath(dir, id)
	if !ok {
		return fmt.Errorf("history snapshot %s not writable", id)
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(path, entries)
}

func pruneSnapshotFiles(dir string, referenced map[string]struct{}) error {
	entries, err := os.ReadDir(filepath.Join(dir, snapshotsDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := strings.TrimSuffix(ent.Name(), ".json")
		if _, ok := referenced[name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(dir, snapshotsDir, ent.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func entriesSnapshotID(entries []Entry) (string, error) {
	raw, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func cloneEntries(in []Entry) []Entry {
	out := make([]Entry, len(in))
	for i, e := range in {
		e.Mods = append([]EntryMod(nil), e.Mods...)
		e.Disabled = append([]string(nil), e.Disabled...)
		e.Tags = append([]string(nil), e.Tags...)
		e.Fomod = cloneFomod(e.Fomod)
		e.ExtraStoreKeys = append([]string(nil), e.ExtraStoreKeys...)
		e.PreviousExtraStoreKeys = append([]string(nil), e.PreviousExtraStoreKeys...)
		out[i] = e
	}
	return out
}

func entriesEqual(a, b []Entry) bool {
	return reflect.DeepEqual(a, b)
}

func recordHistory(dir string, before, after []Entry, kind, label string, keep int) error {
	if entriesEqual(before, after) {
		if kind == "" {
			return nil
		}
		ev := HistoryEvent{Kind: kind, Label: label, Count: 1}
		_, err := appendHistory(dir, ev, after, keep)
		return err
	}
	ev := classifyHistory(before, after)
	if kind != "" {
		ev.Kind = kind
	}
	if label != "" {
		ev.Label = label
	}
	_, err := appendHistory(dir, ev, after, keep)
	return err
}

func appendHistory(dir string, ev HistoryEvent, after []Entry, keep int) (HistoryEvent, error) {
	ev.At = time.Now().UTC().Truncate(time.Second)
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return HistoryEvent{}, err
	}
	ev.ID = hex.EncodeToString(raw[:])
	entries := cloneEntries(after)
	snapshotID, err := entriesSnapshotID(entries)
	if err != nil {
		return HistoryEvent{}, err
	}
	ev.SnapshotID = snapshotID
	data, err := readHistory(dir)
	if err != nil {
		return HistoryEvent{}, err
	}
	if err := writeSnapshotFile(dir, snapshotID, entries); err != nil {
		return HistoryEvent{}, err
	}
	if data.Snapshots == nil {
		data.Snapshots = map[string][]Entry{}
	}
	data.Snapshots[snapshotID] = entries
	data.Events = append(data.Events, ev)
	captureHistoryConfigs(dir, snapshotID, after)
	if err := writeHistory(dir, data, keep); err != nil {
		return HistoryEvent{}, err
	}
	return ev, nil
}

// ModDiffCounts reports how many mods were added, removed, or updated between two snapshots.
func ModDiffCounts(before, after []Entry) (added, removed, updated int) {
	bMap := indexEntries(before)
	aMap := indexEntries(after)
	seen := map[string]struct{}{}
	for id, ae := range aMap {
		seen[id] = struct{}{}
		be, ok := bMap[id]
		if !ok {
			added++
			continue
		}
		if be.Key != ae.Key || entryVersion(be) != entryVersion(ae) {
			updated++
		}
	}
	for id := range bMap {
		if _, ok := seen[id]; ok {
			continue
		}
		if _, ok := aMap[id]; !ok {
			removed++
		}
	}
	return added, removed, updated
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
		case !slices.Equal(be.Disabled, ae.Disabled):
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
