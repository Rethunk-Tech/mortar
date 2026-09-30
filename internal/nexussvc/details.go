package nexussvc

import (
	"context"
	"fmt"
	"time"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
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
// (details-v2-… so copies from before newest-first changelogs are not reused), refetching once they are a day
// old. Signed out, rate-limited or offline, it serves what is cached however old, and errors only with nothing.
func (s *Service) Details(ctx context.Context, modID int) (Details, error) {
	return meta.Cached(s.meta, detailsName(modID), detailsTTL, func() (Details, error) {
		c, err := Authed(s.store, s.client)
		if err != nil {
			return Details{}, err
		}
		page, err := c.Page(ctx, modID)
		if err != nil {
			return Details{}, err
		}
		files, err := c.Files(ctx, modID)
		if err != nil {
			return Details{}, err
		}
		logs, err := c.Changelogs(ctx, modID, changelogs)
		if err != nil {
			return Details{}, err
		}
		// The category name is a label only: a failed lookup leaves it out rather than losing the rest.
		cats, _ := meta.Cached(s.meta, "nexus/categories-"+nexus.Game+".json", categoriesTTL, func() (map[int]string, error) {
			return c.Categories(ctx)
		})
		return Details{Page: page, Category: cats[page.CategoryID], Files: files, Changelogs: logs}, nil
	})
}

func detailsName(modID int) string {
	return fmt.Sprintf("nexus/details-v2-%s-%d.json", nexus.Game, modID)
}

// CachedDetails returns whatever details are cached for modIDs, however old, without a network call, so a list of
// many mods can show them without a burst of requests. Uncached mods are absent from the map.
func (s *Service) CachedDetails(modIDs []int) map[int]Details {
	out := make(map[int]Details, len(modIDs))
	for _, id := range modIDs {
		if d, ok := meta.Peek[Details](s.meta, detailsName(id)); ok {
			out[id] = d
		}
	}
	return out
}
