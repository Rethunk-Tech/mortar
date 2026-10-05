// Package browse searches a game's mod sources.
package browse

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
)

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
}

// Search returns one page of mods for game from the source matching text.
func (c *Client) Search(ctx context.Context, game, sourceID, text string, page int) (Page, error) {
	info, ok := catalogGame(game)
	if !ok {
		return Page{}, fmt.Errorf("%w: game %q", ErrUnknownSource, game)
	}
	return c.search(ctx, info, sourceID, text, page)
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
