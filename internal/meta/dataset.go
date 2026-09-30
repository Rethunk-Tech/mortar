package meta

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultIndexURL = "https://raw.githubusercontent.com/Pathoschild/StardewModDataset/main/dataset/indexes/pages%20by%20mod%20ID.json"
	defaultPageBase = "https://raw.githubusercontent.com/Pathoschild/StardewModDataset/main/dataset/data/Nexus"
	datasetTTL      = 30 * 24 * time.Hour
	indexFile       = "dataset-index.json"
	maxIndex        = 32 << 20
	maxPage         = 8 << 20
)

// Ref names a mod page: Site is "Nexus", "CurseForge" or "ModDrop".
type Ref struct {
	Site string `json:"site"`
	ID   int    `json:"id"`
}

type indexMemo struct {
	mu      sync.Mutex
	fetched time.Time
	byID    map[string][]Ref
}

// Lookup returns the pages that ship uniqueID, matched ignoring case. It is nil for an unknown ID; the error
// is only set when no index could be loaded at all.
func (c *Client) Lookup(ctx context.Context, uniqueID string) ([]Ref, error) {
	byID, err := c.loadIndex(ctx)
	if err != nil {
		return nil, err
	}
	return byID[strings.ToLower(uniqueID)], nil
}

func (c *Client) loadIndex(ctx context.Context) (map[string][]Ref, error) {
	c.index.mu.Lock()
	defer c.index.mu.Unlock()
	if c.index.byID != nil && c.now().Sub(c.index.fetched) < datasetTTL {
		return c.index.byID, nil
	}
	byID, err := cached(c, indexFile, datasetTTL, func() (map[string][]Ref, error) { return c.fetchIndex(ctx) })
	if err != nil {
		return nil, err
	}
	c.index.byID, c.index.fetched = byID, c.now()
	return byID, nil
}

func (c *Client) fetchIndex(ctx context.Context) (map[string][]Ref, error) {
	u := c.IndexURL
	if u == "" {
		u = defaultIndexURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	b, err := c.do(req, maxIndex)
	if err != nil {
		return nil, err
	}
	return parseIndex(b)
}

// parseIndex keys the index by lowercased UniqueID and skips entries that are not "Site:number".
func parseIndex(b []byte) (map[string][]Ref, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := make(map[string][]Ref, len(raw))
	for id, v := range raw {
		var keys list
		if json.Unmarshal(v, &keys) != nil {
			continue
		}
		key := strings.ToLower(id)
		for _, k := range keys {
			site, num, ok := strings.Cut(string(k), ":")
			n, err := strconv.Atoi(num)
			if !ok || err != nil || site == "" {
				continue
			}
			out[key] = append(out[key], Ref{Site: site, ID: n})
		}
	}
	if len(out) == 0 {
		return nil, errors.New("mod dataset index is empty")
	}
	return out, nil
}

// Page is one Nexus mod page's current files as the dataset records them.
type Page struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Author    string `json:"author"`
	PageURL   string `json:"pageUrl"`
	Version   string `json:"version"`
	Downloads []File `json:"downloads"`
}

// File is one downloadable file; its Type is Nexus' category (Main, Optional, ...).
type File struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Version     string `json:"version"`
	FileName    string `json:"fileName"`
	SizeInBytes int64  `json:"sizeInBytes"`
	Mods        []Mod  `json:"mods"`
}

// Mod is a SMAPI manifest found inside a file.
type Mod struct {
	UniqueID     string       `json:"uniqueId"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	UpdateKeys   []string     `json:"updateKeys"`
	Dependencies []Dependency `json:"dependencies"`
}

// Dependency is a Dependencies[] entry, or the ContentPackFor framework as a required one.
type Dependency struct {
	UniqueID       string `json:"uniqueId"`
	MinimumVersion string `json:"minimumVersion"`
	Required       bool   `json:"required"`
}

// Page returns Nexus mod page id, from cache while younger than a month.
func (c *Client) Page(ctx context.Context, id int) (Page, error) {
	return cached(c, "dataset-nexus-"+strconv.Itoa(id)+".json", datasetTTL, func() (Page, error) {
		base := c.PageBase
		if base == "" {
			base = defaultPageBase
		}
		u, err := url.JoinPath(base, strconv.Itoa(id/1000), strconv.Itoa(id)+".json")
		if err != nil {
			return Page{}, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return Page{}, err
		}
		b, err := c.do(req, maxPage)
		if err != nil {
			return Page{}, err
		}
		return parsePage(b)
	})
}
