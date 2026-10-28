package control

import (
	"cmp"
	"context"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/browse"
)

func (s *Services) browseFromParams(ctx context.Context, p Params) (browse.Page, error) {
	text := strings.TrimSpace(p.Query)
	text = cmp.Or(text, strings.TrimSpace(p.Name))
	source := strings.TrimSpace(p.Source)
	source = cmp.Or(source, browse.AllSources)
	page := defaultPage
	if p.Page > 0 {
		page = p.Page
	}
	return s.Browse.Search(ctx, p.Game, source, text, page, p.Profile, browse.Filter{})
}

const defaultPage = 1
