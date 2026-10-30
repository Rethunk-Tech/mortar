package nexussvc

import (
	"context"
	"maps"
	"slices"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

// CategoryNames lists a game's mod category names from the same cached lookup Details uses. It needs a signed-in
// account; signed out it serves whatever is cached, else errors.
func (s *Service) CategoryNames(ctx context.Context, domain string) ([]string, error) {
	cats, err := diskcache.Cached(s.meta.Cache(), meta.NexusCategoriesPrefix+domain+".json", categoriesTTL, func() (map[int]string, error) {
		c, err := Authed(s.store, s.client)
		if err != nil {
			return nil, err
		}
		cats, err := c.Categories(ctx, nexusmods.Title{Domain: domain})
		return cats, nexus.Worded(err)
	})
	if err != nil {
		return nil, err
	}
	return slices.Collect(maps.Values(cats)), nil
}
