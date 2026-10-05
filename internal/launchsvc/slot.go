package launchsvc

import (
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// slot is a game seen through one of its installs. Launch state (status, console session, stop, preparing claim)
// is kept per slot, so each install has its own running lock. A slot without an install id is a game with no
// discovered install.
type slot struct {
	game.Game
	inst string
}

func installOf(g game.Game) string {
	if sl, ok := g.(slot); ok {
		return sl.inst
	}
	return ""
}

func keyOf(g game.Game) string { return slotKey(g.ID(), installOf(g)) }

func slotKey(gameID, installID string) string { return gameID + "/" + installID }

func (s *Service) currentSettings() settings.Settings {
	if s.settings == nil {
		return settings.Defaults()
	}
	return s.settings.Get()
}

// installs returns every install of the game Mortar knows and the selected one, which is among them unless it is a
// folder the user chose that no store reported.
func (s *Service) installs(g game.Game) (all []game.Install, selected game.Install) {
	_, _, all, err := game.Resolve(s.home, s.currentSettings(), g.ID())
	if err != nil {
		return nil, game.Install{}
	}
	selected, _ = game.ResolveInstall(s.home, s.currentSettings(), g.ID(), "")
	return all, selected
}

// slots lists the game's slots, the selected install's first.
func (s *Service) slots(g game.Game) []slot {
	all, selected := s.installs(g)
	out := []slot{{g, selected.ID}}
	for _, in := range all {
		if in.ID != selected.ID {
			out = append(out, slot{g, in.ID})
		}
	}
	return out
}

func (s *Service) selectedSlot(g game.Game) slot {
	_, selected := s.installs(g)
	return slot{g, selected.ID}
}

// profileSlot is the slot of the install the profile launches: its pinned install, else the selected one.
func (s *Service) profileSlot(g game.Game, profileID string) slot {
	if s.profiles == nil || profileID == "" {
		return s.selectedSlot(g)
	}
	pin := s.profiles.InstallOf(g.ID(), profileID)
	if pin == "" {
		return s.selectedSlot(g)
	}
	return slot{g, pin}
}

// activeSlot is the game's slot that is launching or running, the selected install's first.
func (s *Service) activeSlot(g game.Game) (slot, bool) {
	for _, sl := range s.slots(g) {
		if s.current(sl).State.Active() {
			return sl, true
		}
	}
	return slot{}, false
}

// owner is the id of the install the process runs from; a process that names none of them is the selected install's.
func (s *Service) owner(g game.Game, p launch.Process) string {
	all, selected := s.installs(g)
	for _, in := range all {
		if p.RunsFrom(in.Dir) {
			return in.ID
		}
	}
	return selected.ID
}

// ownedBy keeps the processes that belong to g's install.
func (s *Service) ownedBy(g game.Game, procs []launch.Process) []launch.Process {
	var out []launch.Process
	for _, p := range procs {
		if s.owner(g, p) == installOf(g) {
			out = append(out, p)
		}
	}
	return out
}
