package profile

// Origin kinds recorded when a profile is created by import or duplicate.
const (
	OriginLink       = "link"
	OriginMortar     = "mortar"
	OriginGameMods   = "game-mods"
	OriginCopy       = "copy"
	OriginCollection = "collection"
)

// SetOrigin records how the profile was created. Notes and origin never touch mods/, so a running game does not block it.
func (s *Store) SetOrigin(game, id, kind, copyOf string) (Profile, error) {
	return s.update(game, id, func(p *Profile, _ string) error {
		p.Origin = kind
		p.CopyOf = copyOf
		return nil
	})
}
