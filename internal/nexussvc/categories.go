package nexussvc

import (
	"context"
	"maps"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

// CategoryNames lists a game's mod category names from the same cached lookup Details uses. It needs a signed-in
// account; signed out it serves whatever is cached, else errors.
func (s *Service) CategoryNames(ctx context.Context, domain string) ([]string, error) {
	cats, err := meta.Cached(s.meta, meta.NexusCategoriesPrefix+domain+".json", categoriesTTL, func() (map[int]string, error) {
		c, err := Authed(s.store, s.client)
		if err != nil {
			return nil, err
		}
		return c.Categories(ctx, nexus.Title{Domain: domain})
	})
	if err != nil {
		return nil, err
	}
	return slices.Collect(maps.Values(cats)), nil
}
