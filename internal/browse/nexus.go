package browse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

type nexusSearchBody struct {
	Query string `json:"query"`
}

type nexusSearchResp struct {
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
			} `json:"nodes"`
		} `json:"mods"`
	} `json:"data"`
}

func (c *Client) searchNexus(ctx context.Context, game, text string, page int) (Page, error) {
	base := c.NexusURL
	if base == "" {
		base = nexus.BaseURL
	}
	t, err := gamepkg.NexusTitle(game)
	if err != nil {
		return Page{}, err
	}
	offset := (page - defaultPage) * pageSize
	escaped := strings.ReplaceAll(text, `"`, `\"`)
	query := fmt.Sprintf(
		`{ mods(filter:{gameDomainName:[{value:%q}], name:[{value:%q, op:WILDCARD}]}, sort:[{endorsements:{direction:DESC}}], count: %d, offset: %d) { totalCount nodes { modId name summary author version endorsements downloads pictureUrl updatedAt } } }`,
		t.Domain, escaped, pageSize, offset,
	)
	raw, err := json.Marshal(nexusSearchBody{Query: query})
	if err != nil {
		return Page{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+nexusGraphQL, bytes.NewReader(raw))
	if err != nil {
		return Page{}, err
	}
	req.Header.Set("Application-Name", "Mortar")
	req.Header.Set("Application-Version", c.Version)
	req.Header.Set("User-Agent", c.userAgent())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Page{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Page{}, fmt.Errorf("nexus answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return Page{}, err
	}
	var parsed nexusSearchResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Page{}, err
	}
	nodes := parsed.Data.Mods.Nodes
	items := make([]Item, 0, len(nodes))
	for _, n := range nodes {
		id := strconv.Itoa(n.ModID)
		items = append(items, Item{
			Source:       sourceNexus,
			ID:           id,
			Name:         n.Name,
			Summary:      n.Summary,
			Author:       n.Author,
			Version:      n.Version,
			Picture:      n.PictureURL,
			Endorsements: n.Endorsements,
			Downloads:    n.Downloads,
			Updated:      n.UpdatedAt,
			URL:          nexus.ModURL(t.Domain, n.ModID),
		})
	}
	c.markInstalled(items)
	return Page{Total: parsed.Data.Mods.TotalCount, Items: items}, nil
}
