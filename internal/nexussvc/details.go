package nexussvc

import (
	"context"
	"fmt"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

const (
	detailsTTL    = 24 * time.Hour
	categoriesTTL = 30 * 24 * time.Hour
	changelogs    = 5
)

// Details is what the mod detail dialog shows from a Nexus page: the page, its category's name, every file and the
// newest changelog versions.
type Details struct {
	Page       nexus.Page        `json:"page"`
	Category   string            `json:"category"`
	Files      []nexus.File      `json:"files"`
	Changelogs []nexus.Changelog `json:"changelogs"`
}

// Details returns a mod page's details from the cache under <datadir>/cache/nexus
// (details-v3-… so copies from before endorsement status are not reused), refetching once they are a day
// old. Signed out, rate-limited or offline, it serves what is cached however old, and errors only with nothing.
func (s *Service) Details(ctx context.Context, gameID string, modID int) (Details, error) {
	t, err := game.NexusTitle(gameID)
	if err != nil {
		return Details{}, err
	}
	return meta.Cached(s.meta, DetailsName(t.Domain, modID), detailsTTL, func() (Details, error) {
		c, err := Authed(s.store, s.client)
		if err != nil {
			return Details{}, err
		}
		page, err := c.Page(ctx, t, modID)
		if err != nil {
			return Details{}, err
		}
		files, err := c.Files(ctx, t, modID)
		if err != nil {
			return Details{}, err
		}
		logs, err := c.Changelogs(ctx, t, modID, changelogs)
		if err != nil {
			return Details{}, err
		}
		// The category name is a label only: a failed lookup leaves it out rather than losing the rest.
		cats, _ := meta.Cached(s.meta, "nexus/categories-"+t.Domain+".json", categoriesTTL, func() (map[int]string, error) {
			return c.Categories(ctx, t)
		})
		return Details{Page: page, Category: cats[page.CategoryID], Files: files, Changelogs: logs}, nil
	})
}

// DetailsName is the cache file under cache/ for a Nexus mod's details.
func DetailsName(domain string, modID int) string {
	return fmt.Sprintf("nexus/details-v3-%s-%d.json", domain, modID)
}

// CachedDetails returns whatever details are cached for modIDs, however old, without a network call, so a list of
// many mods can show them without a burst of requests. Uncached mods are absent from the map.
func (s *Service) CachedDetails(gameID string, modIDs []int) map[int]Details {
	out := make(map[int]Details, len(modIDs))
	t, err := game.NexusTitle(gameID)
	if err != nil {
		return out
	}
	for _, id := range modIDs {
		if d, ok := meta.Peek[Details](s.meta, DetailsName(t.Domain, id)); ok {
			out[id] = d
		}
	}
	return out
}
