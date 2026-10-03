package browse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const githubAPI = "https://api.github.com"

// RepoCache holds GitHub search answers for githubCacheTTL.
type RepoCache struct {
	mu    sync.Mutex
	items map[string]cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	page  Page
	until time.Time
}

func (c *RepoCache) get(key string, now time.Time) (Page, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		return Page{}, false
	}
	e, ok := c.items[key]
	if !ok || !now.Before(e.until) {
		return Page{}, false
	}
	return e.page, true
}

func (c *RepoCache) set(key string, page Page, now time.Time, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		c.items = map[string]cacheEntry{}
	}
	if ttl <= 0 {
		ttl = githubCacheTTL
	}
	c.items[key] = cacheEntry{page: page, until: now.Add(ttl)}
}

func (c *Client) cache() *RepoCache {
	if c.Cache == nil {
		c.Cache = &RepoCache{ttl: githubCacheTTL}
	}
	return c.Cache
}

type githubSearchResp struct {
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

func (c *Client) searchGitHub(ctx context.Context, text string, page int) (Page, error) {
	key := text + "\n" + strconv.Itoa(page)
	if hit, ok := c.cache().get(key, c.now()); ok {
		items := append([]Item(nil), hit.Items...)
		for i := range items {
			items[i].Installed = false
		}
		c.markInstalled(items)
		return Page{Total: hit.Total, Items: items}, nil
	}
	base := c.GitHubURL
	if base == "" {
		base = githubAPI
	}
	q := url.Values{}
	q.Set("q", strings.TrimSpace(text)+" topic:"+stardewGHTopic)
	q.Set("sort", "stars")
	q.Set("per_page", strconv.Itoa(pageSize))
	q.Set("page", strconv.Itoa(page))
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+githubSearch+"?"+q.Encode(), nil)
	if err != nil {
		return Page{}, err
	}
	req.Header.Set("User-Agent", c.userAgent())
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Page{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return Page{}, ErrBusy
	}
	if resp.StatusCode != http.StatusOK {
		return Page{}, fmt.Errorf("github answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return Page{}, err
	}
	var parsed githubSearchResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Page{}, err
	}
	items := make([]Item, 0, len(parsed.Items))
	for _, n := range parsed.Items {
		items = append(items, Item{
			Source:  sourceGitHub,
			ID:      n.FullName,
			Name:    n.FullName,
			Summary: n.Description,
			Author:  n.Owner.Login,
			Picture: n.Owner.AvatarURL,
			Stars:   n.StargazersCount,
			Updated: n.UpdatedAt,
			URL:     n.HTMLURL,
		})
	}
	plain := Page{Total: parsed.TotalCount, Items: append([]Item(nil), items...)}
	c.cache().set(key, plain, c.now(), c.cache().ttl)
	c.markInstalled(items)
	return Page{Total: parsed.TotalCount, Items: items}, nil
}
