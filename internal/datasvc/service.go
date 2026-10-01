// Package datasvc measures Mortar's data folder and removes unused store items, expired cache files, and leftover temp folders.
package datasvc

import (
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/store"
)

// Service is the Settings › Data backend.
type Service struct {
	items    *store.Store
	profiles *profile.Store
	staged   func() map[string][]string
	mu       sync.Mutex
	progress Progress
	// Busy is true while the game is launching or running; nil means never busy.
	Busy func() bool
	// Restart starts Mortar again after a successful move; nil skips that in tests.
	Restart func() error
}

// NewService measures and cleans the data folder using the store's Collect keep set.
func NewService(items *store.Store, profiles *profile.Store, staged func() map[string][]string) *Service {
	return &Service{items: items, profiles: profiles, staged: staged}
}

// UsageProgress is the in-flight size walk, or Measuring false when idle.
func (s *Service) UsageProgress() Progress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

// Usage walks the data folder in the background (Wails) and reports sizes.
func (s *Service) Usage() (Usage, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return Usage{}, err
	}
	s.setProgress(Progress{Measuring: true})
	defer s.setProgress(Progress{})
	return Measure(dir, s.setProgress)
}

// CleanupPreview lists what Clean up unused would remove.
func (s *Service) CleanupPreview() (Preview, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return Preview{}, err
	}
	keys, err := s.referenced()
	if err != nil {
		return Preview{}, err
	}
	return Select(dir, s.items, keys, time.Now())
}

// Cleanup removes exactly the unused set CleanupPreview listed.
func (s *Service) Cleanup() error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	keys, err := s.referenced()
	if err != nil {
		return err
	}
	preview, err := Select(dir, s.items, keys, time.Now())
	if err != nil {
		return err
	}
	return s.applyPreview(dir, preview)
}

func (s *Service) applyPreview(root string, preview Preview) error {
	var refs []store.Ref
	for _, it := range preview.Items {
		if it.Kind == "store" {
			game, key, ok := strings.Cut(strings.TrimPrefix(it.Rel, "store/"), "/")
			if !ok {
				continue
			}
			keep, err := s.referenced()
			if err != nil {
				return err
			}
			if slices.Contains(keep[game], key) {
				continue
			}
			abs, confErr := confined(root, it.Rel)
			if confErr != nil {
				continue
			}
			_ = os.RemoveAll(abs)
			refs = append(refs, store.Ref{Game: game, Key: key})
			continue
		}
		abs, confErr := confined(root, it.Rel)
		if confErr != nil {
			continue
		}
		_ = os.RemoveAll(abs)
	}
	return s.items.Remove(refs)
}

func (s *Service) referenced() (map[string][]string, error) {
	keys, err := s.profiles.StoreKeys()
	if err != nil {
		return nil, err
	}
	if s.staged != nil {
		for g, staged := range s.staged() {
			keys[g] = append(keys[g], staged...)
		}
	}
	return keys, nil
}

func (s *Service) setProgress(p Progress) {
	s.mu.Lock()
	s.progress = p
	s.mu.Unlock()
}

var errGameRunning = errors.New("stop the game before moving the data folder")

// MoveDataFolder copies the data folder to dest, verifies it, points the default location at dest, removes the old copy, and restarts.
func (s *Service) MoveDataFolder(dest string) error {
	if dest == "" {
		return errors.New("no folder chosen")
	}
	if s.Busy != nil && s.Busy() {
		return errGameRunning
	}
	src, err := datadir.Dir()
	if err != nil {
		return err
	}
	def, err := datadir.DefaultDir()
	if err != nil {
		return err
	}
	if err := datadir.Relocate(src, dest, def); err != nil {
		return err
	}
	if s.Restart != nil {
		return s.Restart()
	}
	return nil
}
