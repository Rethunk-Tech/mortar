package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// modsBatch is how many mods one GraphQL request asks about. Nexus refuses a query whose complexity passes 10,000,
// and with each mod's requirements a mod costs about 160.
const modsBatch = 50

const modsQuery = `query($ids: [CompositeDomainWithIdInput!]!, $count: Int) {
  legacyModsByDomain(ids: $ids, count: $count) {
    nodes {
      modId gameId name version status summary endorsements updatedAt createdAt pictureUrl author uploader { name } category viewerEndorsed adultContent downloads
      modRequirements { nexusRequirements { nodes { modId modName url externalRequirement notes gameId } } }
    }
  }
}`

// ModInfo is one mod page as the batched lookup reports it.
type ModInfo struct {
	ModID        int
	Name         string
	Version      string
	Status       string
	Summary      string
	Author       string
	PictureURL   string
	Category     string
	Endorsements int
	Downloads    int
	Adult        bool
	// Endorsed is whether the signed-in user endorsed the mod.
	Endorsed bool
	Created  time.Time
	Updated  time.Time
	// Requirements are what the mod page lists under "Nexus requirements".
	Requirements []Requirement
}

// Requirement is one mod or outside download a mod page says is needed. A Nexus mod of the same game has ModID and
// its page as URL; anything else (another game's mod, an outside site) is External, with whatever URL Nexus gave.
type Requirement struct {
	ModID    int    `json:"modId"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	External bool   `json:"external"`
	Notes    string `json:"notes"`
}

// Page is the mod page data the batched lookup carries; what only a page request gives (the description, the
// uploader's link, the unique download count) stays empty.
func (m ModInfo) Page() Page {
	p := Page{
		ModID: m.ModID, Name: m.Name, Summary: m.Summary, PictureURL: m.PictureURL, Version: m.Version, Author: m.Author,
		UploadedBy: m.Author, Endorsements: m.Endorsements, Downloads: m.Downloads, Created: m.Created, Updated: m.Updated,
		Adult: m.Adult, Status: m.Status, Available: m.Status == "published", Requirements: m.Requirements,
	}
	if m.Endorsed {
		p.Endorsement = "Endorsed"
	}
	return p
}

// Requests is how many GraphQL requests ModsByDomain makes for n mods.
func Requests(n int) int { return (n + modsBatch - 1) / modsBatch }

// modsParallel bounds the batched requests in flight: each takes about half a second, so a cold list of hundreds of
// mods waits for a couple of rounds instead of one request after another. The request count is the same either way.
const modsParallel = 8

// ModsByDomain looks up many of one game's mods with one GraphQL request per 50 ids, instead of one request per
// mod. A mod Nexus does not return is absent from the map. A refused or failed request stops further requests and
// the lookup returns the mods found and the error; a rate limit is returned as such, never retried.
func (c *Client) ModsByDomain(ctx context.Context, domain string, modIDs []int) (map[int]ModInfo, error) {
	out := make(map[int]ModInfo, len(modIDs))
	var (
		mu    sync.Mutex
		first error
		wg    sync.WaitGroup
	)
	failed := func() bool {
		mu.Lock()
		defer mu.Unlock()
		return first != nil
	}
	slots := make(chan struct{}, modsParallel)
	for start := 0; start < len(modIDs); start += modsBatch {
		slots <- struct{}{}
		if failed() {
			break
		}
		ids := modIDs[start:min(start+modsBatch, len(modIDs))]
		wg.Go(func() {
			defer func() { <-slots }()
			got := make(map[int]ModInfo, len(ids))
			err := c.modsChunk(ctx, domain, ids, got)
			mu.Lock()
			defer mu.Unlock()
			maps.Copy(out, got)
			if first == nil {
				first = err
			}
		})
	}
	wg.Wait()
	return out, first
}

func (c *Client) modsChunk(ctx context.Context, domain string, ids []int, out map[int]ModInfo) error {
	type ref struct {
		Domain string `json:"gameDomain"`
		ModID  int    `json:"modId"`
	}
	refs := make([]ref, len(ids))
	for i, id := range ids {
		refs[i] = ref{domain, id}
	}
	code, status, body, err := c.roundTrip(ctx, http.MethodPost, "/v2/graphql", map[string]any{
		"query": modsQuery, "variables": map[string]any{"ids": refs, "count": len(ids)},
	})
	if err != nil {
		return err
	}
	switch code {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusTooManyRequests:
		return c.rateLimited()
	default:
		return &StatusError{Code: code, Status: status}
	}
	var raw struct {
		Data struct {
			Mods struct {
				Nodes []struct {
					ModID        int       `json:"modId"`
					GameID       int       `json:"gameId"`
					Name         string    `json:"name"`
					Version      string    `json:"version"`
					Status       string    `json:"status"`
					Summary      string    `json:"summary"`
					Endorsements int       `json:"endorsements"`
					Updated      time.Time `json:"updatedAt"`
					Created      time.Time `json:"createdAt"`
					PictureURL   string    `json:"pictureUrl"`
					Author       string    `json:"author"`
					Uploader     struct {
						Name string `json:"name"`
					} `json:"uploader"`
					Category     string `json:"category"`
					Endorsed     *bool  `json:"viewerEndorsed"`
					Adult        bool   `json:"adultContent"`
					Downloads    int    `json:"downloads"`
					Requirements struct {
						Nexus struct {
							Nodes []struct {
								ModID    string `json:"modId"`
								Name     string `json:"modName"`
								URL      string `json:"url"`
								External bool   `json:"externalRequirement"`
								Notes    string `json:"notes"`
								GameID   string `json:"gameId"`
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
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	if len(raw.Data.Mods.Nodes) == 0 && len(raw.Errors) > 0 {
		return errors.New("nexus: " + raw.Errors[0].Message)
	}
	for _, n := range raw.Data.Mods.Nodes {
		info := ModInfo{
			ModID: n.ModID, Name: n.Name, Version: n.Version, Status: n.Status, Summary: n.Summary, Author: n.Author,
			PictureURL: n.PictureURL, Category: n.Category, Endorsements: n.Endorsements, Downloads: n.Downloads,
			Adult: n.Adult, Endorsed: n.Endorsed != nil && *n.Endorsed, Created: n.Created.UTC(), Updated: n.Updated.UTC(),
		}
		if info.Author == "" {
			info.Author = n.Uploader.Name
		}
		for _, r := range n.Requirements.Nexus.Nodes {
			req := Requirement{Name: r.Name, URL: r.URL, Notes: r.Notes, External: true}
			if id, err := strconv.Atoi(r.ModID); err == nil && id > 0 && !r.External && r.GameID == strconv.Itoa(n.GameID) {
				req.ModID, req.URL, req.External = id, ModURL(domain, id), false
			} else if d, id, ok := ParseModURL(r.URL); ok && strings.EqualFold(d, domain) {
				// Authors often list a Nexus mod as an outside link to its page; it is that mod all the same.
				req.ModID, req.URL, req.External = id, ModURL(domain, id), false
			}
			info.Requirements = append(info.Requirements, req)
		}
		out[n.ModID] = info
	}
	return nil
}
