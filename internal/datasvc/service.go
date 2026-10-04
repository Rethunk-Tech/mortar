// Package datasvc measures Mortar's data folder and removes unused store items, expired cache files, and leftover temp folders.
package datasvc

import (
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/store"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

// Service is the Settings › Data backend.
type Service struct {
	items    *store.Store
	profiles *profile.Store
	keep     []KeySource
	mu       sync.Mutex
	progress Progress
	modCache ModUsage
	modFP    string
	usage    Usage
	usageAt  time.Time
	busy     []BusySource
	// OnClearCache runs after ClearCache empties the cache folder, to drop copies held in memory.
	OnClearCache func()
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

// KeySource lists, per game, store keys something other than a profile still needs.
type KeySource func() (map[string][]string, error)

// KeepSet is the store keep set: every key a profile names (history snapshots too when history is set) plus what
// the sources add. Clean up, Remove and the startup sweep all build it here, so they cannot disagree.
func KeepSet(profiles *profile.Store, history bool, sources []KeySource) (map[string][]string, error) {
	keys, err := profiles.StoreKeys(history)
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		extra, err := source()
		if err != nil {
			return nil, err
		}
		for g, more := range extra {
			keys[g] = append(keys[g], more...)
		}
	}
	return keys, nil
}

// NewService measures and cleans the data folder using the store's Collect keep set.
func NewService(items *store.Store, profiles *profile.Store, keep []KeySource, busy ...BusySource) *Service {
	return &Service{items: items, profiles: profiles, keep: keep, busy: busy}
}

// UsageProgress is the in-flight size walk, or Measuring false when idle.
func (s *Service) UsageProgress() Progress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

// usageTTL is how long a measured Usage is reused; backups, caches and saves change without Mortar's store hooks
// seeing it, so the Refresh button passes fresh to measure again.
const usageTTL = time.Minute

// Usage walks the data folder in the background (Wails) and reports sizes. It reuses the last walk for usageTTL
// unless fresh is set or the store changed.
func (s *Service) Usage(fresh bool) (Usage, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return Usage{}, err
	}
	s.mu.Lock()
	if !fresh && !s.usageAt.IsZero() && time.Since(s.usageAt) < usageTTL && s.usage.Path == dir {
		u := s.usage
		s.mu.Unlock()
		return u, nil
	}
	s.mu.Unlock()
	s.setProgress(Progress{Measuring: true})
	defer s.setProgress(Progress{})
	u, err := Measure(dir, s.setProgress)
	if err != nil {
		return Usage{}, err
	}
	s.mu.Lock()
	s.usage, s.usageAt = u, time.Now()
	s.mu.Unlock()
	return u, nil
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

// ClearCache deletes the contents of the cache folder (Nexus and SMAPI details, problem scans) and
// what is held in memory from it; everything is fetched or rebuilt again when next needed.
func (s *Service) ClearCache() error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	if err := clearCache(filepath.Join(dir, "cache")); err != nil {
		return err
	}
	s.forgetModUsage()
	if s.OnClearCache != nil {
		s.OnClearCache()
	}
	return nil
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
	s.usageAt = time.Time{}
	s.mu.Unlock()
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
	sources := map[string]map[string]profile.Source{}
	return Select(dir, s.items, keys, time.Now(), func(game, key string) string {
		if sources[game] == nil {
			sources[game] = s.profiles.SourcesOf(game)
		}
		source := sources[game][key]
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
	return KeepSet(s.profiles, true, s.keep)
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
	estimate, err := datadir.EstimateRelocate(src, dest, extentTotal)
	if err != nil {
		return MoveEstimate{}, err
	}
	return MoveEstimate{Bytes: estimate.Bytes, FreeBytes: estimate.FreeBytes}, nil
}

// extentTotal is Measure's total for root: bytes on disk with shared extents counted once.
func extentTotal(root string) (int64, error) {
	u, err := Measure(root, nil)
	return u.Total, err
}

// DataLocation is where Mortar keeps its data and whether a portable marker put it there.
type DataLocation struct {
	Dir      string `json:"dir"`
	Portable bool   `json:"portable"`
}

// DataLocation reports the data folder in use; in portable mode Move is refused, so the UI hides it.
func (s *Service) DataLocation() (DataLocation, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return DataLocation{}, err
	}
	return DataLocation{Dir: dir, Portable: datadir.Portable()}, nil
}

var errPortable = errors.New("this copy of Mortar is portable: its data stays in the data folder beside it")

var errGameRunning = usererr.New(usererr.Busy, "stop the game before moving the data folder")

var errInUse = errors.New("a profile still uses this store item")

// MoveDataFolder copies the data folder to dest, verifies it, points the default location at dest, removes the old copy, and restarts.
func (s *Service) MoveDataFolder(dest string) error {
	if dest == "" {
		return errors.New("no folder chosen")
	}
	if datadir.Portable() {
		return errPortable
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
	if err := datadir.Relocate(src, dest, def, extentTotal, func(p datadir.CopyProgress) {
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
