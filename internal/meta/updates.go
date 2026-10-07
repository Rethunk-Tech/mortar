package meta

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
)

const (
	defaultUpdatesURL = "https://smapi.io/api/v4.0.0/mods"
	updatesTTL        = time.Hour
	updatesFile       = "smapi-updates.json"
	updatesBatch      = 100
	updatesBackoff    = 5 * time.Minute
	maxUpdates        = 16 << 20
)

// errUpdatesPaused stands in for a failed ask while smapi.io is being given a rest.
var errUpdatesPaused = errors.New("update checks paused after a failed request")

// UpdateRequest describes the installed environment and mods to check.
type UpdateRequest struct {
	APIVersion  string
	GameVersion string
	Platform    string // Android, Linux, Mac or Windows
	Mods        []InstalledMod
	// Fresh asks the API even for answers still young enough to come from cache, for an explicit check.
	Fresh bool
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
	Source  string `json:"source,omitempty"`
}

// UpdateResult is the answer for one mod. Known is false when SMAPI's API could not be reached and no cached
// answer exists: the caller shows "unknown", never an error. A known result with no Suggested has no update.
type UpdateResult struct {
	ID                   string  `json:"id"`
	Known                bool    `json:"known"`
	Suggested            *Update `json:"suggested,omitempty"`
	Unofficial           *Update `json:"unofficial,omitempty"`
	Compatibility        string  `json:"compatibility,omitempty"`
	CompatibilitySummary string  `json:"compatibilitySummary,omitempty"`
	BrokeIn              string  `json:"brokeIn,omitempty"`
	GitHubRepo           string  `json:"gitHubRepo,omitempty"`
}

type apiRequest struct {
	Mods                    []InstalledMod `json:"mods"`
	APIVersion              string         `json:"apiVersion,omitempty"`
	GameVersion             string         `json:"gameVersion,omitempty"`
	Platform                string         `json:"platform,omitempty"`
	IncludeExtendedMetadata bool           `json:"includeExtendedMetadata"`
}

type apiUpdate struct {
	Version            string     `json:"version"`
	URL                string     `json:"url"`
	UpdateKey          string     `json:"updateKey"`
	Unofficial         *apiUpdate `json:"unofficial"`
	UnofficialForSmapi *apiUpdate `json:"unofficialForSmapi"`
}

func (u *apiUpdate) asUpdate() *Update {
	if u == nil || strings.TrimSpace(u.Version) == "" {
		return nil
	}
	return &Update{Version: u.Version, URL: u.URL, Source: u.UpdateKey}
}

func firstUnofficial(parts ...*apiUpdate) *Update {
	for _, p := range parts {
		if got := p.asUpdate(); got != nil {
			return got
		}
	}
	return nil
}

type apiMod struct {
	ID              string     `json:"id"`
	SuggestedUpdate *apiUpdate `json:"suggestedUpdate"`
	Metadata        *struct {
		GitHubRepo           string     `json:"gitHubRepo"`
		CompatibilityStatus  string     `json:"compatibilityStatus"`
		CompatibilitySummary string     `json:"compatibilitySummary"`
		BrokeIn              string     `json:"brokeIn"`
		Unofficial           *apiUpdate `json:"unofficial"`
		UnofficialUpdate     *apiUpdate `json:"unofficialUpdate"`
		UnofficialForSmapi   *apiUpdate `json:"unofficialForSmapi"`
	} `json:"metadata"`
}

func (r UpdateRequest) key(m InstalledMod) string {
	return strings.ToLower(strings.Join([]string{m.ID, strings.Join(m.UpdateKeys, ","), m.Version, r.GameVersion, r.APIVersion, r.Platform}, "|"))
}

