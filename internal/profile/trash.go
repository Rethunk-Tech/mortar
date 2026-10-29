package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// trashRetention is how long a deleted profile stays restorable when no setting is present.
const trashRetention = 30 * 24 * time.Hour

func (s *Store) trashKeep() time.Duration {
	if s.settings != nil {
		return s.settings.Get().TrashKeepFor()
	}
	return trashRetention
}

// TrashItem is a deleted profile that can still be restored.
type TrashItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	DeletedAt time.Time `json:"deletedAt"`
	DaysLeft  int       `json:"daysLeft"`
}

func (s *Store) trashDir(gameID, id string) (string, error) {
	if !game.Valid(gameID) {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", gameID))
	}
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("invalid profile id %q", id)
	}
	return filepath.Join(s.trash, gameID, id), nil
}

// Delete moves the profile folder to <datadir>/trash/<game>/<id>/ and stamps it with the deletion time.
func (s *Store) Delete(gameID, id string) error {
	unlock, err := s.lockLayout(gameID, id, true)
	if err != nil {
		return err
	}
	defer unlock()
	src, err := s.profileDir(gameID, id)
	if err != nil {
		return err
	}
	// A profile in another folder comes home first, so the trash holds it whole and a restore needs no drive.
	if err := s.bringHome(gameID, id, src); err != nil {
		return err
	}
	var tray []TrayFile
	if p, err := s.read(gameID, id); err == nil {
		for _, e := range p.Entries {
			tray = append(tray, e.TrayFiles...)
		}
	}
	dst, _ := s.trashDir(gameID, id)
	forgetProfile(filepath.Join(src, fileName))
	if _, err := os.Stat(src); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	if err := safefs.Rename(src, dst); err != nil {
		return err
	}
	s.listChanged(gameID)
	now := time.Now()
	if err := os.Chtimes(dst, now, now); err != nil {
		return err
	}
	s.releaseTray(gameID, &Profile{ID: id}, tray)
	// The profile is already in the trash; a shortcut left behind is logged, not a failed delete.
	if s.ShortcutRemoved != nil {
		if err := s.ShortcutRemoved(gameID, id); err != nil {
			log.Printf("profile %s/%s: remove its shortcuts: %v", gameID, id, err)
		}
	}
	return nil
}

func (s *Store) trashed(gameID string) ([]TrashItem, error) {
	dir := filepath.Join(s.trash, gameID)
	dirs, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []TrashItem{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []TrashItem{}
	for _, d := range dirs {
		if !d.IsDir() || !idPattern.MatchString(d.Name()) {
			continue
		}
		// One unreadable trashed profile must not hide or keep the others; StoreKeys still fails on it, so its
		// store items are never collected.
		info, err := d.Info()
		if err != nil {
			log.Printf("trashed profile %s/%s skipped: %v", gameID, d.Name(), err)
			continue
		}
		p, err := readAt(filepath.Join(dir, d.Name()), d.Name())
		if err != nil {
			log.Printf("trashed profile %s/%s skipped: %v", gameID, d.Name(), err)
			continue
		}
		out = append(out, TrashItem{ID: p.ID, Name: p.Name, DeletedAt: info.ModTime().UTC()})
	}
	return out, nil
}

// ListTrash returns the game's deleted profiles, newest deletion first, with the days left before they are purged.
func (s *Store) ListTrash(gameID string) ([]TrashItem, error) {
	if !game.Valid(gameID) {
		return nil, usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", gameID))
	}
	items, err := s.trashed(gameID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range items {
		left := s.trashKeep() - now.Sub(items[i].DeletedAt)
		items[i].DaysLeft = max(0, int((left+24*time.Hour-1)/(24*time.Hour)))
	}
	slices.SortFunc(items, func(a, b TrashItem) int { return b.DeletedAt.Compare(a.DeletedAt) })
	return items, nil
}

// Restore moves a deleted profile back. It fails when a profile with that id exists again.
func (s *Store) Restore(gameID, id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, err := s.trashDir(gameID, id)
	if err != nil {
		return Profile{}, err
	}
	dst, _ := s.profileDir(gameID, id)
	if exists(dst) {
		return Profile{}, fmt.Errorf("profile %s already exists", id)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return Profile{}, err
	}
	if err := safefs.Rename(src, dst); err != nil {
		return Profile{}, err
	}
	s.listChanged(gameID)
	return s.replaceTrayLocked(gameID, id)
}

// Purge permanently removes one trashed profile.
func (s *Store) Purge(gameID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir, err := s.trashDir(gameID, id)
	if err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to purge symlink %q", id)
	}
	return safefs.RemoveAll(dir)
}

func (s *Store) purgeTrash(gameID string) error {
	if !game.Valid(gameID) {
		return usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", gameID))
	}
	dir := filepath.Join(s.trash, gameID)
	items, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, item := range items {
		if !item.IsDir() || !idPattern.MatchString(item.Name()) {
			continue
		}
		info, err := os.Lstat(filepath.Join(dir, item.Name()))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			errs = append(errs, fmt.Errorf("refusing to purge symlink %q", item.Name()))
			continue
		}
		errs = append(errs, safefs.RemoveAll(filepath.Join(dir, item.Name())))
	}
	return errors.Join(errs...)
}

