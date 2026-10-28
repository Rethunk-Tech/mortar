package browse

import (
	"context"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Service is the window's browse API.
type Service struct {
	Version  string
	Profiles *profile.Service
	// ShowAdult reports the player's "show adult mods" setting.
	ShowAdult func() bool
	// HiddenMod reports whether the game hides the hit, by mod or by author, from Browse.
	HiddenMod func(game string, it Item) bool
	// SourceOrder is the game's preferred source ids, first first; nil means catalog order.
	SourceOrder func(game string) []string
	// SourcesOff is the game's source ids the player turned off; nil means none.
	SourcesOff func(game string) []string
	// Installed reads the manifests of a profile's mods, whose update keys name the pages they came from; nil skips them.
	Installed func(game, profileID string) ([]profile.Installed, error)
	// Compat returns the game's compatibility list lookup, or nil when the game has none.
	Compat func(game string) func(ctx context.Context) (meta.CompatIndex, error)
}

// HasCompat reports whether the game has a compatibility list or a known-broken list, so its mods can be marked
// broken.
func (s *Service) HasCompat(ctx context.Context, game string) bool {
	return (s.Compat != nil && s.Compat(game) != nil) || len(components.KnownBrokenFor(ctx, game).Entries) > 0
}

// NewService returns a Service that marks hits already on the named profile.
func NewService(version string, profiles *profile.Service) *Service {
	return &Service{Version: version, Profiles: profiles}
}

// Search returns one page of mods for game from sourceID matching text.
func (s *Service) Search(ctx context.Context, game, sourceID, text string, page int, profileID string, filter Filter) (Page, error) {
	c := &Client{Version: s.Version, ShowAdult: s.ShowAdult != nil && s.ShowAdult()}
	if s.SourceOrder != nil {
		c.Prefer = s.SourceOrder(game)
	}
	c.Off = s.off(game)
	if s.HiddenMod != nil {
		c.Hidden = func(it Item) bool { return s.HiddenMod(game, it) }
	}
	if s.Compat != nil {
		c.Compat = s.Compat(game)
	}
	if h := s.holdings(game, profileID); h != nil {
		c.Installed, c.Bundled, c.Identity = h.Has, h.Bundled, h.Identity
	}
	return c.Search(ctx, game, sourceID, text, page, filter)
}

// Categories lists the category names of sourceID (or of every source) for the filter pickers.
func (s *Service) Categories(ctx context.Context, game, sourceID string) ([]string, error) {
	return (&Client{Off: s.off(game)}).Categories(ctx, game, sourceID)
}

// DefaultExcludes is the modpack, translation and cheat categories of sourceID (or of every source), which the
// exclude picker starts with.
func (s *Service) DefaultExcludes(ctx context.Context, game, sourceID string) ([]string, error) {
	names, err := (&Client{Off: s.off(game)}).Categories(ctx, game, sourceID)
	return noisyNames(names), err
}

func (s *Service) off(game string) []string {
	if s.SourcesOff == nil {
		return nil
	}
	return SourcesOff(game, s.SourcesOff(game))
}

// SourcesOff is the game's sources the player turned off, or none when that would leave Browse no available source
// to search (a settings edit outside the window can turn every one off).
func SourcesOff(game string, off []string) []string {
	info, ok := catalogGame(game)
	if !ok || len(off) == 0 {
		return off
	}
	for _, src := range source.Searchable(info) {
		if source.Unavailable(src) == "" && !slices.Contains(off, src.ID()) {
			return off
		}
	}
	return nil
}

// SourceInfo names a source for the window's source chips.
type SourceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Unavailable says why the source cannot be searched now (for example a missing API key); empty when it can.
	Unavailable string `json:"unavailable"`
	// Sorts are the Sort values the source orders by on its own server, besides best match.
	Sorts []string `json:"sorts"`
	// Off is true when the player turned the source off for the game.
	Off bool `json:"off"`
}

// SearchableSources lists the game's sources that can be searched, in catalog order.
func (s *Service) SearchableSources(game string) []SourceInfo {
	out := []SourceInfo{}
	off := s.off(game)
	if info, ok := catalogGame(game); ok {
		for _, src := range source.Searchable(info) {
			out = append(out, SourceInfo{ID: src.ID(), Name: src.Name(), Unavailable: source.Unavailable(src), Sorts: source.SortsOf(src), Off: slices.Contains(off, src.ID())})
		}
	}
	return out
}

func (s *Service) holdings(game, profileID string) *Holdings {
	if s == nil || s.Profiles == nil || strings.TrimSpace(profileID) == "" {
		return nil
	}
	list, err := s.Profiles.List(game)
	if err != nil {
		return nil
	}
	info, _ := catalogGame(game)
	for _, prof := range list {
		if prof.ID == profileID || strings.EqualFold(prof.Name, profileID) {
			var installed []profile.Installed
			if s.Installed != nil {
				installed, _ = s.Installed(game, prof.ID)
			}
			h := Hold(info, prof, installed)
			if id := s.Profiles.InstalledLoader(game, prof.ID); id != "" {
				h.holdLoader(info, id)
			}
			return &h
		}
	}
	return nil
}
