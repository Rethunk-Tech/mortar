package profile

// ModInProfile is one profile that holds a UniqueID, as read from profile.json.
type ModInProfile struct {
	ProfileID   string `json:"profileId"`
	ProfileName string `json:"profileName"`
	Key         string `json:"key"`
	UniqueID    string `json:"uniqueId"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
}

// ProfilesWithMod lists the profiles of game whose profile.json names uniqueID, without reading the store.
func (s *Store) ProfilesWithMod(game, uniqueID string) ([]ModInProfile, error) {
	if uniqueID == "" {
		return []ModInProfile{}, nil
	}
	all, err := s.listOK(game)
	if err != nil {
		return nil, err
	}
	out := []ModInProfile{}
	for _, p := range all {
		if row, ok := modIn(p, uniqueID); ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func modIn(p Profile, uniqueID string) (ModInProfile, bool) {
	for _, e := range p.Entries {
		for _, m := range e.Mods {
			if !SameID(m.UniqueID, uniqueID) {
				continue
			}
			return ModInProfile{
				ProfileID:   p.ID,
				ProfileName: p.Name,
				Key:         e.Key,
				UniqueID:    m.UniqueID,
				Name:        m.Name,
				Version:     m.Version,
				Enabled:     !isDisabled(e, m),
			}, true
		}
	}
	return ModInProfile{}, false
}
