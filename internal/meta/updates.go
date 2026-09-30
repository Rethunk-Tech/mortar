package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	defaultUpdatesURL = "https://smapi.io/api/v4.0.0/mods"
	updatesTTL        = time.Hour
	updatesFile       = "smapi-updates.json"
	updatesBatch      = 100
	maxUpdates        = 16 << 20
)

// UpdateRequest describes the installed environment and mods to check.
type UpdateRequest struct {
	APIVersion  string
	GameVersion string
	Platform    string // Android, Linux, Mac or Windows
	Mods        []InstalledMod
}

// InstalledMod is one mod to check.
type InstalledMod struct {
	ID         string   `json:"id"`
	UpdateKeys []string `json:"updateKeys,omitempty"`
	Version    string   `json:"installedVersion,omitempty"`
}

// Update is SMAPI's suggested newer version.
type Update struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// UpdateResult is the answer for one mod. Known is false when SMAPI's API could not be reached and no cached
// answer exists: the caller shows "unknown", never an error. A known result with no Suggested has no update.
type UpdateResult struct {
	ID            string  `json:"id"`
	Known         bool    `json:"known"`
	Suggested     *Update `json:"suggested,omitempty"`
	Compatibility string  `json:"compatibility,omitempty"`
	BrokeIn       string  `json:"brokeIn,omitempty"`
	GitHubRepo    string  `json:"gitHubRepo,omitempty"`
}

type apiRequest struct {
	Mods                    []InstalledMod `json:"mods"`
	APIVersion              string         `json:"apiVersion,omitempty"`
	GameVersion             string         `json:"gameVersion,omitempty"`
	Platform                string         `json:"platform,omitempty"`
	IncludeExtendedMetadata bool           `json:"includeExtendedMetadata"`
}

type apiMod struct {
	ID              string  `json:"id"`
	SuggestedUpdate *Update `json:"suggestedUpdate"`
	Metadata        *struct {
		GitHubRepo          string `json:"gitHubRepo"`
		CompatibilityStatus string `json:"compatibilityStatus"`
		BrokeIn             string `json:"brokeIn"`
	} `json:"metadata"`
}

func (r UpdateRequest) key(m InstalledMod) string {
	return strings.ToLower(strings.Join([]string{m.ID, strings.Join(m.UpdateKeys, ","), m.Version, r.GameVersion, r.APIVersion, r.Platform}, "|"))
}

// CheckUpdates returns one result per requested mod, in order. Answers younger than an hour come from cache;
// the rest are asked in batches, and a batch that fails falls back to stale cache, else unknown.
func (c *Client) CheckUpdates(ctx context.Context, req UpdateRequest) []UpdateResult {
	path, pathErr := c.cachePath(updatesFile)
	var store map[string]entry[UpdateResult]
	if pathErr == nil {
		if e, ok := readEntry[map[string]entry[UpdateResult]](path); ok {
			store = e.Value
		}
	}
	if store == nil {
		store = map[string]entry[UpdateResult]{}
	}
	now := c.now()
	out := make([]UpdateResult, len(req.Mods))
	var stale []int
	for i, m := range req.Mods {
		if e, ok := store[req.key(m)]; ok && now.Sub(e.Fetched) < updatesTTL {
			out[i] = e.Value
		} else {
			stale = append(stale, i)
		}
	}
	dirty := false
	for start := 0; start < len(stale); start += updatesBatch {
		batch := stale[start:min(start+updatesBatch, len(stale))]
		asked := make([]InstalledMod, len(batch))
		for j, i := range batch {
			asked[j] = req.Mods[i]
		}
		got, err := c.askUpdates(ctx, req, asked)
		for j, i := range batch {
			k := req.key(req.Mods[i])
			switch {
			case err == nil:
				out[i] = got[strings.ToLower(req.Mods[i].ID)]
				out[i].ID = req.Mods[i].ID
				if out[i].Known {
					store[k] = entry[UpdateResult]{Fetched: now, Value: out[i]}
					dirty = true
				}
			case store[k].Value.Known:
				out[i] = store[k].Value
			default:
				out[i] = UpdateResult{ID: asked[j].ID}
			}
		}
	}
	if dirty && pathErr == nil {
		for k, e := range store {
			if now.Sub(e.Fetched) > 24*updatesTTL {
				delete(store, k)
			}
		}
		writeEntry(path, entry[map[string]entry[UpdateResult]]{Fetched: now, Value: store})
	}
	return out
}

func (c *Client) askUpdates(ctx context.Context, req UpdateRequest, mods []InstalledMod) (map[string]UpdateResult, error) {
	body, err := json.Marshal(apiRequest{Mods: mods, APIVersion: req.APIVersion, GameVersion: req.GameVersion, Platform: req.Platform, IncludeExtendedMetadata: true})
	if err != nil {
		return nil, err
	}
	u := c.UpdatesURL
	if u == "" {
		u = defaultUpdatesURL
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	hr.Header.Set("Content-Type", "application/json")
	b, err := c.do(hr, maxUpdates)
	if err != nil {
		return nil, err
	}
	var answers []apiMod
	if err := json.Unmarshal(b, &answers); err != nil {
		return nil, err
	}
	out := make(map[string]UpdateResult, len(answers))
	for _, a := range answers {
		r := UpdateResult{ID: a.ID, Known: true, Suggested: a.SuggestedUpdate}
		if a.Metadata != nil {
			r.Compatibility, r.BrokeIn, r.GitHubRepo = a.Metadata.CompatibilityStatus, a.Metadata.BrokeIn, a.Metadata.GitHubRepo
		}
		out[strings.ToLower(a.ID)] = r
	}
	return out, nil
}
