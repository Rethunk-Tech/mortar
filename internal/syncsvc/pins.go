package syncsvc

import (
	"errors"
	"slices"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Pins are what a computer keeps its own way on a synced profile.
type Pins struct {
	// Mods are mod ids whose on/off state stays as it is here.
	Mods []mod.ID `json:"mods,omitempty"`
	// Files are config files, named by a zip entry name of the profile's export or its trailing path ("com.x.cfg",
	// "Mod.Id/config.json"), whose content stays as it is here.
	Files []string `json:"files,omitempty"`
}

func (p Pins) empty() bool { return len(p.Mods) == 0 && len(p.Files) == 0 }

// Pin kinds for SetPin.
const (
	PinMod  = "mod"
	PinFile = "file"
)

// Pins returns what this computer keeps its own way in the profile.
func (s *Service) Pins(game, profileID string) Pins {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st.Profiles[key(game, profileID)].Pins
}

// SetPin keeps a mod's on/off state (kind mod, id the mod id) or a config file (kind file, id as Pins.Files) of a
// synced profile as it is on this computer, or lets it follow the sync again. The profile must have been synced once,
// so there is a revision whose value the other computers keep.
func (s *Service) SetPin(game, profileID, kind, id string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(game, profileID)
	t, ok := s.st.Profiles[k]
	if on && (!ok || len(t.Synced) == 0) {
		return usererr.New(usererr.Invalid, "this profile has not been synced yet")
	}
	if id == "" {
		return errors.New("name what to keep on this computer")
	}
	switch kind {
	case PinMod:
		t.Pins.Mods = toggle(t.Pins.Mods, mod.ID(id), on)
	case PinFile:
		t.Pins.Files = toggle(t.Pins.Files, id, on)
	default:
		return errors.New("unknown pin kind " + kind)
	}
	s.st.Profiles[k] = t
	return s.saveState()
}

func toggle[T comparable](list []T, v T, on bool) []T {
	list = slices.DeleteFunc(slices.Clone(list), func(x T) bool { return x == v })
	if on {
		list = append(list, v)
	}
	return list
}

// unpinned is the payload to write: local with each pinned item as the sync folder's current revision has it. same
// reports that this leaves the folder's revision as it is, so there is nothing to write.
func (s *Service) unpinned(folder string, ref Ref, t *tracked, local []byte) (payload []byte, same bool) {
	if t.Pins.empty() {
		return local, false
	}
	head, err := safefs.ReadFile(s.payloadPath(folder, ref.Game, t.Remote))
	if err != nil {
		return local, false
	}
	pinned, err := share.Pin(local, head, t.Pins.Mods, t.Pins.Files)
	if err != nil {
		return local, false
	}
	norm, err := share.Pin(head, head, nil, nil)
	return pinned, err == nil && hashOf(pinned) == hashOf(norm)
}

// repinned is the payload to apply: the incoming one with each pinned item as it is here.
func (s *Service) repinned(game, id string, incoming []byte) []byte {
	t := s.st.Profiles[key(game, id)]
	if t.Pins.empty() {
		return incoming
	}
	here, err := s.d.Source.Export(game, id)
	if err != nil {
		return incoming
	}
	out, err := share.Pin(incoming, here, t.Pins.Mods, t.Pins.Files)
	if err != nil {
		return incoming
	}
	return out
}
