// Package github is the GitHub source driver: repositories carrying one of the game's topics.
package github

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	ghauth "github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const (
	apiURL    = "https://api.github.com"
	searchURL = "/search/repositories"
	cacheTTL  = 10 * time.Minute
)

// Driver searches GitHub repositories. HTTP, URL, Web, Now, Topics and LoggedIn are for tests; the zero value talks to
// the real site.
type Driver struct {
	HTTP *http.Client
	URL  string
	Now  func() time.Time
	// Topics returns the game's topics; nil reads the catalog's key and topics for its GitHub source.
	Topics func(game string) []string
	// CacheDir is Mortar's cache directory, where the releases lists live; empty means the data folder's.
	CacheDir string
	// Web is github.com, whose search page is the last way to search.
	Web string
	// LoggedIn reports whether requests carry the user's gh token, which GraphQL needs; nil asks gh.
	LoggedIn func(context.Context) bool

	mu    sync.Mutex
	cache map[string]cacheEntry
	// Each way of searching keeps its last refusal while its wait lasts: asking again before then only extends the
	// limit, and a merged search asks once per topic.
	restRefused, graphRefused, webRefused *source.BusyError

	// webMu holds github.com's search pages to one at a time, webGap (or the default) apart.
	webMu   sync.Mutex
	webLast time.Time
	webGap  time.Duration
}

type cacheEntry struct {
	page  topicPage
	until time.Time
}

// topicPage is one page of a topic's search.
type topicPage struct {
	total int
	repos []repo
}

var _ = source.Register(&Driver{})

// ID is the catalog id.
func (*Driver) ID() string { return "github" }

// Name is the site's name.
func (*Driver) Name() string { return "GitHub" }

// Modes says Mortar downloads release assets itself.
func (*Driver) Modes() []source.Acquire { return []source.Acquire{source.Download} }

type searchResp struct {
	TotalCount int    `json:"total_count"`
	Items      []repo `json:"items"`
}

type repo struct {
	FullName        string   `json:"full_name"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	HTMLURL         string   `json:"html_url"`
	StargazersCount int      `json:"stargazers_count"`
	ForksCount      int      `json:"forks_count"`
	UpdatedAt       string   `json:"updated_at"`
	Topics          []string `json:"topics"`
	Owner           struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"owner"`
}

func (d *Driver) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now().UTC()
}

func (d *Driver) cached(key string) (topicPage, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.cache[key]
	if !ok || !d.now().Before(e.until) {
		return topicPage{}, false
	}
	return e.page, true
}

func (d *Driver) remember(key string, page topicPage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cache == nil {
		d.cache = map[string]cacheEntry{}
	}
	now := d.now()
	maps.DeleteFunc(d.cache, func(_ string, e cacheEntry) bool { return !now.Before(e.until) })
	d.cache[key] = cacheEntry{page: page, until: now.Add(cacheTTL)}
}

// MaxPageSize is the search API's largest per_page.
func (*Driver) MaxPageSize() int { return 100 }

// Sorts are the sorts the repository search honours. It has no download count (a repository's downloads are
// release-asset counts that need a request per repository), no name sort and none by creation date.
func (*Driver) Sorts() []string {
	return []string{source.SortStars, source.SortForks, source.SortUpdated}
}

// sortParam is the search's sort value; "" leaves GitHub's best match. With no text best match is arbitrary within
// the topic, so the star order is the default there.
func sortParam(sort, text string) string {
	switch sort {
	case source.SortForks:
		return "forks"
	case source.SortUpdated:
		return "updated"
	case source.SortStars:
		return "stars"
	}
	if strings.TrimSpace(text) == "" {
		return "stars"
	}
	return ""
}

// topics are the topics q searches: its key and the catalog's further topics for the game.
func (d *Driver) topics(q source.Query) []string {
	if d.Topics != nil {
		return d.Topics(q.Game)
	}
	topics := []string{q.Key}
	if g, ok := components.Game(q.Game); ok {
		if gs, ok := g.Source(d.ID()); ok {
			for _, t := range gs.Topics {
				if !slices.Contains(topics, t) {
					topics = append(topics, t)
				}
			}
		}
	}
	return topics
}

