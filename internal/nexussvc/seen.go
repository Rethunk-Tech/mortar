package nexussvc

import (
	"errors"
	"strconv"

	"github.com/Rethunk-AI/mortar/internal/nexus"
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
	return nil
}

// Seen is the newest file upload time and changelog version the user has looked at, keyed by Nexus mod id.
func (s *Service) Seen() map[string]nexus.SeenEntry {
	if s.seen == nil {
		return map[string]nexus.SeenEntry{}
	}
	return s.seen.Snapshot()
}

// MarkSeen records that the user looked at this Nexus mod's details.
func (s *Service) MarkSeen(modID int, newestFileUnix int64, newestChange string) error {
	if s.seen == nil {
		return errors.New("nexus: seen store not open")
	}
	if modID <= 0 {
		return errors.New("nexus: invalid mod id")
	}
	return s.seen.MarkSeen(strconv.Itoa(modID), newestFileUnix, newestChange)
}
