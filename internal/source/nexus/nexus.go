// Package nexus is the Nexus Mods source driver.
package nexus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

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
	return nexusmods.ModURL(gameKey, modID)
}

// MaxPageSize is the most mods one GraphQL mods query returns: asking for more still answers 80.
func (Driver) MaxPageSize() int { return nexusmods.MaxSearchCount }

// Search lists mods of the game, most endorsed first.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	if q.Key == "" {
		return source.Page{}, fmt.Errorf("game %q has no Nexus domain", q.Game)
	}
	c := nexusmods.New("Mortar", q.Version)
	c.BaseURL = d.URL
	if d.HTTP != nil {
		c.HTTP = d.HTTP
	} else {
		c.HTTP = &http.Client{Transport: &httpkit.Transport{UserAgent: appversion.UserAgent(q.Version)}}
	}
	res, err := c.Search(ctx, nexusmods.SearchQuery{
		Domain: q.Key, Text: q.Text, Sort: modSort(q.Sort),
		Categories: q.Categories, ExcludeCategories: q.ExcludeCategories,
		Count: q.Len(), Offset: (q.Page - source.FirstPage) * q.Len(),
	})
	if err != nil {
		return source.Page{}, d.searchErr(err)
	}
	items := make([]source.Item, 0, len(res.Mods))
	for _, n := range res.Mods {
		it := source.Item{
			Source:       d.ID(),
			ID:           strconv.Itoa(n.ModID),
			Name:         n.Name,
			Summary:      n.Summary,
			Author:       n.Author,
			Version:      n.Version,
			Picture:      n.PictureURL,
			Endorsements: n.Endorsements,
			Downloads:    n.Downloads,
			Size:         n.SizeKB * 1024,
			Adult:        n.Adult,
			Category:     n.Category,
			URL:          nexusmods.ModURL(q.Key, n.ModID),
		}
		if !n.Updated.IsZero() {
			it.Updated = n.Updated.Format(time.RFC3339)
		}
		items = append(items, it)
	}
	return source.Page{Total: res.Total, Items: items}, nil
}

// searchErr keeps Browse's *BusyError and *StatusError; the library reports those as httpkit types.
func (d Driver) searchErr(err error) error {
	if errors.Is(err, nexusmods.ErrUnauthorized) {
		return &source.StatusError{Service: d.Name(), Code: http.StatusUnauthorized, Status: "401 Unauthorized"}
	}
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return &source.BusyError{Source: d.Name(), Reset: rl.Reset}
	}
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return &source.StatusError{Service: d.Name(), Code: se.Code, Status: se.Status}
	}
	return err
}

// Sorts are the values of Nexus's ModsSort the driver offers (checked against the v2 schema's introspection).
func (Driver) Sorts() []string {
	return []string{source.SortEndorsements, source.SortDownloads, source.SortUpdated, source.SortNewest, source.SortName}
}

func modSort(sort string) nexusmods.ModSort {
	switch sort {
	case source.SortDownloads:
		return nexusmods.SortDownloads
	case source.SortUpdated:
		return nexusmods.SortUpdated
	case source.SortNewest:
		return nexusmods.SortCreated
	case source.SortName:
		return nexusmods.SortName
	}
	return nexusmods.SortEndorsements
}

// CategoryNames lists a game's mod category names. Nexus serves them only to a signed-in account, so Mortar's
// account service sets this; without it the driver offers no category choices.
var CategoryNames func(ctx context.Context, domain string) ([]string, error)

// Categories lists the game's category names, or none while no account lookup is wired; each hit still carries its own
// category name, which Browse filters noise by.
func (Driver) Categories(ctx context.Context, key string) ([]string, error) {
	if CategoryNames == nil {
		return []string{}, nil
	}
	names, err := CategoryNames(ctx, key)
	return source.UniqueNames(names), err
}