// PurgeTrash permanently removes all trashed profiles for a game. A time.Time
// argument is retained for the startup expiry sweep.
func (s *Store) PurgeTrash(target any) error {
	if gameID, ok := target.(string); ok {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.purgeTrash(gameID)
	}
	now, ok := target.(time.Time)
	if !ok {
		return fmt.Errorf("invalid trash purge target")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	games, err := os.ReadDir(s.trash)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, g := range games {
		if !g.IsDir() || !game.Valid(g.Name()) {
			continue
		}
		items, err := s.trashed(g.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, it := range items {
			if now.Sub(it.DeletedAt) > s.trashKeep() {
				errs = append(errs, safefs.RemoveAll(filepath.Join(s.trash, g.Name(), it.ID)))
			}
		}
	}
	return errors.Join(errs...)
}

// StoreKeys lists, per game, every store key that a profile or trashed profile names. With history, keys that only
// older history snapshots name are included too; collection that never deletes can skip them.
func (s *Store) StoreKeys(history bool) (map[string][]string, error) {
	out := map[string][]string{}
	games := map[string]bool{}
	for _, root := range []string{s.root, s.trash} {
		dirs, err := os.ReadDir(root)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		for _, d := range dirs {
			if d.IsDir() && game.Valid(d.Name()) {
				games[d.Name()] = true
			}
		}
	}
	for g := range games {
		profiles, err := s.List(g)
		if err != nil {
			return nil, err
		}
		// A damaged profile's keys are unknown, so a collection run from this list would delete what it still names.
		if bad := len(profiles) - len(skipDamaged(profiles)); bad > 0 {
			return nil, fmt.Errorf("%d %s profile(s) are damaged: repair or delete them before collecting the store", bad, g)
		}
		dirs, err := os.ReadDir(filepath.Join(s.trash, g))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		for _, d := range dirs {
			if !d.IsDir() || !idPattern.MatchString(d.Name()) {
				continue
			}
			p, err := readAt(filepath.Join(s.trash, g, d.Name()), d.Name())
			if err != nil {
				return nil, err
			}
			profiles = append(profiles, p)
		}
		for _, p := range profiles {
			out[g] = append(out[g], entriesStoreKeys(p.Entries)...)
			dir := filepath.Join(s.root, g, p.ID)
			if _, err := os.Stat(filepath.Join(dir, fileName)); err != nil {
				dir = filepath.Join(s.trash, g, p.ID)
			}
			key, err := loaderStoreKey(g, p, dir)
			if err != nil {
				return nil, err
			}
			out[g] = append(out[g], key)
			if !history {
				continue
			}
			keys, err := historyStoreKeys(dir)
			if err != nil {
				return nil, err
			}
			out[g] = append(out[g], keys...)
		}
	}
	for g, keys := range out {
		seen := map[string]struct{}{}
		var uniq []string
		for _, k := range keys {
			if k == "" {
				continue
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			uniq = append(uniq, k)
		}
		out[g] = uniq
	}
	return out, nil
}

func entriesStoreKeys(entries []Entry) []string {
	var keys []string
	for _, e := range entries {
		keys = append(keys, e.StoreKey())
		if e.PreviousKey != "" {
			keys = append(keys, e.PreviousKey)
		}
		keys = append(keys, e.ExtraStoreKeys...)
		keys = append(keys, e.PreviousExtraStoreKeys...)
		keys = append(keys, entriesStoreKeys(e.replacedEntries())...)
	}
	return keys
}

// snapshotKeysFile caches the store keys a history's snapshots name. A snapshot never changes under its hash, so
// decoding a long history's gzipped snapshots happens once instead of on every launch.
const snapshotKeysFile = "history-keys.json"

// snapshotKeys is the cache file: the snapshots it covers and the union of their keys. Releases through 0.3.x wrote a
// map of snapshot hash to its own key list instead, which is still read.
type snapshotKeys struct {
	Format    int      `json:"format"`
	Snapshots []string `json:"snapshots"`
	Keys      []string `json:"keys"`
	old       bool     // read from the earlier shape, so rewritten even when nothing changed
}

const snapshotKeysFormat = 2

// readSnapshotKeys returns the cache file's content in the current shape, empty when absent or unreadable.
func readSnapshotKeys(path string) snapshotKeys {
	b, err := safefs.ReadFile(path)
	if err != nil {
		return snapshotKeys{}
	}
	var cur snapshotKeys
	if json.Unmarshal(b, &cur) == nil && cur.Format == snapshotKeysFormat {
		return cur
	}
	var old map[string][]string
	if json.Unmarshal(b, &old) != nil {
		return snapshotKeys{}
	}
	cur = snapshotKeys{Format: snapshotKeysFormat, old: true}
	for id, ks := range old {
		cur.Snapshots = append(cur.Snapshots, id)
		cur.Keys = append(cur.Keys, ks...)
	}
	slices.Sort(cur.Keys)
	cur.Keys = slices.Compact(cur.Keys)
	return cur
}

func historyStoreKeys(dir string) ([]string, error) {
	data, err := readHistory(dir)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, snapshotKeysFile)
	cached := readSnapshotKeys(path)
	current := map[string]struct{}{}
	var ids []string
	for _, ev := range data.Events {
		if _, dup := current[ev.SnapshotID]; !dup {
			current[ev.SnapshotID] = struct{}{}
			ids = append(ids, ev.SnapshotID)
		}
	}
	// A snapshot the history dropped leaves its keys in the union, so the union is rebuilt rather than trimmed.
	if !slices.ContainsFunc(cached.Snapshots, func(id string) bool { _, ok := current[id]; return !ok }) {
		covered := map[string]struct{}{}
		for _, id := range cached.Snapshots {
			covered[id] = struct{}{}
		}
		ids = slices.DeleteFunc(ids, func(id string) bool { _, ok := covered[id]; return ok })
	} else {
		cached = snapshotKeys{}
		ids = slices.Clone(ids)
	}
	if len(ids) == 0 && !cached.old && len(cached.Snapshots) == len(current) {
		return cached.Keys, nil
	}
	seen := make(map[string]struct{}, len(cached.Keys))
	keys := slices.Clone(cached.Keys)
	for _, k := range keys {
		seen[k] = struct{}{}
	}
	for _, id := range ids {
		entries, found := snapshotEntries(&data, id)
		if !found {
			return nil, fmt.Errorf("history snapshot %s not found", id)
		}
		for _, k := range entriesStoreKeys(entries) {
			if _, dup := seen[k]; !dup {
				seen[k] = struct{}{}
				keys = append(keys, k)
			}
		}
	}
	next := snapshotKeys{Format: snapshotKeysFormat, Snapshots: slices.Sorted(maps.Keys(current)), Keys: keys}
	if err := datadir.WriteJSON(path, next); err != nil {
		log.Printf("history key cache: %v", err)
	}
	return keys, nil
}

// loaderStoreKey is the store key of the installer a per-profile loader was laid into the profile from, "" when the
// profile holds no such loader. No entry names that key, yet removing it would force a download for the next profile.
func loaderStoreKey(gameID string, p Profile, dir string) (string, error) {
	l, ok := game.LoaderOf(gameID, p.Loader)
	if !ok {
		return "", nil
	}
	if _, perProfile := l.(loader.InProfile); !perProfile {
		return "", nil
	}
	st, err := l.Status(loader.Target{Game: gameID, ProfileDir: dir})
	if err != nil || !st.Installed || st.Version == "" {
		return "", err
	}
	return store.LoaderKey(l.ID(), st.Version), nil
}
