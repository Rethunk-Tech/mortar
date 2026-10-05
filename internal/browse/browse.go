// Package browse searches a game's mod sources.
package browse

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/components"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
)

// AllSources is the source id that searches every searchable source of the game at once.
const AllSources = "all"

// ErrUnknownSource means the game has no searchable source with that id.
var ErrUnknownSource = errors.New("unknown browse source")

// Item is one search hit.
type Item = source.Item

// Page is one slice of search hits.
type Page = source.Page

// InstalledFunc reports whether the open profile already has this mod (a Nexus mod id or a GitHub repo).
type InstalledFunc func(source, id string) bool

// Client searches a game's sources. Version names Mortar to the sites.
type Client struct {
	Version   string
	Installed InstalledFunc
	// ShowAdult keeps hits their source flags as adult content; by default they are dropped.
	ShowAdult bool
}

// Search returns one page of mods for game from the source matching text.
func (c *Client) Search(ctx context.Context, game, sourceID, text string, page int) (Page, error) {
	info, ok := catalogGame(game)
	if !ok {
		return Page{}, fmt.Errorf("%w: game %q", ErrUnknownSource, game)
	}
	if strings.EqualFold(strings.TrimSpace(sourceID), AllSources) {
		return c.searchAll(ctx, info, text, page)
	}
	return c.search(ctx, info, sourceID, text, page)
}

// searchAll asks every searchable source for the same page at once and interleaves the answers, so each source keeps
// its own ranking and none crowds out the rest. Sources that fail are named in Failed; only when all fail is it an
// error.
func (c *Client) searchAll(ctx context.Context, info components.GameInfo, text string, page int) (Page, error) {
	sources := source.Searchable(info)
	if len(sources) == 0 {
		return Page{}, fmt.Errorf("%w: %s has no searchable source", ErrUnknownSource, info.ID)
	}
	pages := make([]Page, len(sources))
	errs := make([]error, len(sources))
	var wg sync.WaitGroup
	for i, s := range sources {
		wg.Go(func() { pages[i], errs[i] = c.search(ctx, info, s.ID(), text, page) })
	}
	wg.Wait()
	merged := Page{Items: []Item{}}
	var answered []Page
	for i, err := range errs {
		if err != nil {
			merged.Failed = append(merged.Failed, sources[i].Name())
			continue
		}
		answered = append(answered, pages[i])
		merged.Total += pages[i].Total
		merged.Pages = max(merged.Pages, (pages[i].Total+source.PageSize-1)/source.PageSize)
	}
	if len(answered) == 0 {
		return Page{}, errors.Join(errs...)
	}
	merged.Items = interleave(answered)
	return merged, nil
}

func interleave(pages []Page) []Item {
	var out []Item
	for i := 0; ; i++ {
		added := false
		for _, p := range pages {
			if i < len(p.Items) {
				out = append(out, p.Items[i])
				added = true
			}
		}
		if !added {
			return out
		}
	}
}

func catalogGame(id string) (components.GameInfo, bool) {
	catalog := gamepkg.Catalog()
	i := slices.IndexFunc(catalog, func(g components.GameInfo) bool { return g.ID == id })
	if i < 0 {
		return components.GameInfo{}, false
	}
	return catalog[i], true
}

func (c *Client) search(ctx context.Context, info components.GameInfo, sourceID, text string, page int) (Page, error) {
	id := strings.ToLower(strings.TrimSpace(sourceID))
	gs, listed := info.Source(id)
	entry, registered := source.Get(id)
	searcher, canSearch := entry.Source.(source.Searcher)
	if !listed || !registered || !canSearch {
		return Page{}, fmt.Errorf("%w: %s", ErrUnknownSource, sourceID)
	}
	result, err := searcher.Search(ctx, source.Query{Game: info.ID, Key: gs.Key, Text: text, Page: max(page, source.FirstPage), Version: c.Version})
	if err != nil {
		return Page{}, err
	}
	if !c.ShowAdult {
		before := len(result.Items)
		result.Items = slices.DeleteFunc(result.Items, func(i Item) bool { return i.Adult })
		result.Total = max(result.Total-(before-len(result.Items)), 0)
	}
	c.markInstalled(result.Items)
	return result, nil
}

func (c *Client) markInstalled(items []Item) {
	if c.Installed == nil {
		return
	}
	for i := range items {
		items[i].Installed = c.Installed(items[i].Source, items[i].ID)
	}
}
