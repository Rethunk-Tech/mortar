// Package nexus is the Nexus Mods source driver.
package nexus

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
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
	base := cmp.Or(d.URL, nexus.BaseURL)
	client := cmp.Or(d.HTTP, http.DefaultClient)
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
	// An empty text lists the game's top mods, so no name filter. A filter's category list is an AND, and branches
	// joined by OR give "any included category": one branch per included category, each also excluding the others.
	scope := fmt.Sprintf(`gameDomainName:[{value:%s}]`, key)
	if strings.TrimSpace(q.Text) != "" {
		scope += fmt.Sprintf(`, name:[{value:%s, op:WILDCARD}]`, text)
	}
	cats := func(include string) (string, error) {
		var vals []string
		if include != "" {
			name, err := json.Marshal(include)
			if err != nil {
				return "", err
			}
			vals = append(vals, fmt.Sprintf(`{value:%s}`, name))
		}
		for _, c := range q.ExcludeCategories {
			name, err := json.Marshal(c)
			if err != nil {
				return "", err
			}
			vals = append(vals, fmt.Sprintf(`{value:%s, op:NOT_EQUALS}`, name))
		}
		if len(vals) == 0 {
			return scope, nil
		}
		return scope + ", categoryName:[" + strings.Join(vals, ", ") + "]", nil
	}
	filter, err := cats("")
	if err != nil {
		return source.Page{}, err
	}
	if len(q.Categories) > 0 {
		branches := make([]string, 0, len(q.Categories))
		for _, c := range q.Categories {
			br, err := cats(c)
			if err != nil {
				return source.Page{}, err
			}
			branches = append(branches, "{"+br+"}")
		}
		filter = "op:OR, filter:[" + strings.Join(branches, ", ") + "]"
	}
	query := fmt.Sprintf(
		`{ mods(filter:{%s}, sort:[{%s}], count: %d, offset: %d) { totalCount nodes { modId name summary author version endorsements downloads pictureUrl updatedAt adultContent } } }`,
		filter, sortClause(q.Sort), source.PageSize, offset,
	)
	var parsed searchResp
	err = source.DoJSON(ctx, source.Request{
		Service: d.Name(), Client: client, Method: http.MethodPost, URL: strings.TrimRight(base, "/") + graphQL,
		Body: searchBody{Query: query}, UserAgent: source.UserAgent(q.Version),
		Header: func(h http.Header) {
			h.Set("Application-Name", "Mortar")
			h.Set("Application-Version", q.Version)
		},
	}, &parsed)
	if err != nil {
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

// Sorts are the values of Nexus's ModsSort the driver offers (checked against the v2 schema's introspection).
func (Driver) Sorts() []string {
	return []string{source.SortEndorsements, source.SortDownloads, source.SortUpdated, source.SortNewest, source.SortName}
}

func sortClause(sort string) string {
	switch sort {
	case source.SortDownloads:
		return "downloads:{direction:DESC}"
	case source.SortUpdated:
		return "updatedAt:{direction:DESC}"
	case source.SortNewest:
		return "createdAt:{direction:DESC}"
	case source.SortName:
		return "name:{direction:ASC}"
	}
	return "endorsements:{direction:DESC}"
}

// CategoryNames lists a game's mod category names. Nexus serves them only to a signed-in account, so Mortar's
// account service sets this; without it the driver offers no category choices.
var CategoryNames func(ctx context.Context, domain string) ([]string, error)

// Categories lists the game's category names, or none while no account lookup is wired.
func (Driver) Categories(ctx context.Context, key string) ([]string, error) {
	if CategoryNames == nil {
		return []string{}, nil
	}
	names, err := CategoryNames(ctx, key)
	return source.UniqueNames(names), err
}
