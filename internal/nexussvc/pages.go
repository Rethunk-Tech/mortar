package nexussvc

import (
	"context"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// PageName is the cache file under cache/ for a Nexus mod's page data from the batched lookup.
func PageName(domain string, modID int) string {
	return fmt.Sprintf("nexus/page-v1-%s-%d.json", domain, modID)
}

// PrimeDetails fills the page data of many mods at once: whatever is cached and fresh is kept, and the rest comes
// from one GraphQL request per 100 mods, so a list of hundreds costs a handful of requests instead of several each.
// It returns the best details held for each mod, which are partial (the page's headline data, no files or
// changelogs) unless the full details were cached; Details fills in the rest when a mod is opened. A rate limit or
// a signed-out account ends the lookup and returns what the cache has with the error.
func (s *Service) PrimeDetails(ctx context.Context, gameID string, modIDs []int) (map[int]Details, error) {
	t, err := game.NexusTitle(gameID)
	if err != nil {
		return nil, err
	}
	var missing []int
	seen := map[int]bool{}
	for _, id := range modIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		if !meta.Fresh[Details](s.meta, DetailsName(t.Domain, id), detailsTTL) && !meta.Fresh[Details](s.meta, PageName(t.Domain, id), detailsTTL) {
			missing = append(missing, id)
		}
	}
	var fetchErr error
	if len(missing) > 0 {
		fetchErr = s.fetchPages(ctx, t.Domain, missing)
	}
	out := make(map[int]Details, len(seen))
	for id := range seen {
		if d, ok := meta.Peek[Details](s.meta, DetailsName(t.Domain, id)); ok {
			out[id] = d
		} else if d, ok := meta.Peek[Details](s.meta, PageName(t.Domain, id)); ok {
			out[id] = d
		}
	}
	return out, fetchErr
}

func (s *Service) fetchPages(ctx context.Context, domain string, ids []int) error {
	c, err := Authed(s.store, s.client)
	if err != nil {
		return err
	}
	infos, err := c.ModsByDomain(ctx, domain, ids)
	for id, info := range infos {
		meta.Put(s.meta, PageName(domain, id), Details{Page: info.Page(), Category: info.Category, Partial: true})
	}
	return err
}