// Search lists repositories tagged with any of the game's topics, by q.Sort. Answers are cached for ten minutes
// because GitHub's anonymous search limit is small.
func (d *Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	// Without a topic the query would search every repository on GitHub, which is noise in a game's browse.
	if q.Key == "" {
		return source.Page{}, nil
	}
	topics := d.topics(q)
	terms := make([]string, 0, len(topics)+1)
	for _, t := range topics {
		terms = append(terms, topicTerms(q.Text, t))
	}
	if owner := ownerName(q.Text); owner != "" {
		terms = append(terms, "user:"+owner)
	}
	d.warm(ctx, q, terms)
	if len(topics) == 1 && ownerName(q.Text) == "" {
		got, err := d.searchTopic(ctx, q, topicTerms(q.Text, topics[0]), q.Page)
		if err != nil {
			return source.Page{}, err
		}
		return source.Page{Total: got.total, Items: d.items(got.repos)}, nil
	}
	return d.searchTopics(ctx, q, topics)
}

func topicTerms(text, topic string) string { return strings.TrimSpace(text + " topic:" + topic) }

var loginRE = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$`)

// ownerName is text when it could be a GitHub user or organisation name, else empty. GitHub's search text never
// matches a repository's owner, so such a word is also searched as one.
func ownerName(text string) string {
	if t := strings.TrimSpace(text); loginRE.MatchString(t) {
		return t
	}
	return ""
}

// searchTopics merges the topics' results by the query's sort and returns page q.Page of the merge. GitHub's search
// ORs only text, never topic: qualifiers, so each topic is its own search, read from its first page to q.Page because
// a merged page needs every topic's leading rows. A repository under several topics is listed once. A topic GitHub
// refuses is left out while another answers. A query that could be an owner adds that owner's first page of
// repositories carrying any of the topics, one request rather than one per topic. A part that fails is named in Failed,
// so the page is not cached as complete.
func (d *Driver) searchTopics(ctx context.Context, q source.Query, topics []string) (source.Page, error) {
	type ranked struct {
		repo        repo
		rank, topic int
	}
	var all []ranked
	seen := map[string]bool{}
	var errs []error
	var failed []string
	topicsFailed := 0
	sum, read := 0, 0
	for ti, topic := range topics {
		for p := source.FirstPage; p <= q.Page; p++ {
			got, err := d.searchTopic(ctx, q, topicTerms(q.Text, topic), p)
			if err != nil {
				errs, failed = append(errs, err), append(failed, "topic:"+topic)
				topicsFailed++
				break
			}
			if p == source.FirstPage {
				sum += got.total
			}
			for i, r := range got.repos {
				read++
				if name := strings.ToLower(r.FullName); !seen[name] {
					seen[name] = true
					all = append(all, ranked{r, (p-source.FirstPage)*q.Len() + i, ti})
				}
			}
			if len(got.repos) < q.Len() {
				break
			}
		}
	}
	if owner := ownerName(q.Text); owner != "" {
		got, err := d.searchTopic(ctx, q, "user:"+owner, source.FirstPage)
		if err != nil {
			errs, failed = append(errs, err), append(failed, "user:"+owner)
		}
		for i, r := range got.repos {
			name := strings.ToLower(r.FullName)
			if seen[name] || !slices.ContainsFunc(r.Topics, func(t string) bool { return slices.Contains(topics, t) }) {
				continue
			}
			seen[name] = true
			sum, read = sum+1, read+1
			all = append(all, ranked{r, i, len(topics)})
		}
	}
	if len(all) == 0 && topicsFailed == len(topics) {
		return source.Page{}, errs[0]
	}
	if len(failed) > 0 {
		log.Printf("github: search of %d topics: %d answered with %d repositories, %d failed (%s): %v",
			len(topics), len(topics)-len(failed), len(all), len(failed), strings.Join(failed, ","), errors.Join(errs...))
	}
	by := sortParam(q.Sort, q.Text)
	slices.SortStableFunc(all, func(a, b ranked) int {
		if c := compareBySort(by, a.repo, b.repo); c != 0 {
			return c
		}
		return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(a.topic, b.topic))
	})
	// The topics' totals count a shared repository once per topic; the overlap in the rows read scales that down.
	total := sum
	if read > 0 {
		total = sum * len(all) / read
	}
	start := (max(q.Page, source.FirstPage) - source.FirstPage) * q.Len()
	if start >= len(all) {
		return source.Page{Total: total, Failed: failed}, nil
	}
	repos := make([]repo, 0, q.Len())
	for _, r := range all[start:min(start+q.Len(), len(all))] {
		repos = append(repos, r.repo)
	}
	return source.Page{Total: total, Items: d.items(repos), Failed: failed}, nil
}

// compareBySort orders two repositories as the search's sort parameter does: negative when a comes first.
func compareBySort(by string, a, b repo) int {
	switch by {
	case "stars":
		return cmp.Compare(b.StargazersCount, a.StargazersCount)
	case "forks":
		return cmp.Compare(b.ForksCount, a.ForksCount)
	case "updated":
		return cmp.Compare(b.UpdatedAt, a.UpdatedAt)
	}
	return 0
}

// held is the refusal in *refused while it is in force, else nil.
func (d *Driver) held(refused **source.BusyError) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if *refused != nil && d.now().Before((*refused).Reset) {
		return *refused
	}
	return nil
}

// refuse records resp's refusal until GitHub's Retry-After, else its rate limit's reset, else a minute (the search
// limit's window).
func (d *Driver) refuse(refused **source.BusyError, resp *http.Response) error {
	busy := source.Busy("GitHub", resp)
	if busy.Reset.IsZero() {
		if at, err := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64); err == nil {
			busy.Reset = time.Unix(at, 0)
		}
	}
	if !d.now().Before(busy.Reset) {
		busy.Reset = d.now().Add(time.Minute)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	*refused = busy
	return busy
}

// searchTopic is page of the search for terms, sorted as q asks: cached, else from the REST search, else, while that
// refuses, from github.com's search page.
func (d *Driver) searchTopic(ctx context.Context, q source.Query, terms string, page int) (topicPage, error) {
	key := pageKey(terms, q, page)
	if hit, ok := d.cached(key); ok {
		return hit, nil
	}
	got, err := d.searchREST(ctx, q, terms, page, key)
	if !errors.Is(err, source.ErrBusy) {
		return got, err
	}
	web, werr := d.scrape(ctx, q, terms, page)
	if werr != nil {
		log.Printf("github: search page for %q: %v", terms, werr)
		return topicPage{}, err
	}
	d.remember(key, web)
	return web, nil
}

// searchREST is page of the search for terms from the REST search API.
func (d *Driver) searchREST(ctx context.Context, q source.Query, terms string, page int, key string) (topicPage, error) {
	if err := d.held(&d.restRefused); err != nil {
		return topicPage{}, err
	}
	base := cmp.Or(d.URL, apiURL)
	client := cmp.Or(d.HTTP, http.DefaultClient)
	params := url.Values{}
	params.Set("q", terms)
	if by := sortParam(q.Sort, q.Text); by != "" {
		params.Set("sort", by)
	}
	params.Set("per_page", strconv.Itoa(q.Len()))
	params.Set("page", strconv.Itoa(page))
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return topicPage{}, err
	}
	req.Header.Set("User-Agent", appversion.UserAgent(q.Version))
	req.Header.Set("Accept", "application/vnd.github+json")
	ghauth.DefaultAuth.Apply(req)
	resp, err := client.Do(req)
	if err != nil {
		return topicPage{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return topicPage{}, d.refuse(&d.restRefused, resp)
	}
	// GitHub refuses a user: qualifier naming no account; the word was then only search text.
	if resp.StatusCode == http.StatusUnprocessableEntity && strings.HasPrefix(terms, "user:") {
		d.remember(key, topicPage{})
		return topicPage{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return topicPage{}, fmt.Errorf("github answered %s", resp.Status)
	}
	body, err := fsx.ReadCapped(resp.Body, source.MaxBody)
	if err != nil {
		return topicPage{}, err
	}
	var parsed searchResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return topicPage{}, err
	}
	got := topicPage{total: parsed.TotalCount, repos: parsed.Items}
	d.remember(key, got)
	return got, nil
}

func (d *Driver) items(repos []repo) []source.Item {
	items := make([]source.Item, 0, len(repos))
	for _, n := range repos {
		items = append(items, source.Item{
			Source: d.ID(),
			ID:     n.FullName,
			// The owner is already the author line, so the title is the repository's own name.
			Name:    cmp.Or(n.Name, n.FullName),
			Summary: n.Description,
			Author:  n.Owner.Login,
			Picture: n.Owner.AvatarURL,
			Stars:   n.StargazersCount,
			Updated: n.UpdatedAt,
			URL:     n.HTMLURL,
			Repo:    n.FullName,
		})
	}
	return items
}
