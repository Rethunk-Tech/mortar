package profile

import (
	"errors"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/migrate"
)

// externalModsDir is the Mods folder the importers fall back to when a manager's own staging folder is gone: only a
// game whose loader reads manifest mods from one has it.
func (s *Service) externalModsDir(gameID string) (string, error) {
	dir, err := s.gameModsDir(gameID)
	if errors.Is(err, errNoManifestMods) {
		return "", nil
	}
	return dir, err
}

// ExternalSources lists detected mod-manager profiles for the selected game.
func (s *Service) ExternalSources(gameID string) ([]migrate.SourceInfo, error) {
	modsDir, err := s.externalModsDir(gameID)
	if err != nil {
		return nil, err
	}
	cur := s.settings.Get()
	return migrate.Detect("", modsDir, gameID, cur.VortexFolder, cur.MO2Folder)
}

// ExternalPreview reads one detected external profile without changing either manager's files.
func (s *Service) ExternalPreview(gameID, kind, id string) (migrate.ProfilePreview, error) {
	modsDir, err := s.externalModsDir(gameID)
	if err != nil {
		return migrate.ProfilePreview{}, err
	}
	cur := s.settings.Get()
	return migrate.Preview("", modsDir, gameID, cur.VortexFolder, cur.MO2Folder, kind, id)
}

// ExternalVortexSupported is whether Vortex profiles can be imported for the game at all.
func (s *Service) ExternalVortexSupported(gameID string) bool {
	info, _ := components.Game(gameID)
	return info.ImportIDs.Vortex != ""
}

// ExternalMO2Supported is whether Mod Organizer 2 profiles can be imported for the game at all.
func (s *Service) ExternalMO2Supported(gameID string) bool {
	info, _ := components.Game(gameID)
	return info.ImportIDs.MO2 != ""
}

// ExternalVortexContents says which games the Vortex data folder holds profiles for, so the importer can explain
// an empty list.
func (s *Service) ExternalVortexContents() (migrate.VortexInventory, error) {
	return migrate.VortexContents("", s.settings.Get().VortexFolder)
}
