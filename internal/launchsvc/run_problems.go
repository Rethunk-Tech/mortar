package launchsvc

import (
	"fmt"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launch"
)

// RunProblems returns recognised SMAPI errors in a stored run (or the live/latest log when runID is empty).
func (s *Service) RunProblems(gameID, profileID, runID string) ([]launch.SMAPIProblem, error) {
	g := game.Find(gameID)
	if g == nil {
		return nil, fmt.Errorf("unknown game %q", gameID)
	}
	text, err := s.problemLog(g, profileID, runID)
	if err != nil {
		return nil, err
	}
	found := launch.ParseSMAPIProblems(text)
	refs := s.installedModRefs(gameID, profileID)
	return launch.ResolveSMAPIProblemMods(found, refs), nil
}

func (s *Service) problemLog(g game.Game, profileID, runID string) (string, error) {
	if runID != "" {
		return s.RunLog(g.ID(), profileID, runID)
	}
	modsDir, err := s.profiles.ModsDir(g.ID(), profileID)
	if err != nil {
		return "", err
	}
	if text := s.runText(g, profileID, modsDir); text != "" {
		return text, nil
	}
	runs, err := s.Runs(g.ID(), profileID)
	if err != nil {
		return "", err
	}
	if len(runs) == 0 {
		return "", nil
	}
	return s.RunLog(g.ID(), profileID, runs[0].ID)
}

func (s *Service) installedModRefs(gameID, profileID string) []launch.ModRef {
	if s.profiles == nil || profileID == "" {
		return nil
	}
	installed, err := s.profiles.Installed(gameID, profileID)
	if err != nil {
		return nil
	}
	refs := make([]launch.ModRef, 0, len(installed))
	for _, mod := range installed {
		refs = append(refs, launch.ModRef{
			Name: mod.Name, UniqueID: mod.UniqueID, Key: mod.Key, Version: mod.Version, SourceVersion: mod.Source.Version,
		})
	}
	return refs
}
