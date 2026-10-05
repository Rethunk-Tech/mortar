// Package browse searches a game's mod sources.
package browse

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
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

// Filter narrows and orders a search. Include keeps mods in any of the named categories, Exclude drops mods in any of
// them, and Sort is one of the source package's Sort values; zero means no narrowing and each source's own order.
type Filter struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
	Sort    string   `json:"sort"`
	// Installed, Obsolete and Broken say what to do with mods in the open profile, marked obsolete, or marked broken:
	// ModeOff (or empty) keeps them unmarked, ModeGray keeps them for the window to dim, ModeHide leaves them out.
	Installed string `json:"installed"`
	Obsolete  string `json:"obsolete"`
	Broken    string `json:"broken"`
}

// What a Filter does with a kind of mod.
const (
	ModeOff  = "off"
	ModeGray = "gray"
	ModeHide = "hide"
)

// Client searches a game's sources. Version names Mortar to the sites.
type Client struct {
	Version   string
	Installed InstalledFunc
	// Bundled reports a hit Mortar installs itself, such as a loader companion; those are Installed too.
	Bundled InstalledFunc
	// ShowAdult keeps hits their source flags as adult content; by default they are dropped.
	ShowAdult bool
	// Prefer lists source ids that rank ahead of the rest of the game's sources, which keep catalog order.
	Prefer []string
	// Compat is the game's compatibility list; nil when the game has none, so nothing is marked broken.
	Compat func(ctx context.Context) (meta.CompatIndex, error)
}

// Search returns one page of mods for game from the source matching text.
func (c *Client) Search(ctx context.Context, game, sourceID, text string, page int, f Filter) (Page, error) {
	info, ok := catalogGame(game)
	if !ok {
		return Page{}, fmt.Errorf("%w: game %q", ErrUnknownSource, game)
	}
	if strings.EqualFold(strings.TrimSpace(sourceID), AllSources) {
		return c.searchAll(ctx, info, text, page, f)
	}
	return c.search(ctx, info, sourceID, text, page, f)
}

// searchAll asks every searchable source for the same page at once and interleaves the answers, so each source keeps
// its own ranking and none crowds out the rest. Sources that fail are named in Failed; only when all fail is it an
// error.
func (c *Client) searchAll(ctx context.Context, info components.GameInfo, text string, page int, f Filter) (Page, error) {
	sources := slices.DeleteFunc(source.Searchable(info), func(s source.Source) bool { return source.Unavailable(s) != "" })
	if len(f.Include) > 0 {
		// A source with no categories cannot match an include list.
		sources = slices.DeleteFunc(sources, func(s source.Source) bool {
			_, ok := s.(source.Categorizer)
			return !ok
		})
	}
	if len(sources) == 0 {
		return Page{}, fmt.Errorf("%w: %s has no searchable source", ErrUnknownSource, info.ID)
	}
	pages := make([]Page, len(sources))
	errs := make([]error, len(sources))
	var wg sync.WaitGroup
	for i, s := range sources {
		wg.Go(func() { pages[i], errs[i] = c.search(ctx, info, s.ID(), text, page, f) })
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
		merged.Hidden += pages[i].Hidden
		merged.Pages = max(merged.Pages, (pages[i].Total+source.PageSize-1)/source.PageSize)
	}
	if len(answered) == 0 {
		return Page{}, errors.Join(errs...)
	}
	ids := make([]string, len(sources))
	for i, s := range sources {
		ids[i] = s.ID()
	}
	merged.Items = mergeSame(interleave(answered), ranked(c.Prefer, ids))
	return merged, nil
}

