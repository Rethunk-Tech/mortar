package bundles

import "github.com/Rethunk-Tech/mortar/internal/mod"

// ProfileMods snapshots every mod of a profile in the form bundles store them.
//
//wails:ignore
func (s *Service) ProfileMods(gameID, profileID string) ([]Mod, error) {
	profiles, err := s.profiles.List(gameID)
	if err != nil {
		return nil, err
	}
	p, err := profileFor(profiles, profileID)
	if err != nil {
		return nil, err
	}
	var ids []mod.ID
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		for _, m := range e.Mods {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		return []Mod{}, nil
	}
	return snapshot(p, ids)
}
