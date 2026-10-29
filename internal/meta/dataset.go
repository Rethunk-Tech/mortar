package meta

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/go-smapiweb"
)

const datasetTTL = 30 * 24 * time.Hour

type indexMemo struct {
	mu      sync.Mutex
	fetched time.Time
	byID    map[string][]smapiweb.Ref
}

// Lookup returns the pages that ship uniqueID, matched ignoring case. It is nil for an unknown ID; the error
// is only set when no index could be loaded at all.
func (c *Client) Lookup(ctx context.Context, uniqueID string) ([]smapiweb.Ref, error) {
	byID, err := c.loadIndex(ctx)
	if err != nil {
		return nil, err
	}
	return byID[strings.ToLower(uniqueID)], nil
}

// Index returns the whole index, keyed by lowercased UniqueID. Callers must not modify it.
func (c *Client) Index(ctx context.Context) (map[string][]smapiweb.Ref, error) {
	return c.loadIndex(ctx)
}

func (c *Client) loadIndex(ctx context.Context) (map[string][]smapiweb.Ref, error) {
	c.index.mu.Lock()
	defer c.index.mu.Unlock()
	if c.index.byID != nil && c.now().Sub(c.index.fetched) < datasetTTL {
		return c.index.byID, nil
	}
	byID, err := diskcache.Cached(c.Cache(), DatasetPrefix+"index.json", datasetTTL, func() (map[string][]smapiweb.Ref, error) { return c.web().Index(ctx) })
	if err != nil {
		return nil, err
	}
	c.index.byID, c.index.fetched = byID, c.now()
	return byID, nil
}

// Page returns Nexus mod page id, from cache while younger than a month.
func (c *Client) Page(ctx context.Context, id int) (smapiweb.Page, error) {
	p, err := c.page(ctx, id)
	// The dataset gives newer files by their storage path ("4a/35/fc/<uuid>"), which names nothing a user or
	// Nexus would recognise. Cleaning on read also covers pages cached before this rule.
	for i := range p.Downloads {
		if strings.ContainsAny(p.Downloads[i].FileName, `/\`) {
			p.Downloads[i].FileName = ""
		}
	}
	return p, err
}

func (c *Client) page(ctx context.Context, id int) (smapiweb.Page, error) {
	return diskcache.Cached(c.Cache(), DatasetPrefix+"nexus-"+strconv.Itoa(id)+".json", datasetTTL, func() (smapiweb.Page, error) {
		return c.web().Page(ctx, id)
	})
}
