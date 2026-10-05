package savessvc

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// PlayedMod is one enabled mod from the profile at the end of a run.
type PlayedMod struct {
	ID             mod.ID `json:"id"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	Key            string `json:"key"`
	SourceKind     string `json:"sourceKind,omitempty"`
	ModID          int    `json:"modId,omitempty"`
	FileID         int    `json:"fileId,omitempty"`
	Repo           string `json:"repo,omitempty"`
	Tag            string `json:"tag,omitempty"`
	Asset          string `json:"asset,omitempty"`
	ContentPackFor mod.ID `json:"contentPackFor,omitempty"`
}

// LastPlayed maps a save folder name to the profile it was last launched with.
type LastPlayed struct {
	ProfileID string      `json:"profileId"`
	At        time.Time   `json:"at"`
	Mods      []PlayedMod `json:"mods,omitempty"`
}

type lastPlayedFile struct {
	Saves map[string]LastPlayed `json:"saves"`
}

// Store holds per-game last-played associations.
type Store struct {
	mu   sync.Mutex
	root string
}

// NewStore writes last-played records under root/<gameID>/last-played.json.
func NewStore(root string) *Store {
	return &Store{root: root}
}

func (s *Store) path(gameID string) string {
	return filepath.Join(s.root, gameID, "last-played.json")
}

// Get returns the last-played record for a save, or ok=false if none.
func (s *Store) Get(gameID, saveFolder string) (LastPlayed, bool, error) {
	all, err := s.load(gameID)
	if err != nil {
		return LastPlayed{}, false, err
	}
	rec, ok := all.Saves[saveFolder]
	return rec, ok, nil
}

// All returns every last-played record for a game.
func (s *Store) All(gameID string) (map[string]LastPlayed, error) {
	all, err := s.load(gameID)
	if err != nil {
		return nil, err
	}
	return maps.Clone(all.Saves), nil
}

// Record stores that saveFolder was last played with profileID at at.
func (s *Store) Record(gameID, saveFolder, profileID string, at time.Time) error {
	return s.RecordRun(gameID, saveFolder, profileID, at, nil)
}

// RecordRun stores the profile and the enabled mod list from that run.
func (s *Store) RecordRun(gameID, saveFolder, profileID string, at time.Time, mods []PlayedMod) error {
	if gameID == "" || saveFolder == "" || profileID == "" {
		return errors.New("game, save, and profile are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.loadUnlocked(gameID)
	if err != nil {
		return err
	}
	all.Saves[saveFolder] = LastPlayed{ProfileID: profileID, At: at.UTC(), Mods: slices.Clone(mods)}
	return s.writeUnlocked(gameID, all)
}

func (s *Store) load(gameID string) (lastPlayedFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked(gameID)
}

func (s *Store) loadUnlocked(gameID string) (lastPlayedFile, error) {
	var parsed lastPlayedFile
	found, err := datadir.ReadJSON(s.path(gameID), &parsed)
	if err != nil {
		return lastPlayedFile{}, err
	}
	if !found {
		return lastPlayedFile{Saves: map[string]LastPlayed{}}, nil
	}
	if parsed.Saves == nil {
		parsed.Saves = map[string]LastPlayed{}
	}
	return parsed, nil
}

func (s *Store) writeUnlocked(gameID string, all lastPlayedFile) error {
	if err := os.MkdirAll(filepath.Dir(s.path(gameID)), 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(s.path(gameID), all)
}
