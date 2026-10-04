package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/usererr"
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(gameID, id); err != nil {
		return err
	}
	src, err := s.profileDir(gameID, id)
	if err != nil {
		return err
	}
	dst, _ := s.trashDir(gameID, id)
	if _, err := os.Stat(src); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	if err := fsx.Rename(src, dst); err != nil {
		return err
	}
	now := time.Now()
	if err := os.Chtimes(dst, now, now); err != nil {
		return err
	}
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
	if err := fsx.Rename(src, dst); err != nil {
		return Profile{}, err
	}
	return s.read(gameID, id)
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
	return os.RemoveAll(dir)
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
		errs = append(errs, os.RemoveAll(filepath.Join(dir, item.Name())))
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
				errs = append(errs, os.RemoveAll(filepath.Join(s.trash, g.Name(), it.ID)))
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
		profiles, err := s.listOK(g)
		if err != nil {
			return nil, err
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
			if !history {
				continue
			}
			dir := filepath.Join(s.root, g, p.ID)
			if _, err := os.Stat(filepath.Join(dir, fileName)); err != nil {
				dir = filepath.Join(s.trash, g, p.ID)
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
		keys = append(keys, e.Key)
		if e.PreviousKey != "" {
			keys = append(keys, e.PreviousKey)
		}
		keys = append(keys, e.ExtraStoreKeys...)
		keys = append(keys, e.PreviousExtraStoreKeys...)
	}
	return keys
}

// snapshotKeysFile caches, per snapshot hash, the store keys its entries name. A snapshot never changes under its
// hash, so decoding a long history's gzipped snapshots happens once instead of on every launch.
const snapshotKeysFile = "history-keys.json"

func historyStoreKeys(dir string) ([]string, error) {
	data, err := readHistory(dir)
	if err != nil {
		return nil, err
	}
	cached := map[string][]string{}
	if _, err := datadir.ReadJSON(filepath.Join(dir, snapshotKeysFile), &cached); err != nil {
		cached = map[string][]string{}
	}
	next := make(map[string][]string, len(data.Events))
	var keys []string
	for _, ev := range data.Events {
		k, ok := next[ev.SnapshotID]
		if !ok {
			if k, ok = cached[ev.SnapshotID]; !ok {
				entries, found := snapshotEntries(&data, ev.SnapshotID)
				if !found {
					return nil, fmt.Errorf("history snapshot %s not found", ev.SnapshotID)
				}
				k = entriesStoreKeys(entries)
			}
			next[ev.SnapshotID] = k
		}
		keys = append(keys, k...)
	}
	if !maps.EqualFunc(cached, next, slices.Equal) {
		if err := datadir.WriteJSON(filepath.Join(dir, snapshotKeysFile), next); err != nil {
			log.Printf("history key cache: %v", err)
		}
	}
	return keys, nil
}
