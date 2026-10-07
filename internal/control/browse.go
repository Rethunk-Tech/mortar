package control

import (
	"cmp"
	"context"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/browse"
)

// Browse searches a game's mod sources (or all of them) for mods of game and marks hits already on profileID.
func (s *Services) Browse(ctx context.Context, game, source, text string, page int, profileID string) (browse.Page, error) {
	c := &browse.Client{Version: s.Version}
	if profileID != "" && s.Profiles != nil {
		if prof, err := s.resolve(game, profileID); err == nil {
			c.Installed = browse.InstalledOn(prof)
		}
	}
	return c.Search(ctx, game, source, text, page, browse.Filter{})
}

func (s *Services) browseFromParams(ctx context.Context, p Params) (browse.Page, error) {
	text := strings.TrimSpace(p.Query)
	text = cmp.Or(text, strings.TrimSpace(p.Name))
	source := strings.TrimSpace(p.Source)
	source = cmp.Or(source, browse.AllSources)
	page := defaultPage
	if p.Page > 0 {
		page = p.Page
	}
	return s.Browse(ctx, p.Game, source, text, page, p.Profile)
}

const defaultPage = 1
