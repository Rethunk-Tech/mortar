package profile

import "github.com/Rethunk-Tech/mortar/internal/migrate"

// ExternalSources lists detected mod-manager profiles for the selected game.
func (s *Service) ExternalSources(gameID string) ([]migrate.SourceInfo, error) {
	modsDir, err := s.gameModsDir(gameID)
	if err != nil {
		return nil, err
	}
	return migrate.Detect("", modsDir)
}

// ExternalPreview reads one detected external profile without changing either manager's files.
func (s *Service) ExternalPreview(gameID, kind, id string) (migrate.ProfilePreview, error) {
	modsDir, err := s.gameModsDir(gameID)
	if err != nil {
		return migrate.ProfilePreview{}, err
	}
	return migrate.Preview("", modsDir, kind, id)
}
