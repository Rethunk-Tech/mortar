// Package browse searches Nexus Mods and GitHub for mods of one game.
package browse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	pageSize       = 20
	requestTimeout = 20 * time.Second
	maxBody        = 16 << 20
	githubCacheTTL = 10 * time.Minute
	defaultPage    = 1
	sourceNexus    = "nexus"
	sourceGitHub   = "github"
	stardewDomain  = "stardewvalley"
	stardewGHTopic = "stardew-valley-mod"
	nexusGraphQL   = "/v2/graphql"
	githubSearch   = "/search/repositories"
)

// ErrBusy means GitHub refused the search (403/429); the caller should wait a minute.
var ErrBusy = errors.New("GitHub is busy, try again in a minute")

// ErrUnknownSource means source is not nexus or github.
var ErrUnknownSource = errors.New("unknown browse source")

// Item is one search hit.
type Item struct {
	Source       string `json:"source"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	Summary      string `json:"summary"`
	Author       string `json:"author"`
	Version      string `json:"version"`
	Picture      string `json:"picture"`
	Endorsements int    `json:"endorsements"`
	Stars        int    `json:"stars"`
	Downloads    int    `json:"downloads"`
	Updated      string `json:"updated"`
	URL          string `json:"url"`
	Installed    bool   `json:"installed"`
}

// Page is one slice of search hits.
type Page struct {
	Total int    `json:"total"`
	Items []Item `json:"items"`
}

// InstalledFunc reports whether the open profile already has this Nexus mod id or GitHub repo.
type InstalledFunc func(source, id string) bool

// Client searches remote catalogues. Tests set NexusURL, GitHubURL, HTTP, and Cache.
type Client struct {
	HTTP      *http.Client
	NexusURL  string
	GitHubURL string
	Version   string
	Now       func() time.Time
	Installed InstalledFunc
	Cache     *RepoCache
}

// Search returns one page of mods for game from source matching text.
func (c *Client) Search(ctx context.Context, game, source, text string, page int) (Page, error) {
	if page < defaultPage {
		page = defaultPage
	}
	switch strings.ToLower(strings.TrimSpace(source)) {
	case sourceNexus:
		return c.searchNexus(ctx, game, text, page)
	case sourceGitHub:
		return c.searchGitHub(ctx, text, page)
	default:
		return Page{}, fmt.Errorf("%w: %s", ErrUnknownSource, source)
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now().UTC()
}

func (c *Client) userAgent() string {
	if c.Version == "" {
		return "Mortar"
	}
	return "Mortar/" + c.Version
}

func (c *Client) markInstalled(items []Item) {
	if c.Installed == nil {
		return
	}
	for i := range items {
		items[i].Installed = c.Installed(items[i].Source, items[i].ID)
	}
}
