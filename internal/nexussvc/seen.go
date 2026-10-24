package nexussvc

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

// UseDataDir opens the per-Nexus-mod seen-state file under dir.
//
//wails:ignore
func (s *Service) UseDataDir(dir string) error {
	st, err := nexus.OpenSeenStore(dir)
	if err != nil {
		return err
	}
	prompts, err := openPromptStore(dir)
	if err != nil {
		return err
	}
	s.seen = st
	s.prompts = prompts
	s.trackedPath = filepath.Join(dir, trackedFile)
	return nil
}

// Seen is the newest file upload time and changelog version the user has looked at. A Nexus mod is keyed
// NexusSeenKey(game, id); a bare mod id is how releases before that recorded one, readable as a look at that id in any
// game; a mod from another source is keyed "source:id".
func (s *Service) Seen() map[string]nexus.SeenEntry {
	if s.seen == nil {
		return map[string]nexus.SeenEntry{}
	}
	return s.seen.Snapshot()
}

// NexusSeenKey is the seen-state key of a Nexus mod: an id names a page only within its game's domain.
func NexusSeenKey(gameID string, modID int) string {
	return "nexus:" + gameID + ":" + strconv.Itoa(modID)
}

// MarkNexusSeen records that the user looked at this Nexus mod's details in this game.
func (s *Service) MarkNexusSeen(gameID string, modID int, newestFileUnix int64, newestChange string) error {
	if gameID == "" || modID <= 0 {
		return fmt.Errorf("nexus: no mod %d in game %q", modID, gameID)
	}
	return s.MarkSeen(NexusSeenKey(gameID, modID), newestFileUnix, newestChange)
}

// MarkSeen records that the user looked at a mod from a source other than Nexus, by its favourites key ("source:id",
// such as "github:me/mod"). A bare Nexus id is refused: it names no game, so it goes through MarkNexusSeen.
func (s *Service) MarkSeen(key string, newestFileUnix int64, newestChange string) error {
	if s.seen == nil {
		return errors.New("nexus: seen store not open")
	}
	if _, err := strconv.Atoi(key); err == nil {
		return fmt.Errorf("nexus: seen key %q names no game", key)
	}
	return s.seen.MarkSeen(key, newestFileUnix, newestChange)
}
