package profile

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func (s *Store) ProfileDir(game, id string) (string, error) {
	mods, err := s.ModsDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Dir(mods), nil
}

// SetGmcmOption makes one option of the mod's in-game menu a pending edit, which the bridge applies when the game next
// starts; a value equal to the captured one drops the option's edit instead. Each change is a history event, so Undo
// takes it back like a config.json edit.
func (s *Store) SetGmcmOption(game, id string, uniqueID mod.ID, page string, index int, value string) (gmcm.Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return gmcm.Pending{}, err
	}
	_, dir, err := s.readDir(game, id)
	if err != nil {
		return gmcm.Pending{}, err
	}
	menu, err := gmcm.ReadCapture(dir, uniqueID)
	if err != nil {
		return gmcm.Pending{}, err
	}
	edit, err := gmcm.EditFromCapture(menu, page, index, value)
	if err != nil {
		return gmcm.Pending{}, err
	}
	opt, _ := gmcm.Find(menu, page, index)
	pending, err := gmcm.ReadPending(dir, uniqueID)
	if err != nil {
		return gmcm.Pending{}, err
	}
	edits := slices.DeleteFunc(pending.Edits, func(e gmcm.Edit) bool { return e.Page == page && e.Index == index })
	if gmcm.Text(edit.Value) != gmcm.Text(opt.Value) {
		edits = append(edits, edit)
	}
	note := HistoryEvent{Change: ChangeOptionSet, Name: uniqueID.Local(), Detail: opt.Name}
	err = s.editConfigLocked(game, id, note, func() error { return gmcm.WritePending(dir, uniqueID, edits) })
	return gmcm.Pending{Schema: gmcm.Schema, Edits: edits}, err
}

// hasGmcmCapture reports whether the bridge captured the mod's in-game menu in this profile.
func hasGmcmCapture(dir string, uniqueID mod.ID) bool {
	_, err := os.Stat(gmcm.CapturePath(dir, uniqueID))
	return err == nil
}

func (s *Service) GmcmMenu(game, profile string, uniqueID mod.ID) (gmcm.Capture, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return gmcm.Capture{}, err
	}
	return gmcm.ReadCapture(d, uniqueID)
}

func (s *Service) PendingGmcm(game, profile string, uniqueID mod.ID) (gmcm.Pending, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return gmcm.Pending{}, err
	}
	return gmcm.ReadPending(d, uniqueID)
}

// SetGmcmOption sets one in-game menu option for the next start; see Store.SetGmcmOption.
func (s *Service) SetGmcmOption(game, profile string, uniqueID mod.ID, page string, index int, value string) (gmcm.Pending, error) {
	return s.store.SetGmcmOption(game, profile, uniqueID, page, index, value)
}

func (s *Service) GmcmResult(game, profile string, uniqueID mod.ID) (gmcm.Result, error) {
	d, err := s.store.ProfileDir(game, profile)
	if err != nil {
		return gmcm.Result{}, err
	}
	return gmcm.ReadResult(d, uniqueID)
}