// ranked puts the preferred ids that are among ids first, then the rest in their given order.
func ranked(prefer, ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, p := range prefer {
		if slices.Contains(ids, p) && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	for _, id := range ids {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
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

// topTTL is how long an empty-text (top mods) answer is reused, so opening Browse costs each source one request.
const topTTL = 10 * time.Minute

type topEntry struct {
	page  Page
	until time.Time
}

var (
	topMu    sync.Mutex
	topCache = map[string]topEntry{}
)

// searchCached runs the source's search, reusing an empty-text answer for topTTL. It returns a copy callers may edit.
func searchCached(ctx context.Context, s source.Searcher, q source.Query) (Page, error) {
	if strings.TrimSpace(q.Text) != "" {
		return s.Search(ctx, q)
	}
	key := fmt.Sprintf("%T|%s|%s|%d|%s|%q|%q", s, q.Game, q.Key, q.Page, q.Sort, q.Categories, q.ExcludeCategories)
	topMu.Lock()
	e, ok := topCache[key]
	topMu.Unlock()
	if !ok || !time.Now().Before(e.until) {
		page, err := s.Search(ctx, q)
		if err != nil {
			return Page{}, err
		}
		e = topEntry{page: page, until: time.Now().Add(topTTL)}
		topMu.Lock()
		topCache[key] = e
		topMu.Unlock()
	}
	out := e.page
	out.Items = slices.Clone(out.Items)
	return out, nil
}

func (c *Client) search(ctx context.Context, info components.GameInfo, sourceID, text string, page int, f Filter) (Page, error) {
	id := strings.ToLower(strings.TrimSpace(sourceID))
	gs, listed := info.Source(id)
	entry, registered := source.Get(id)
	searcher, canSearch := entry.Source.(source.Searcher)
	if !listed || !registered || !canSearch {
		return Page{}, fmt.Errorf("%w: %s", ErrUnknownSource, sourceID)
	}
	result, err := searchCached(ctx, searcher, source.Query{
		Game: info.ID, Key: gs.Key, Text: text, Page: max(page, source.FirstPage), Version: c.Version,
		Categories: f.Include, ExcludeCategories: f.Exclude, Sort: f.Sort,
	})
	if err != nil {
		return Page{}, err
	}
	if !c.ShowAdult {
		before := len(result.Items)
		result.Items = slices.DeleteFunc(result.Items, func(i Item) bool { return i.Adult })
		result.Total = max(result.Total-(before-len(result.Items)), 0)
	}
	c.markInstalled(result.Items)
	c.mark(ctx, info, result.Items)
	c.applyModes(&result, f)
	return result, nil
}

func (c *Client) markInstalled(items []Item) {
	if c.Installed == nil {
		return
	}
	for i := range items {
		items[i].Bundled = c.Bundled != nil && c.Bundled(items[i].Source, items[i].ID)
		items[i].Installed = items[i].Bundled || c.Installed(items[i].Source, items[i].ID)
	}
}

// categoryTTL is how long a source's category list is reused.
const categoryTTL = 24 * time.Hour

var (
	catMu    sync.Mutex
	catCache = map[string]struct {
		names []string
		until time.Time
	}{}
)

// Categories lists the category names of one source, or of every source that has categories when sourceID is
// AllSources. Answers are cached for a day; a source that fails is skipped.
func (c *Client) Categories(ctx context.Context, game, sourceID string) ([]string, error) {
	info, ok := catalogGame(game)
	if !ok {
		return nil, fmt.Errorf("%w: game %q", ErrUnknownSource, game)
	}
	sources := source.Searchable(info)
	if id := strings.ToLower(strings.TrimSpace(sourceID)); id != AllSources {
		sources = slices.DeleteFunc(sources, func(s source.Source) bool { return s.ID() != id })
	}
	var all []string
	for _, s := range sources {
		cat, ok := s.(source.Categorizer)
		if !ok {
			continue
		}
		key := info.ID + "|" + s.ID()
		gs, _ := info.Source(s.ID())
		catMu.Lock()
		e, hit := catCache[key]
		catMu.Unlock()
		if !hit || !time.Now().Before(e.until) {
			names, err := cat.Categories(ctx, gs.Key)
			if err != nil {
				continue
			}
			e.names, e.until = names, time.Now().Add(categoryTTL)
			catMu.Lock()
			catCache[key] = e
			catMu.Unlock()
		}
		all = append(all, e.names...)
	}
	return source.UniqueNames(all), nil
}
