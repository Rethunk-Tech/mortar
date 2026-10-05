package browse

import (
	"context"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// Service is the window's browse API.
type Service struct {
	Version  string
	Profiles *profile.Service
}

// NewService returns a Service that marks hits already on the named profile.
func NewService(version string, profiles *profile.Service) *Service {
	return &Service{Version: version, Profiles: profiles}
}

// Search returns one page of mods for game from source matching text.
func (s *Service) Search(ctx context.Context, game, source, text string, page int, profileID string) (Page, error) {
	c := &Client{Version: s.Version, Installed: s.installed(game, profileID)}
	return c.Search(ctx, game, source, text, page)
}

func (s *Service) installed(game, profileID string) InstalledFunc {
	if s == nil || s.Profiles == nil || strings.TrimSpace(profileID) == "" {
		return nil
	}
	list, err := s.Profiles.List(game)
	if err != nil {
		return nil
	}
	for _, prof := range list {
		if prof.ID == profileID || strings.EqualFold(prof.Name, profileID) {
			return InstalledOn(prof)
		}
	}
	return nil
}

// InstalledOn reports Nexus mod ids and GitHub repos already on prof.
func InstalledOn(prof profile.Profile) InstalledFunc {
	return func(source, id string) bool {
		for _, entry := range prof.Entries {
			switch source {
			case sourceNexus:
				if entry.Source.Kind == profile.KindNexus && strconv.Itoa(entry.Source.ModID) == id {
					return true
				}
			case sourceGitHub:
				if entry.Source.Kind == profile.KindGitHub && entry.Source.Repo == id {
					return true
				}
			}
		}
		return false
	}
}
