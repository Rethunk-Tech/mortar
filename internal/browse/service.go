package browse

import (
	"context"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Service is the window's browse API.
type Service struct {
	Version  string
	Profiles *profile.Service
	// ShowAdult reports the player's "show adult mods" setting.
	ShowAdult func() bool
	// SourceOrder is the game's preferred source ids, first first; nil means catalog order.
	SourceOrder func(game string) []string
}

// NewService returns a Service that marks hits already on the named profile.
func NewService(version string, profiles *profile.Service) *Service {
	return &Service{Version: version, Profiles: profiles}
}

// Search returns one page of mods for game from sourceID matching text.
func (s *Service) Search(ctx context.Context, game, sourceID, text string, page int, profileID string, filter Filter) (Page, error) {
	c := &Client{Version: s.Version, Installed: s.installed(game, profileID), ShowAdult: s.ShowAdult != nil && s.ShowAdult()}
	if s.SourceOrder != nil {
		c.Prefer = s.SourceOrder(game)
	}
	return c.Search(ctx, game, sourceID, text, page, filter)
}

// Categories lists the category names of sourceID (or of every source) for the filter pickers.
func (s *Service) Categories(ctx context.Context, game, sourceID string) ([]string, error) {
	return (&Client{}).Categories(ctx, game, sourceID)
}

// SourceInfo names a source for the window's source chips.
type SourceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Unavailable says why the source cannot be searched now (for example a missing API key); empty when it can.
	Unavailable string `json:"unavailable"`
}

// SearchableSources lists the game's sources that can be searched, in catalog order.
func (s *Service) SearchableSources(game string) []SourceInfo {
	out := []SourceInfo{}
	if info, ok := catalogGame(game); ok {
		for _, src := range source.Searchable(info) {
			out = append(out, SourceInfo{ID: src.ID(), Name: src.Name(), Unavailable: source.Unavailable(src)})
		}
	}
	return out
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

// InstalledOn reports Nexus mod ids and GitHub repos already on prof, by the entry's source kind.
func InstalledOn(prof profile.Profile) InstalledFunc {
	return func(source, id string) bool {
		for _, entry := range prof.Entries {
			switch entry.Source.Kind {
			case profile.KindNexus:
				if source == "nexus" && strconv.Itoa(entry.Source.ModID) == id {
					return true
				}
			case profile.KindThunderstore:
				if source == "thunderstore" && strings.EqualFold(entry.Source.Name, id) {
					return true
				}
			case profile.KindGitHub:
				if source == "github" && entry.Source.Repo == id {
					return true
				}
			}
		}
		return false
	}
}
