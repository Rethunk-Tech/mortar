// Package browse searches a game's mod sources.
package browse

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-smapiweb"

	"github.com/Rethunk-Tech/mortar/internal/components"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/netstate"
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
	// Hidden reports a hit the player hid from Browse; those are dropped from every page.
	Hidden func(Item) bool
	// Prefer lists source ids that rank ahead of the rest of the game's sources, which keep catalog order.
	Prefer []string
	// Off lists source ids the player turned off, which the all-sources search skips.
	Off []string
	// Compat is the game's compatibility list; nil when the game has none, so nothing is marked broken.
	Compat func(ctx context.Context) (smapiweb.CompatIndex, error)
	// Identity gives a hit's package identity from what the open profile holds; nil or "" falls back to the
	// compatibility list. Hits that share an identity are one mod.
	Identity func(source, id string) string
}

// identity resolves a hit to its package identity from the profile, then from the compatibility list's refs.
func (c *Client) identity(ctx context.Context) identityOf {
	var idx smapiweb.CompatIndex
	if c.Compat != nil {
		idx, _ = c.Compat(ctx)
	}
	return func(it Item) string {
		if c.Identity != nil {
			if id := c.Identity(it.Source, it.ID); id != "" {
				return id
			}
		}
		var uid string
		switch it.Source {
		case "nexus":
			if n, err := strconv.Atoi(it.ID); err == nil {
				uid = idx.NexusToID[n]
			}
		case "github":
			uid = idx.GitHubToID[strings.ToLower(it.ID)]
		}
		if uid == "" {
			return ""
		}
		return mod.SMAPI(uid).Fold()
	}
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

// searchAll serves one page across every searchable source. Sorted by one of MergedSorts it is one ranked list
// (searchRanked); by best match, whose scores do not compare across sites, it asks each source for the same page and
// interleaves the answers, so each keeps its own ranking. Sources that fail, and parts of a source that did not answer,
// are named in Failed; only when all fail is it an error.
func (c *Client) searchAll(ctx context.Context, info components.GameInfo, text string, page int, f Filter) (Page, error) {
	sources := slices.DeleteFunc(source.Searchable(info), func(s source.Source) bool {
		return source.Unavailable(s) != "" || slices.Contains(c.Off, s.ID())
	})
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
	if slices.Contains(MergedSorts, f.Sort) && slices.ContainsFunc(sources, func(s source.Source) bool {
		return slices.Contains(source.SortsOf(s), f.Sort)
	}) {
		return c.searchRanked(ctx, info, sources, text, page, f)
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
			merged.Failed = append(merged.Failed, sources[i].ID())
			continue
		}
		answered = append(answered, pages[i])
		for _, name := range pages[i].Failed {
			if !slices.Contains(merged.Failed, name) {
				merged.Failed = append(merged.Failed, name)
			}
		}
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
	items := interleave(answered)
	merged.Items = mergeSame(items, ranked(c.Prefer, ids), c.identity(ctx), aliasesOf(components.Authors()))
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

// topTTL is how long a source's answer is reused, so opening Browse costs each source one request and page N of a
// search rereads its earlier pages from here.
const topTTL = 10 * time.Minute

// topCap is the most source pages kept: a walk through a few sources' pages, the least recently fetched going first.
const topCap = 256

type topEntry struct {
	page  Page
	until time.Time
}

var (
	topMu    sync.Mutex
	topCache = map[string]topEntry{}
)

// RefreshedEvent tells the window that a source's data changed under the answers it shows; the payload is the game's
// community key.
const RefreshedEvent = "browse:refreshed"

// Forget drops what is remembered of the source's answers (pages and where each visible page ended), for when the
// source's own data changed under them: a search then reads it again.
func Forget(id string) { forgetWhere(func(parts []string) bool { return parts[0] == id }) }

// forgetGame drops what is remembered of every source's answers for the game.
func forgetGame(game string) { forgetWhere(func(parts []string) bool { return parts[1] == game }) }

// forgetWhere drops the remembered answers whose key (source id, game, ...) the match accepts.
func forgetWhere(match func(parts []string) bool) {
	drop := func(k string) bool { return match(strings.SplitN(k, "|", 3)) }
	topMu.Lock()
	maps.DeleteFunc(topCache, func(k string, _ topEntry) bool { return drop(k) })
	topMu.Unlock()
	cursorMu.Lock()
	maps.DeleteFunc(cursors, func(k string, _ cursorEntry) bool { return drop(k) })
	cursorMu.Unlock()
}

// searchCached runs the source's search, reusing its answer for topTTL. It returns a copy callers may edit.
func searchCached(ctx context.Context, id string, s source.Searcher, q source.Query) (Page, error) {
	key := fmt.Sprintf("%s|%d", queryKey(id, q), q.Page)
	topMu.Lock()
	e, ok := topCache[key]
	topMu.Unlock()
	if !ok || !time.Now().Before(e.until) {
		page, err := s.Search(ctx, q)
		netstate.Record(id, err)
		if err != nil {
			return Page{}, err
		}
		if len(page.Failed) > 0 {
			// A partial answer is not kept: the next look asks the failed part again.
			page.Items = slices.Clone(page.Items)
			return page, nil
		}
		now := time.Now()
		e = topEntry{page: page, until: now.Add(topTTL)}
		topMu.Lock()
		maps.DeleteFunc(topCache, func(_ string, old topEntry) bool { return !now.Before(old.until) })
		topCache[key] = e
		for len(topCache) > topCap {
			oldest := ""
			for k, v := range topCache {
				if oldest == "" || v.until.Before(topCache[oldest].until) {
					oldest = k
				}
			}
			delete(topCache, oldest)
		}
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
	q := source.Query{
		Game: info.ID, Key: gs.Key, Text: text, Version: c.Version,
		Categories: f.Include, ExcludeCategories: f.Exclude, Sort: f.Sort,
	}
	return c.searchFilled(ctx, info, id, searcher, q, max(page, source.FirstPage), f)
}

// maxScan is the most upstream pages one visible page reads: a filter that hides nearly everything yields a short
// page instead of a loop.
const maxScan = 10

// cursorCap is the most queries whose page ends are remembered, the least recently used going first.
const cursorCap = 64

// upPos is a place in a query's upstream stream: an upstream page and how many of its hits left after the drops come
// before it.
type upPos struct{ page, offset int }

type cursorEntry struct {
	ends  map[int]upPos
	until time.Time
}

var (
	cursorMu sync.Mutex
	// cursors remembers, per query and filter, where each served visible page ended upstream, so page N continues
	// from page N-1 without reading the pages before it.
	cursors = map[string]cursorEntry{}
)

func queryKey(id string, q source.Query) string {
	return fmt.Sprintf("%s|%s|%s|%d|%s|%q|%q|%q", id, q.Game, q.Key, q.Len(), q.Sort, q.Categories, q.ExcludeCategories, strings.TrimSpace(q.Text))
}

func cursorKey(id string, q source.Query, f Filter, showAdult bool) string {
	return fmt.Sprintf("%s|%s|%s|%s|%t", queryKey(id, q), f.Installed, f.Obsolete, f.Broken, showAdult)
}

func cursorEnd(key string, page int) (upPos, bool) {
	cursorMu.Lock()
	defer cursorMu.Unlock()
	e, ok := cursors[key]
	if !ok || !time.Now().Before(e.until) {
		return upPos{}, false
	}
	pos, ok := e.ends[page]
	return pos, ok
}

// nearestEnd is the latest remembered end of a visible page from below down to floor, with the page it ends.
func nearestEnd(key string, below, floor int) (int, upPos, bool) {
	cursorMu.Lock()
	defer cursorMu.Unlock()
	e, ok := cursors[key]
	if !ok || !time.Now().Before(e.until) {
		return 0, upPos{}, false
	}
	for page := below; page >= max(floor, source.FirstPage); page-- {
		if pos, ok := e.ends[page]; ok {
			return page, pos, true
		}
	}
	return 0, upPos{}, false
}

func storeEnds(key string, ends map[int]upPos) {
	if len(ends) == 0 {
		return
	}
	cursorMu.Lock()
	defer cursorMu.Unlock()
	now := time.Now()
	e, ok := cursors[key]
	if !ok || !now.Before(e.until) {
		e = cursorEntry{ends: map[int]upPos{}}
	}
	maps.Copy(e.ends, ends)
	e.until = now.Add(topTTL)
	cursors[key] = e
	maps.DeleteFunc(cursors, func(_ string, old cursorEntry) bool { return !now.Before(old.until) })
	for len(cursors) > cursorCap {
		oldest := ""
		for k, v := range cursors {
			if oldest == "" || v.until.Before(cursors[oldest].until) {
				oldest = k
			}
		}
		delete(cursors, oldest)
	}
}

// searchFilled serves page `first` of the visible stream, the upstream hits left after the drops, so each page is
// full unless the upstream ends or maxScan upstream pages are read, and consecutive pages neither repeat nor skip a
// hit. Page N starts where page N-1 ended when that is remembered; a page within maxScan is counted from the start;
// a deeper cold jump starts at upstream page N, so the drops before it shift it slightly.
func (c *Client) searchFilled(ctx context.Context, info components.GameInfo, id string, s source.Searcher, q source.Query, first int, f Filter) (Page, error) {
	key := cursorKey(id, q, f, c.ShowAdult)
	size := source.PageSize
	// pos is where reading starts, numbered the visible hit at it.
	pos, numbered := upPos{page: source.FirstPage}, 0
	// guessed is a start taken without a remembered end; the page ends it reads are then not remembered either.
	guessed := false
	switch end, ok := cursorEnd(key, first-1); {
	case first == source.FirstPage:
	case ok:
		pos, numbered = end, (first-1)*size
	default:
		// The window asks for a screen's pages at once, so the page before may still be reading: an earlier page's end
		// within the scan is exact, where taking upstream page N as visible page N repeats or skips the hits dropped
		// before it.
		if page, end, ok := nearestEnd(key, first-1, first-maxScan); ok {
			pos, numbered = end, page*size
		} else if first > maxScan {
			pos, numbered, guessed = upPos{page: first}, (first-1)*size, true
		}
	}
	from, want := (first-1)*size, first*size
	out := Page{Items: []Item{}}
	ends := map[int]upPos{}
	dropped, ended := 0, false
	for p := pos.page; p < pos.page+maxScan && numbered < want; p++ {
		q.Page = p
		raw, total, last, err := c.fetchMarked(ctx, info, id, s, q)
		if err != nil {
			return Page{}, err
		}
		dropped += total - raw.Total
		c.applyModes(&raw, f)
		out.Total = max(out.Total, total)
		out.Hidden += raw.Hidden
		for _, name := range raw.Failed {
			if !slices.Contains(out.Failed, name) {
				out.Failed = append(out.Failed, name)
			}
		}
		skip := 0
		if p == pos.page {
			skip = min(pos.offset, len(raw.Items))
		}
		for i := skip; i < len(raw.Items) && numbered < want; i++ {
			if numbered >= from {
				out.Items = append(out.Items, raw.Items[i])
			}
			numbered++
			if numbered%size == 0 {
				ends[numbered/size] = upPos{page: p, offset: i + 1}
			}
		}
		if last {
			ended = true
			break
		}
	}
	// The upstream's own total keeps the page count the same on every page and lets a source that only knows there is
	// more offer the next page; a read that saw every hit from the first knows exactly how many it dropped.
	if ended && pos == (upPos{page: source.FirstPage}) {
		out.Total = max(out.Total-dropped, 0)
	}
	if !guessed {
		storeEnds(key, ends)
	}
	log.Printf("browse: %s page %d (sort %q): %d hits from upstream pages %d to %d (resumed at %d+%d), %d hidden, %d dropped, total %d",
		id, first, q.Sort, len(out.Items), pos.page, q.Page, pos.page, pos.offset, out.Hidden, dropped, out.Total)
	return out, nil
}

// fetchMarked reads one upstream page, dropping adult and hidden hits and marking each hit; the page's Total leaves out
// its own drops. total is the upstream's own count, and last reports that the upstream has no later page.
func (c *Client) fetchMarked(ctx context.Context, info components.GameInfo, id string, s source.Searcher, q source.Query) (result Page, total int, last bool, err error) {
	result, err = searchCached(ctx, id, s, q)
	if err != nil {
		return Page{}, 0, false, err
	}
	total = result.Total
	last = total <= q.Page*q.Len()
	before := len(result.Items)
	if !c.ShowAdult {
		result.Items = slices.DeleteFunc(result.Items, func(i Item) bool { return i.Adult })
	}
	if c.Hidden != nil {
		result.Items = slices.DeleteFunc(result.Items, c.Hidden)
	}
	result.Total = max(total-(before-len(result.Items)), 0)
	c.markInstalled(result.Items)
	c.mark(ctx, info, result.Items)
	return result, total, last, nil
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
// AllSources. Answers are cached for a day; a source that fails is skipped, and the error comes back only when no
// source gave any, so the pickers can say why they are empty.
func (c *Client) Categories(ctx context.Context, game, sourceID string) ([]string, error) {
	info, ok := catalogGame(game)
	if !ok {
		return nil, fmt.Errorf("%w: game %q", ErrUnknownSource, game)
	}
	sources := source.Searchable(info)
	if id := strings.ToLower(strings.TrimSpace(sourceID)); id != AllSources {
		sources = slices.DeleteFunc(sources, func(s source.Source) bool { return s.ID() != id })
	} else {
		sources = slices.DeleteFunc(sources, func(s source.Source) bool { return slices.Contains(c.Off, s.ID()) })
	}
	return categoriesOf(ctx, info, sources)
}

func categoriesOf(ctx context.Context, info components.GameInfo, sources []source.Source) ([]string, error) {
	var all []string
	var errs []error
	for _, s := range sources {
		// A source browse shows as unavailable (no API key in this build) is not searched, so its categories do not apply.
		if source.Unavailable(s) != "" {
			continue
		}
		gs, _ := info.Source(s.ID())
		names, err := categoriesCached(ctx, info.ID, s, gs.Key)
		if err != nil {
			errs = append(errs, err)
		}
		all = append(all, names...)
	}
	if len(all) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return source.UniqueNames(all), nil
}
