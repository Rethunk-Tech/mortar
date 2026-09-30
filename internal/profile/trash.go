package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
)

// trashRetention is how long a deleted profile stays restorable.
const trashRetention = 30 * 24 * time.Hour

// TrashItem is a deleted profile that can still be restored.
type TrashItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	DeletedAt time.Time `json:"deletedAt"`
	DaysLeft  int       `json:"daysLeft"`
}

func (s *Store) trashDir(gameID, id string) (string, error) {
	if !game.Valid(gameID) {
		return "", fmt.Errorf("unknown game %q", gameID)
	}
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("invalid profile id %q", id)
	}
	return filepath.Join(s.trash, gameID, id), nil
}

// Delete moves the profile folder to <datadir>/trash/<game>/<id>/ and stamps it with the deletion time.
func (s *Store) Delete(gameID, id string) error {
	if err := s.unlocked(gameID, id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
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
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	now := time.Now()
	return os.Chtimes(dst, now, now)
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
		info, err := d.Info()
		if err != nil {
			return nil, err
		}
		p, err := readAt(filepath.Join(dir, d.Name()), d.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, TrashItem{ID: p.ID, Name: p.Name, DeletedAt: info.ModTime().UTC()})
	}
	return out, nil
}

// ListTrash returns the game's deleted profiles, newest deletion first, with the days left before they are purged.
func (s *Store) ListTrash(gameID string) ([]TrashItem, error) {
	if !game.Valid(gameID) {
		return nil, fmt.Errorf("unknown game %q", gameID)
	}
	items, err := s.trashed(gameID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range items {
		left := trashRetention - now.Sub(items[i].DeletedAt)
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
	if err := os.Rename(src, dst); err != nil {
		return Profile{}, err
	}
	return s.read(gameID, id)
}

// PurgeTrash deletes trashed profiles deleted more than 30 days before now.
func (s *Store) PurgeTrash(now time.Time) error {
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
			if now.Sub(it.DeletedAt) > trashRetention {
				errs = append(errs, os.RemoveAll(filepath.Join(s.trash, g.Name(), it.ID)))
			}
		}
	}
	return errors.Join(errs...)
}

// StoreKeys lists, per game, every store key that a profile or trashed profile names as key or previousKey.
func (s *Store) StoreKeys() (map[string][]string, error) {
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
			for _, e := range p.Entries {
				out[g] = append(out[g], e.Key)
				if e.PreviousKey != "" {
					out[g] = append(out[g], e.PreviousKey)
				}
			}
		}
	}
	return out, nil
}
