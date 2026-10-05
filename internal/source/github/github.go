// Package github is the GitHub source driver: repositories carrying the game's topic.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

const (
	apiURL    = "https://api.github.com"
	searchURL = "/search/repositories"
	cacheTTL  = 10 * time.Minute
)

// Driver searches GitHub repositories. HTTP, URL and Now are for tests; the zero value talks to the real site.
type Driver struct {
	HTTP *http.Client
	URL  string
	Now  func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	page  source.Page
	until time.Time
}

var _ = source.Register(&Driver{})

// ID is the catalog id.
func (*Driver) ID() string { return "github" }

// Name is the site's name.
func (*Driver) Name() string { return "GitHub" }

// Modes says Mortar downloads release assets itself.
func (*Driver) Modes() []source.Acquire { return []source.Acquire{source.Download} }

type searchResp struct {
	TotalCount int `json:"total_count"`
	Items      []struct {
		FullName        string `json:"full_name"`
		Description     string `json:"description"`
		HTMLURL         string `json:"html_url"`
		StargazersCount int    `json:"stargazers_count"`
		UpdatedAt       string `json:"updated_at"`
		Owner           struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
		} `json:"owner"`
	} `json:"items"`
}

func (d *Driver) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now().UTC()
}

func (d *Driver) cached(key string) (source.Page, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.cache[key]
	if !ok || !d.now().Before(e.until) {
		return source.Page{}, false
	}
	return source.Page{Total: e.page.Total, Items: append([]source.Item(nil), e.page.Items...)}, true
}

func (d *Driver) remember(key string, page source.Page) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cache == nil {
		d.cache = map[string]cacheEntry{}
	}
	now := d.now()
	maps.DeleteFunc(d.cache, func(_ string, e cacheEntry) bool { return !now.Before(e.until) })
	d.cache[key] = cacheEntry{page: page, until: now.Add(cacheTTL)}
}

// Search lists repositories tagged with the game's topic (q.Key), most starred first. Answers are cached for ten
// minutes because GitHub's anonymous search limit is small.
func (d *Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	// Without a topic the query would search every repository on GitHub, which is noise in a game's browse.
	if q.Key == "" {
		return source.Page{}, nil
	}
	key := q.Key + "\n" + q.Text + "\n" + strconv.Itoa(q.Page)
	if hit, ok := d.cached(key); ok {
		return hit, nil
	}
	base := d.URL
	if base == "" {
		base = apiURL
	}
	client := d.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	terms := strings.TrimSpace(q.Text + " topic:" + q.Key)
	params := url.Values{}
	params.Set("q", terms)
	params.Set("sort", "stars")
	params.Set("per_page", strconv.Itoa(source.PageSize))
	params.Set("page", strconv.Itoa(q.Page))
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return source.Page{}, err
	}
	req.Header.Set("User-Agent", source.UserAgent(q.Version))
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return source.Page{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return source.Page{}, source.ErrBusy
	}
	if resp.StatusCode != http.StatusOK {
		return source.Page{}, fmt.Errorf("github answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, source.MaxBody))
	if err != nil {
		return source.Page{}, err
	}
	var parsed searchResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return source.Page{}, err
	}
	items := make([]source.Item, 0, len(parsed.Items))
	for _, n := range parsed.Items {
		items = append(items, source.Item{
			Source:  d.ID(),
			ID:      n.FullName,
			Name:    n.FullName,
			Summary: n.Description,
			Author:  n.Owner.Login,
			Picture: n.Owner.AvatarURL,
			Stars:   n.StargazersCount,
			Updated: n.UpdatedAt,
			URL:     n.HTMLURL,
			Repo:    n.FullName,
		})
	}
	page := source.Page{Total: parsed.TotalCount, Items: items}
	d.remember(key, source.Page{Total: page.Total, Items: append([]source.Item(nil), items...)})
	return page, nil
}
