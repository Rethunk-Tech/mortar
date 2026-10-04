package nexus

import (
	"cmp"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

const seenFileName = "nexus-seen.json"

// MaxSeenMods is the cap on remembered Nexus mods. Oldest LastLooked entries
// are dropped first so the file stays bounded.
const MaxSeenMods = 2000

// SeenEntry is what the user last looked at for one Nexus mod.
type SeenEntry struct {
	NewestFileUnix int64  `json:"newestFileUnix"`
	NewestChange   string `json:"newestChange"`
	LastLookedUnix int64  `json:"lastLookedUnix"`
}

type seenFile struct {
	Mods map[string]SeenEntry `json:"mods"`
}

// SeenStore records the newest file upload time and newest changelog version
// the user has seen for each Nexus mod.
type SeenStore struct {
	mu   sync.Mutex
	path string
	data seenFile
}

// OpenSeenStore loads (or creates) the seen-state file under dataDir.
func OpenSeenStore(dataDir string) (*SeenStore, error) {
	if dataDir == "" {
		return nil, errors.New("nexus: empty data dir")
	}
	s := &SeenStore{
		path: filepath.Join(dataDir, seenFileName),
		data: seenFile{Mods: map[string]SeenEntry{}},
	}
	found, err := datadir.ReadJSON(s.path, &s.data)
	if err != nil {
		return nil, err
	}
	if !found {
		return s, nil
	}
	if s.data.Mods == nil {
		s.data.Mods = map[string]SeenEntry{}
	}
	return s, nil
}

// Get returns the stored entry for nexusID, if any.
func (s *SeenStore) Get(nexusID string) (SeenEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data.Mods[nexusID]
	return e, ok
}

// MarkSeen records that the user looked at this Nexus mod's details.
func (s *SeenStore) MarkSeen(nexusID string, newestFileUnix int64, newestChange string) error {
	if nexusID == "" {
		return errors.New("nexus: empty mod id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Mods == nil {
		s.data.Mods = map[string]SeenEntry{}
	}
	s.data.Mods[nexusID] = SeenEntry{
		NewestFileUnix: newestFileUnix,
		NewestChange:   newestChange,
		LastLookedUnix: time.Now().Unix(),
	}
	s.evictLocked()
	return s.writeLocked()
}

func (s *SeenStore) evictLocked() {
	n := len(s.data.Mods)
	if n <= MaxSeenMods {
		return
	}
	type pair struct {
		id string
		at int64
	}
	pairs := make([]pair, 0, n)
	for id, e := range s.data.Mods {
		pairs = append(pairs, pair{id: id, at: e.LastLookedUnix})
	}
	slices.SortFunc(pairs, func(a, b pair) int {
		if c := cmp.Compare(a.at, b.at); c != 0 {
			return c
		}
		return cmp.Compare(a.id, b.id)
	})
	drop := n - MaxSeenMods
	for i := range drop {
		delete(s.data.Mods, pairs[i].id)
	}
}

func (s *SeenStore) writeLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(s.path, s.data)
}

// Snapshot copies the map for the frontend.
func (s *SeenStore) Snapshot() map[string]SeenEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.data.Mods)
}
