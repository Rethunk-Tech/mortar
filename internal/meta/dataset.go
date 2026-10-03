package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	requirementsTTL = 7 * 24 * time.Hour
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

// Index returns the whole index, keyed by lowercased UniqueID. Callers must not modify it.
func (c *Client) Index(ctx context.Context) (map[string][]Ref, error) { return c.loadIndex(ctx) }

func (c *Client) loadIndex(ctx context.Context) (map[string][]Ref, error) {
	c.index.mu.Lock()
	defer c.index.mu.Unlock()
	if c.index.byID != nil && c.now().Sub(c.index.fetched) < datasetTTL {
		return c.index.byID, nil
	}
	byID, err := Cached(c, indexFile, datasetTTL, func() (map[string][]Ref, error) { return c.fetchIndex(ctx) })
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

// Requirement is a Nexus page requirement.
type Requirement struct {
	ModID int    `json:"modId"`
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

// Page returns Nexus mod page id, from cache while younger than a month.
func (c *Client) Page(ctx context.Context, id int) (Page, error) {
	return Cached(c, "dataset-nexus-"+strconv.Itoa(id)+".json", datasetTTL, func() (Page, error) {
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
		if statusErr, ok := errors.AsType[*StatusError](err); ok && statusErr.Code == http.StatusNotFound {
			// The dataset has no file for this page: an answer, so it is cached like one.
			return Page{}, nil
		}
		if err != nil {
			return Page{}, err
		}
		return parsePage(b)
	})
}

// RequirementsCacheFile is the cache name for a Nexus page's listed requirements.
func RequirementsCacheFile(pageID int) string {
	return "nexus-requirements-" + strconv.Itoa(pageID) + ".json"
}

// PageRequirements returns the requirements listed on a Nexus mod page.
func (c *Client) PageRequirements(ctx context.Context, pageID int) ([]Requirement, error) {
	return Cached(c, RequirementsCacheFile(pageID), requirementsTTL, func() ([]Requirement, error) {
		body, err := json.Marshal(struct {
			Query string `json:"query"`
		}{Query: fmt.Sprintf(`{ legacyModsByDomain(ids:[{gameDomain:"stardewvalley",modId:%d}]) { nodes { modId name modRequirements { nexusRequirements { nodes { modId modName notes externalRequirement url } } } } } }`, pageID)})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.nexusmods.com/v2/graphql", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mortar")
		req.Header.Set("Accept", "application/json")
		raw, err := c.do(req, maxPage)
		if err != nil {
			return nil, err
		}
		var response struct {
			Data struct {
				LegacyModsByDomain struct {
					Nodes []struct {
						ModRequirements struct {
							NexusRequirements struct {
								Nodes []struct {
									ModID               string `json:"modId"`
									Name                string `json:"modName"`
									Notes               string `json:"notes"`
									ExternalRequirement bool   `json:"externalRequirement"`
								} `json:"nodes"`
							} `json:"nexusRequirements"`
						} `json:"modRequirements"`
					} `json:"nodes"`
				} `json:"legacyModsByDomain"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(raw, &response); err != nil {
			return nil, err
		}
		if len(response.Errors) > 0 {
			// Nexus answered (a hidden or removed page): no listed requirements, cached like any answer.
			return []Requirement{}, nil
		}
		if len(response.Data.LegacyModsByDomain.Nodes) == 0 {
			return []Requirement{}, nil
		}
		var out []Requirement
		for _, node := range response.Data.LegacyModsByDomain.Nodes[0].ModRequirements.NexusRequirements.Nodes {
			if node.ExternalRequirement || node.ModID == "2400" {
				continue
			}
			id, err := strconv.Atoi(node.ModID)
			if err != nil {
				continue
			}
			out = append(out, Requirement{ModID: id, Name: node.Name, Notes: node.Notes})
		}
		return out, nil
	})
}
