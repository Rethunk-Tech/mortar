package profile

import "github.com/Rethunk-Tech/mortar/internal/mod"

// ModInProfile is one profile that holds a UniqueID, as read from profile.json.
type ModInProfile struct {
	ProfileID   string `json:"profileId"`
	ProfileName string `json:"profileName"`
	Key         string `json:"key"`
	ID          mod.ID `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
}

// ProfilesWithMod lists the profiles of game whose profile.json names uniqueID, without reading the store.
func (s *Store) ProfilesWithMod(game string, uniqueID mod.ID) ([]ModInProfile, error) {
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

func modIn(p Profile, uniqueID mod.ID) (ModInProfile, bool) {
	e, m, ok := p.FindMod("", uniqueID)
	if !ok {
		return ModInProfile{}, false
	}
	return ModInProfile{
		ProfileID:   p.ID,
		ProfileName: p.Name,
		Key:         e.Key,
		ID:          m.ID,
		Name:        m.Name,
		Version:     m.Version,
		Enabled:     e.Enabled(m.ID),
	}, true
}
