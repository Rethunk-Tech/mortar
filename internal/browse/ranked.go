package browse

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// lane is one source's hits for a ranked search, in that source's own order for the sort, read page by page.
type lane struct {
	id     string
	s      source.Searcher
	q      source.Query
	hits   []Item
	read   int
	total  int
	hidden int
	failed []string
	err    error
	done   bool
}

// searchRanked serves page `page` of every source's hits as one list ordered by f.Sort. Each source already returns its
// pages in that order, so a hit is placed once no source that is still being read could have a better one after its
// last hit read, and reading stops when page `page` is full of placed cards. A source is read in its largest pages (Sizer),
// at most (N+maxScan)×PageSize hits deep for page N. Sources that cannot order by f.Sort are named in Unsorted and left out, since their hits have no place in it.
// ponytail: each request reads pages 1..k of every source again (empty-text pages come from topCache); a per-query
// cursor if paging deep into a text search gets slow.
func (c *Client) searchRanked(ctx context.Context, info components.GameInfo, sources []source.Source, text string, page int, f Filter) (Page, error) {
	out := Page{Items: []Item{}}
	var lanes []*lane
	for _, s := range sources {
		if !slices.Contains(source.SortsOf(s), f.Sort) {
			out.Unsorted = append(out.Unsorted, s.ID())
			continue
		}
		searcher, ok := s.(source.Searcher)
		if !ok {
			continue
		}
		gs, _ := info.Source(s.ID())
		// One request of a source's largest page usually holds every hit of it that page 1 needs.
		size := source.PageSize
		if sz, ok := s.(source.Sizer); ok {
			size = max(sz.MaxPageSize(), size)
		}
		lanes = append(lanes, &lane{id: s.ID(), s: searcher, q: source.Query{
			Game: info.ID, Key: gs.Key, Text: text, Version: c.Version,
			Categories: f.Include, ExcludeCategories: f.Exclude, Sort: f.Sort, Size: size,
		}})
	}
	page = max(page, source.FirstPage)
	want := page * source.PageSize
	order := make([]string, len(lanes))
	for i, l := range lanes {
		order[i] = l.id
	}
	order = ranked(c.Prefer, order)
	ident, people := c.identity(ctx), aliasesOf(components.Authors())
	compare := byMergedSort(f.Sort)
	var cards []Item
	for {
		var window []Item
		for _, l := range lanes {
			window = append(window, l.hits...)
		}
		slices.SortStableFunc(window, compare)
		placed := placedPrefix(window, lanes, compare)
		cards = mergeSame(slices.Clone(placed), order, ident, people)
		var next []*lane
		for _, l := range lanes {
			if l.done {
				continue
			}
			if len(l.hits) == 0 || len(window) < want || compare(l.hits[len(l.hits)-1], window[want-1]) < 0 {
				next = append(next, l)
			}
		}
		if len(cards) >= want || len(next) == 0 {
			break
		}
		var wg sync.WaitGroup
		for _, l := range next {
			wg.Go(func() { c.readNext(ctx, info, l, f, ((page+maxScan)*source.PageSize+l.q.Len()-1)/l.q.Len()) })
		}
		wg.Wait()
	}
	var errs []error
	for _, l := range lanes {
		out.Total += l.total
		out.Hidden += l.hidden
		if l.err != nil {
			out.Failed = append(out.Failed, l.id)
			errs = append(errs, l.err)
		}
		for _, name := range l.failed {
			if !slices.Contains(out.Failed, name) {
				out.Failed = append(out.Failed, name)
			}
		}
	}
	if len(errs) == len(lanes) {
		return Page{}, errors.Join(errs...)
	}
	out.Pages = (out.Total + source.PageSize - 1) / source.PageSize
	out.Items = cards[min((page-1)*source.PageSize, len(cards)):min(want, len(cards))]
	return out, nil
}

// placedPrefix is how much of the sorted window is final: the hits that rank at least as high as the last hit read
// from every source still being read, since each source's later hits rank no higher than its last one. A source still
// being read with nothing read yet leaves nothing placed.
func placedPrefix(window []Item, lanes []*lane, compare func(a, b Item) int) []Item {
	var bound *Item
	for _, l := range lanes {
		if l.done {
			continue
		}
		if len(l.hits) == 0 {
			return nil
		}
		last := l.hits[len(l.hits)-1]
		if bound == nil || compare(last, *bound) < 0 {
			bound = &last
		}
	}
	if bound == nil {
		return window
	}
	n := 0
	for n < len(window) && compare(window[n], *bound) <= 0 {
		n++
	}
	return window[:n]
}

// readNext reads a lane's next upstream page, dropping the hits the filter hides; a lane ends at its source's last
// page, at limit pages of its size, or when its source fails, keeping what it already read.
func (c *Client) readNext(ctx context.Context, info components.GameInfo, l *lane, f Filter, limit int) {
	l.q.Page = source.FirstPage + l.read
	l.read++
	raw, err := c.fetchMarked(ctx, info, l.id, l.s, l.q)
	if err != nil {
		l.err, l.done = err, true
		return
	}
	c.applyModes(&raw, f)
	l.total = raw.Total
	l.hidden += raw.Hidden
	l.hits = append(l.hits, raw.Items...)
	l.failed = append(l.failed, raw.Failed...)
	l.done = raw.Total <= l.read*l.q.Len() || l.read >= limit || len(raw.Items)+raw.Hidden == 0
}
