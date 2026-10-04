package bundles

// ProfileMods snapshots every mod of a profile in the form bundles store them.
func (s *Service) ProfileMods(gameID, profileID string) ([]Mod, error) {
	profiles, err := s.profiles.List(gameID)
	if err != nil {
		return nil, err
	}
	p, err := profileFor(profiles, profileID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		for _, m := range e.Mods {
			ids = append(ids, m.UniqueID)
		}
	}
	if len(ids) == 0 {
		return []Mod{}, nil
	}
	return snapshot(p, ids)
}
