package nexussvc

import (
	"errors"
	"path/filepath"

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

// Seen is the newest file upload time and changelog version the user has looked at, keyed by Nexus mod id or "source:id".
func (s *Service) Seen() map[string]nexus.SeenEntry {
	if s.seen == nil {
		return map[string]nexus.SeenEntry{}
	}
	return s.seen.Snapshot()
}

// MarkSeen records that the user looked at this mod's details. key is the Nexus mod id, or "source:id" for a mod from
// another source (the favourites key, such as "github:me/mod").
func (s *Service) MarkSeen(key string, newestFileUnix int64, newestChange string) error {
	if s.seen == nil {
		return errors.New("nexus: seen store not open")
	}
	return s.seen.MarkSeen(key, newestFileUnix, newestChange)
}
