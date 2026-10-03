// Package datasvc measures Mortar's data folder and removes unused store items, expired cache files, and leftover temp folders.
package datasvc

import (
	"errors"
	"path/filepath"
	"slices"
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
	modCache ModUsage
	modFP    string
	busy     []BusySource
	// Busy is true while the game is launching or running; nil means never busy.
	Busy func() bool
	// Restart starts Mortar again after a successful move; nil skips that in tests.
	Restart func() error
}

type BusySource interface {
	Busy() bool
}

type BusyFunc func() bool

func (f BusyFunc) Busy() bool { return f() }

type MoveEstimate struct {
	Bytes     int64 `json:"bytes"`
	FreeBytes int64 `json:"freeBytes"`
}

// NewService measures and cleans the data folder using the store's Collect keep set.
func NewService(items *store.Store, profiles *profile.Store, staged func() map[string][]string, busy ...BusySource) *Service {
	return &Service{items: items, profiles: profiles, staged: staged, busy: busy}
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

// CacheInfo is the cache folder path and size.
func (s *Service) CacheInfo() (CacheInfo, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return CacheInfo{}, err
	}
	cache := filepath.Join(dir, "cache")
	return CacheInfo{Path: cache, Size: dirSize(cache)}, nil
}

// ClearCache deletes the contents of the cache folder. Problem scans rebuild on the next check.
func (s *Service) ClearCache() error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	return clearCache(filepath.Join(dir, "cache"))
}

// ModUsage lists store items with sizes, cached until the store or profiles change.
func (s *Service) ModUsage() (ModUsage, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return ModUsage{}, err
	}
	fp := usageFingerprint(dir)
	s.mu.Lock()
	if s.modFP == fp && s.modFP != "" {
		u := s.modCache
		s.mu.Unlock()
		return u, nil
	}
	s.mu.Unlock()
	u, err := MeasureMods(dir)
	if err != nil {
		return ModUsage{}, err
	}
	s.mu.Lock()
	s.modCache = u
	s.modFP = fp
	s.mu.Unlock()
	return u, nil
}

// EntrySizes is each store key's size, the same data ModUsage uses.
func (s *Service) EntrySizes() ([]EntrySize, error) {
	u, err := s.ModUsage()
	if err != nil {
		return nil, err
	}
	out := make([]EntrySize, 0, len(u.Items))
	for _, it := range u.Items {
		out = append(out, EntrySize{Game: it.Game, Key: it.Key, Size: it.Size})
	}
	return out, nil
}

func (s *Service) forgetModUsage() {
	s.mu.Lock()
	s.modFP = ""
	s.mu.Unlock()
}

// RemoveStoreItem deletes one store folder when no keep-set entry still names it.
func (s *Service) RemoveStoreItem(game, key string) error {
	keys, err := s.referenced()
	if err != nil {
		return err
	}
	if slices.Contains(keys[game], key) {
		return errInUse
	}
	if err := s.items.Remove([]store.Ref{{Game: game, Key: key}}); err != nil {
		return err
	}
	s.forgetModUsage()
	return nil
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
	return Select(dir, s.items, keys, time.Now(), func(game, key string) string {
		source := s.profiles.SourceOf(game, key)
		if source.Name == "" {
			return ""
		}
		if source.Version == "" {
			return source.Name
		}
		return source.Name + " " + source.Version
	})
}

// Cleanup removes exactly the unused set CleanupPreview listed.
func (s *Service) Cleanup(preview Preview) error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	err = s.applyPreview(dir, preview)
	if err == nil {
		s.forgetModUsage()
	}
	return err
}

func (s *Service) applyPreview(root string, preview Preview) error {
	keys, err := s.referenced()
	if err != nil {
		return err
	}
	return Apply(root, s.items, preview, keys)
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

func (s *Service) MoveDataFolderPreview(dest string) (MoveEstimate, error) {
	src, err := datadir.Dir()
	if err != nil {
		return MoveEstimate{}, err
	}
	estimate, err := datadir.EstimateRelocate(src, dest)
	if err != nil {
		return MoveEstimate{}, err
	}
	return MoveEstimate{Bytes: estimate.Bytes, FreeBytes: estimate.FreeBytes}, nil
}

var errGameRunning = errors.New("stop the game before moving the data folder")

var errInUse = errors.New("a profile still uses this store item")

// MoveDataFolder copies the data folder to dest, verifies it, points the default location at dest, removes the old copy, and restarts.
func (s *Service) MoveDataFolder(dest string) error {
	if dest == "" {
		return errors.New("no folder chosen")
	}
	if s.Busy != nil && s.Busy() {
		return errGameRunning
	}
	for _, busy := range s.busy {
		if busy != nil && busy.Busy() {
			return errGameRunning
		}
	}
	src, err := datadir.Dir()
	if err != nil {
		return err
	}
	def, err := datadir.DefaultDir()
	if err != nil {
		return err
	}
	if err := datadir.Relocate(src, dest, def, func(p datadir.CopyProgress) {
		s.setProgress(Progress{
			Copying: true, Files: p.Files, TotalFiles: p.TotalFiles, Bytes: p.Bytes, TotalBytes: p.TotalBytes,
		})
	}); err != nil {
		s.setProgress(Progress{})
		return err
	}
	s.setProgress(Progress{})
	if s.Restart != nil {
		return s.Restart()
	}
	return nil
}
