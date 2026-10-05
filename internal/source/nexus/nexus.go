// Package nexus is the Nexus Mods source driver.
package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const graphQL = "/v2/graphql"

// Driver searches Nexus Mods. The zero value talks to the real site.
type Driver struct {
	HTTP *http.Client
	URL  string
}

var _ = source.Register(Driver{})

// ID is the catalog id.
func (Driver) ID() string { return "nexus" }

// Name is the site's name.
func (Driver) Name() string { return "Nexus Mods" }

// Modes lists both ways: Premium accounts download through the API, everyone else clicks Mod Manager Download.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Download, source.Handoff} }

// Schemes is the Mod Manager Download link scheme.
func (Driver) Schemes() []string { return []string{"nxm"} }

// ModPageURL is the mod's page.
func (Driver) ModPageURL(gameKey, id string) string {
	modID, err := strconv.Atoi(id)
	if err != nil {
		return ""
	}
	return nexus.ModURL(gameKey, modID)
}

type searchBody struct {
	Query string `json:"query"`
}

type searchResp struct {
	Data struct {
		Mods struct {
			TotalCount int `json:"totalCount"`
			Nodes      []struct {
				ModID        int    `json:"modId"`
				Name         string `json:"name"`
				Summary      string `json:"summary"`
				Author       string `json:"author"`
				Version      string `json:"version"`
				Endorsements int    `json:"endorsements"`
				Downloads    int    `json:"downloads"`
				PictureURL   string `json:"pictureUrl"`
				UpdatedAt    string `json:"updatedAt"`
				AdultContent bool   `json:"adultContent"`
			} `json:"nodes"`
		} `json:"mods"`
	} `json:"data"`
}

// Search lists mods of the game, most endorsed first.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	if q.Key == "" {
		return source.Page{}, fmt.Errorf("game %q has no Nexus domain", q.Game)
	}
	base := d.URL
	if base == "" {
		base = nexus.BaseURL
	}
	client := d.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	offset := (q.Page - source.FirstPage) * source.PageSize
	// A JSON string is a valid GraphQL string, so no input can end the literal or inject a field.
	key, err := json.Marshal(q.Key)
	if err != nil {
		return source.Page{}, err
	}
	text, err := json.Marshal(q.Text)
	if err != nil {
		return source.Page{}, err
	}
	// An empty text lists the game's most endorsed mods, so no name filter.
	filter := fmt.Sprintf(`gameDomainName:[{value:%s}]`, key)
	if strings.TrimSpace(q.Text) != "" {
		filter += fmt.Sprintf(`, name:[{value:%s, op:WILDCARD}]`, text)
	}
	query := fmt.Sprintf(
		`{ mods(filter:{%s}, sort:[{endorsements:{direction:DESC}}], count: %d, offset: %d) { totalCount nodes { modId name summary author version endorsements downloads pictureUrl updatedAt adultContent } } }`,
		filter, source.PageSize, offset,
	)
	raw, err := json.Marshal(searchBody{Query: query})
	if err != nil {
		return source.Page{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+graphQL, bytes.NewReader(raw))
	if err != nil {
		return source.Page{}, err
	}
	req.Header.Set("Application-Name", "Mortar")
	req.Header.Set("Application-Version", q.Version)
	req.Header.Set("User-Agent", source.UserAgent(q.Version))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return source.Page{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return source.Page{}, fmt.Errorf("nexus answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, source.MaxBody))
	if err != nil {
		return source.Page{}, err
	}
	var parsed searchResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return source.Page{}, err
	}
	nodes := parsed.Data.Mods.Nodes
	items := make([]source.Item, 0, len(nodes))
	for _, n := range nodes {
		items = append(items, source.Item{
			Source:       d.ID(),
			ID:           strconv.Itoa(n.ModID),
			Name:         n.Name,
			Summary:      n.Summary,
			Author:       n.Author,
			Version:      n.Version,
			Picture:      n.PictureURL,
			Endorsements: n.Endorsements,
			Downloads:    n.Downloads,
			Updated:      n.UpdatedAt,
			Adult:        n.AdultContent,
			URL:          nexus.ModURL(q.Key, n.ModID),
		})
	}
	return source.Page{Total: parsed.Data.Mods.TotalCount, Items: items}, nil
}
