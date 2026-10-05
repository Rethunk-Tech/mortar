package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// modsBatch is how many mods one GraphQL request asks about.
const modsBatch = 100

const modsQuery = `query($ids: [CompositeDomainWithIdInput!]!, $count: Int) {
  legacyModsByDomain(ids: $ids, count: $count) {
    nodes { modId name version status summary endorsements updatedAt createdAt pictureUrl author uploader { name } category viewerEndorsed adultContent downloads }
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
}

// Page is the mod page data the batched lookup carries; what only a page request gives (the description, the
// uploader's link, the unique download count) stays empty.
func (m ModInfo) Page() Page {
	p := Page{
		ModID: m.ModID, Name: m.Name, Summary: m.Summary, PictureURL: m.PictureURL, Version: m.Version, Author: m.Author,
		UploadedBy: m.Author, Endorsements: m.Endorsements, Downloads: m.Downloads, Created: m.Created, Updated: m.Updated,
		Adult: m.Adult, Status: m.Status, Available: m.Status == "published",
	}
	if m.Endorsed {
		p.Endorsement = "Endorsed"
	}
	return p
}

// ModsByDomain looks up many of one game's mods with one GraphQL request per 100 ids, instead of one request per
// mod. A mod Nexus does not return is absent from the map. A refused or failed request ends the lookup with the
// mods found so far and the error; a rate limit is returned as such, never retried.
func (c *Client) ModsByDomain(ctx context.Context, domain string, modIDs []int) (map[int]ModInfo, error) {
	out := make(map[int]ModInfo, len(modIDs))
	for start := 0; start < len(modIDs); start += modsBatch {
		if err := c.modsChunk(ctx, domain, modIDs[start:min(start+modsBatch, len(modIDs))], out); err != nil {
			return out, err
		}
	}
	return out, nil
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
					Category  string `json:"category"`
					Endorsed  *bool  `json:"viewerEndorsed"`
					Adult     bool   `json:"adultContent"`
					Downloads int    `json:"downloads"`
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
		out[n.ModID] = info
	}
	return nil
}
