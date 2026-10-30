package nexussvc

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

const (
	detailsTTL    = 24 * time.Hour
	categoriesTTL = 30 * 24 * time.Hour
)

// Details is what the mod detail dialog shows from a Nexus page: the page, its category's name, every file and every
// changelog version (one request however many there are).
type Details struct {
	Page       nexusmods.Page    `json:"page"`
	Category   string            `json:"category"`
	Files      []nexusmods.File  `json:"files"`
	Changelogs []nexus.Changelog `json:"changelogs"`
	// Partial marks details from the batched lookup: the page's headline data only, with no files or changelogs.
	Partial bool `json:"partial,omitempty"`
}

// Details returns a mod page's details from the cache under <datadir>/cache/nexus
// (details-v4-…, named for the shape it holds, so a file of an older shape is not reused), refetching once they are a day
// old. Signed out, rate-limited or offline, it serves what is cached however old, and errors only with nothing.
func (s *Service) Details(ctx context.Context, gameID string, modID int) (Details, error) {
	t, err := game.NexusTitle(gameID)
	if err != nil {
		return Details{}, err
	}
	return diskcache.Cached(s.meta.Cache(), DetailsName(t.Domain, modID), detailsTTL, func() (Details, error) {
		c, err := Authed(s.store, s.client)
		if err != nil {
			return Details{}, err
		}
		page, err := c.Page(ctx, t, modID)
		if err != nil {
			return Details{}, nexus.Worded(err)
		}
		// Requirements are only in the GraphQL data; a failed lookup shows the page without them.
		if infos, err := c.ModsByDomain(ctx, t.Domain, []int{modID}); err == nil {
			page.Requirements = infos[modID].Requirements
		}
		files, err := c.Files(ctx, t, modID)
		if err != nil {
			return Details{}, nexus.Worded(err)
		}
		logs, err := c.Changelogs(ctx, t, modID, math.MaxInt)
		if err != nil {
			return Details{}, nexus.Worded(err)
		}
		// The category name is a label only: a failed lookup leaves it out rather than losing the rest.
		cats, _ := diskcache.Cached(s.meta.Cache(), meta.NexusCategoriesPrefix+t.Domain+".json", categoriesTTL, func() (map[int]string, error) {
			cats, err := c.Categories(ctx, t)
			return cats, nexus.Worded(err)
		})
		return Details{Page: page, Category: cats[page.CategoryID], Files: files, Changelogs: nexusChangelogs(logs)}, nil
	})
}

// DetailsName is the cache file under cache/ for a Nexus mod's details.
func DetailsName(domain string, modID int) string {
	return fmt.Sprintf("%sv4-%s-%d.json", meta.NexusDetailsPrefix, domain, modID)
}

// CachedFiles returns the cached file list of each mod that has one, however old, without a network call.
//
//wails:ignore
func (s *Service) CachedFiles(t nexusmods.Title, modIDs []int) map[int][]nexusmods.BatchFile {
	out := make(map[int][]nexusmods.BatchFile, len(modIDs))
	for _, id := range modIDs {
		d, ok := PeekDetails(s.meta, t.Domain, id)
		if !ok || len(d.Files) == 0 {
			continue
		}
		list := make([]nexusmods.BatchFile, len(d.Files))
		for i, f := range d.Files {
			list[i] = nexusmods.BatchFile{FileID: f.FileID, Name: f.Name, Version: f.Version, Category: f.Category, ReplacedBy: f.ReplacedBy}
		}
		out[id] = list
	}
	return out
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
		if d, ok := PeekDetails(s.meta, t.Domain, id); ok {
			out[id] = d
		}
	}
	return out
}
