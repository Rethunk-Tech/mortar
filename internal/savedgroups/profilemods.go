package savedgroups

// ProfileMods snapshots every mod of a profile in the form saved groups store them.
//
//wails:ignore
func (s *Service) ProfileMods(gameID, profileID string) ([]Mod, error) {
	p, err := s.profiles.Get(gameID, profileID)
	if err != nil {
		return nil, err
	}
	ids := profileModIDs(p)
	if len(ids) == 0 {
		return []Mod{}, nil
	}
	return snapshot(p, ids)
}