// CheckUpdates returns one result per requested mod, in order. Answers younger than an hour come from cache;
// the rest are asked in batches, and a batch that fails falls back to stale cache, else unknown.
func (c *Client) CheckUpdates(ctx context.Context, req UpdateRequest) []UpdateResult {
	c.updatesMu.Lock()
	defer c.updatesMu.Unlock()
	path, pathErr := c.cachePath(updatesFile)
	var store map[string]entry[UpdateResult]
	if pathErr == nil {
		if e, ok := readEntry[map[string]entry[UpdateResult]](path); ok {
			store = e.Value
			if e.Build != appversion.Build() {
				// Another build's answers are asked again, and serve only while the API cannot be reached.
				for k, v := range store {
					v.Fetched = time.Time{}
					store[k] = v
				}
			}
		}
	}
	if store == nil {
		store = map[string]entry[UpdateResult]{}
	}
	now := c.now()
	out := make([]UpdateResult, len(req.Mods))
	var stale []int
	for i, m := range req.Mods {
		if e, ok := store[req.key(m)]; ok && !req.Fresh && now.Sub(e.Fetched) < updatesTTL {
			out[i] = e.Value
		} else {
			stale = append(stale, i)
		}
	}
	var batches [][]int
	for start := 0; start < len(stale); start += updatesBatch {
		batches = append(batches, stale[start:min(start+updatesBatch, len(stale))])
	}
	answers := c.askBatches(ctx, req, batches, now)
	dirty := false
	for b, batch := range batches {
		got, err := answers[b].got, answers[b].err
		for _, i := range batch {
			k := req.key(req.Mods[i])
			switch {
			case err == nil:
				// The API answered: a mod it does not list is known to have no data, not unknown.
				out[i] = got[strings.ToLower(req.Mods[i].ID)]
				out[i].ID = req.Mods[i].ID
				out[i].Known = true
				store[k] = entry[UpdateResult]{Fetched: now, Value: out[i]}
				dirty = true
			case store[k].Value.Known:
				out[i] = store[k].Value
			default:
				out[i] = UpdateResult{ID: req.Mods[i].ID}
			}
		}
	}
	if dirty && pathErr == nil {
		for k, e := range store {
			if now.Sub(e.Fetched) > 24*updatesTTL {
				delete(store, k)
			}
		}
		writeEntry(path, entry[map[string]entry[UpdateResult]]{Fetched: now, Value: store, Build: appversion.Build()})
	}
	return out
}

type updatesAnswer struct {
	got map[string]UpdateResult
	err error
}

// updatesParallel bounds the batches asked of smapi.io at once: a cold profile of hundreds of mods is several
// batches, which one after another cost about a second.
const updatesParallel = 4

// askBatches asks smapi.io about each batch of req's mods, a few at a time. A failed ask pauses asking: batches not
// yet sent then answer errUpdatesPaused.
func (c *Client) askBatches(ctx context.Context, req UpdateRequest, batches [][]int, now time.Time) []updatesAnswer {
	answers := make([]updatesAnswer, len(batches))
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, updatesParallel)
	for b, batch := range batches {
		slots <- struct{}{}
		mu.Lock()
		paused := now.Before(c.updatesPause)
		mu.Unlock()
		if paused {
			<-slots
			answers[b].err = errUpdatesPaused
			continue
		}
		asked := make([]InstalledMod, len(batch))
		for j, i := range batch {
			asked[j] = req.Mods[i]
		}
		wg.Go(func() {
			defer func() { <-slots }()
			got, err := c.askUpdates(ctx, req, asked)
			answers[b] = updatesAnswer{got, err}
			if err != nil {
				mu.Lock()
				c.updatesPause = now.Add(updatesBackoff)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return answers
}

func (c *Client) askUpdates(ctx context.Context, req UpdateRequest, mods []InstalledMod) (map[string]UpdateResult, error) {
	body, err := json.Marshal(apiRequest{Mods: mods, APIVersion: req.APIVersion, GameVersion: req.GameVersion, Platform: req.Platform, IncludeExtendedMetadata: true})
	if err != nil {
		return nil, err
	}
	u := cmp.Or(c.UpdatesURL, defaultUpdatesURL)
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
		r := UpdateResult{ID: a.ID, Known: true, Suggested: a.SuggestedUpdate.asUpdate()}
		if a.Metadata != nil {
			r.Compatibility = a.Metadata.CompatibilityStatus
			r.CompatibilitySummary = a.Metadata.CompatibilitySummary
			r.BrokeIn, r.GitHubRepo = a.Metadata.BrokeIn, a.Metadata.GitHubRepo
			r.Unofficial = firstUnofficial(a.Metadata.UnofficialUpdate, a.Metadata.Unofficial, a.Metadata.UnofficialForSmapi)
		}
		if r.Unofficial == nil && a.SuggestedUpdate != nil {
			r.Unofficial = firstUnofficial(a.SuggestedUpdate.Unofficial, a.SuggestedUpdate.UnofficialForSmapi)
		}
		out[strings.ToLower(a.ID)] = r
	}
	return out, nil
}
