package profile

// ModpackDraft is what the modpack form remembers for a profile: its fields, and the packages and changelog of the
// last export, which the next export's changelog is written against.
type ModpackDraft struct {
	Name   string `json:"name,omitempty"`
	Author string `json:"author,omitempty"`
	// Team is the Thunderstore team of the last upload, which the next upload starts from.
	Team            string `json:"team,omitempty"`
	Version         string `json:"version,omitempty"`
	Description     string `json:"description,omitempty"`
	Website         string `json:"website,omitempty"`
	Icon            string `json:"icon,omitempty"`
	Readme          string `json:"readme,omitempty"`
	Changelog       string `json:"changelog,omitempty"`
	IncludeDisabled bool   `json:"includeDisabled,omitempty"`
	// Exported lists the packages of the last export as Namespace-Name-version.
	Exported []string `json:"exported,omitempty"`
}

// SetModpackDraft records the modpack form of the profile's last export.
func (s *Store) SetModpackDraft(game, id string, d ModpackDraft) (Profile, error) {
	return s.update(game, id, func(p *Profile, _ string) error {
		p.Modpack = &d
		return nil
	})
}
