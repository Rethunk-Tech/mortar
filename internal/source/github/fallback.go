package github

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	ghauth "github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// A search is answered by the most favourable way open: the cache, GraphQL when the user's gh login lends a token
// (one request for every topic, a hundred rows each), the REST search, and last github.com's own search page, which
// a mod browser may read for a basic topic search as long as it does so sparingly: one page at a time, webGap apart,
// waiting out any refusal.

const (
	webURL = "https://github.com"
	// graphRows is the most rows one GraphQL search connection returns; asking for fewer costs the same.
	graphRows = 100
	// webRows is how many repositories a page of github.com's search holds.
	webRows = 10
	webGap  = 500 * time.Millisecond
)

// graphRepo asks only for what a result card and the merge's sorts read. The owner search alone names topics, to
// keep an owner's repositories for other games out.
const graphRepo = `nameWithOwner name description stargazerCount forkCount updatedAt owner { login avatarUrl }`

type graphNode struct {
	NameWithOwner  string `json:"nameWithOwner"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	StargazerCount int    `json:"stargazerCount"`
	ForkCount      int    `json:"forkCount"`
	UpdatedAt      string `json:"updatedAt"`
	Owner          struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"owner"`
	RepositoryTopics struct {
		Nodes []struct {
			Topic struct {
				Name string `json:"name"`
			} `json:"topic"`
		} `json:"nodes"`
	} `json:"repositoryTopics"`
}

type graphSearch struct {
	RepositoryCount int         `json:"repositoryCount"`
	Nodes           []graphNode `json:"nodes"`
}

func (n graphNode) repo() repo {
	r := repo{
		FullName:        n.NameWithOwner,
		Name:            n.Name,
		Description:     n.Description,
		HTMLURL:         webURL + "/" + n.NameWithOwner,
		StargazersCount: n.StargazerCount,
		ForksCount:      n.ForkCount,
		UpdatedAt:       n.UpdatedAt,
	}
	r.Owner.Login, r.Owner.AvatarURL = n.Owner.Login, n.Owner.AvatarURL
	for _, t := range n.RepositoryTopics.Nodes {
		r.Topics = append(r.Topics, t.Topic.Name)
	}
	return r
}

// loggedIn reports whether a request to the API would carry the gh token, which Apply sends to GitHub alone.
func (d *Driver) loggedIn(ctx context.Context) bool {
	if d.LoggedIn != nil {
		return d.LoggedIn(ctx)
	}
	u, err := url.Parse(cmp.Or(d.URL, apiURL))
	return err == nil && ghauth.IsGitHubHost(u.Hostname()) && ghauth.DefaultAuth.LoggedIn(ctx)
}

func pageKey(terms string, q source.Query, page int) string {
	return terms + "\n" + q.Sort + "\n" + strconv.Itoa(page) + "\n" + strconv.Itoa(q.Len())
}

// warm fills the cache with every page up to q.Page of each of terms that is not cached yet, in one GraphQL request,
// when the user is signed in to gh. Anything it cannot answer is left to searchTopic.
func (d *Driver) warm(ctx context.Context, q source.Query, terms []string) {
	if q.Page*q.Len() > graphRows || !d.loggedIn(ctx) || d.held(&d.graphRefused) != nil {
		return
	}
	var want []string
	for _, t := range terms {
		if _, ok := d.cached(pageKey(t, q, q.Page)); !ok {
			want = append(want, t)
		}
	}
	if len(want) == 0 {
		return
	}
	got, err := d.graphQL(ctx, q, want)
	if err != nil {
		log.Printf("github: GraphQL search of %d queries: %v", len(want), err)
		return
	}
	for i, t := range want {
		s, ok := got[i]
		if !ok {
			continue
		}
		rows := make([]repo, 0, len(s.Nodes))
		for _, n := range s.Nodes {
			rows = append(rows, n.repo())
		}
		// A full answer may go on past what was read, so only its whole pages are known; a short one is the whole
		// search, so every page the request covered is known, empty ones included.
		pages := len(rows) / q.Len()
		if len(rows) < graphRows {
			pages = (graphRows + q.Len() - 1) / q.Len()
		}
		for p := range pages {
			d.remember(pageKey(t, q, source.FirstPage+p), topicPage{
				total: s.RepositoryCount,
				repos: rows[min(p*q.Len(), len(rows)):min((p+1)*q.Len(), len(rows))],
			})
		}
	}
}

// graphQL runs one search per terms, as aliases of one request, keyed by their index in terms.
func (d *Driver) graphQL(ctx context.Context, q source.Query, terms []string) (map[int]graphSearch, error) {
	var b strings.Builder
	vars := map[string]any{}
	b.WriteString("query(")
	for i := range terms {
		fmt.Fprintf(&b, "$q%d: String!,", i)
	}
	b.WriteString(") {")
	by := sortParam(q.Sort, q.Text)
	for i, t := range terms {
		fields := graphRepo
		if strings.HasPrefix(t, "user:") {
			fields += ` repositoryTopics(first: 20) { nodes { topic { name } } }`
		}
		fmt.Fprintf(&b, " s%d: search(type: REPOSITORY, query: $q%d, first: %d) { repositoryCount nodes { ... on Repository { %s } } }",
			i, i, graphRows, fields)
		if by != "" {
			t += " sort:" + by
		}
		vars["q"+strconv.Itoa(i)] = t
	}
	b.WriteString(" }")
	body, err := json.Marshal(map[string]any{"query": b.String(), "variables": vars})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cmp.Or(d.URL, apiURL), "/")+"/graphql", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", appversion.UserAgent(q.Version))
	req.Header.Set("Content-Type", "application/json")
	ghauth.DefaultAuth.Apply(req)
	resp, err := cmp.Or(d.HTTP, http.DefaultClient).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, d.refuse(&d.graphRefused, resp)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github GraphQL answered %s", resp.Status)
	}
	raw, err := fsx.ReadCapped(resp.Body, source.MaxBody)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data   map[string]*graphSearch `json:"data"`
		Errors []struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	for _, e := range parsed.Errors {
		if e.Type == "RATE_LIMITED" {
			return nil, d.refuse(&d.graphRefused, resp)
		}
	}
	out := map[int]graphSearch{}
	for i := range terms {
		if s := parsed.Data["s"+strconv.Itoa(i)]; s != nil {
			out[i] = *s
		}
	}
	if len(out) == 0 && len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("github GraphQL: %s", parsed.Errors[0].Message)
	}
	return out, nil
}

type webResult struct {
	Description string   `json:"hl_trunc_description"`
	Followers   int      `json:"followers"`
	Topics      []string `json:"topics"`
	Repo        struct {
		Repository struct {
			Name      string `json:"name"`
			OwnerID   int64  `json:"owner_id"`
			OwnerName string `json:"owner_login"`
			UpdatedAt string `json:"updated_at"`
		} `json:"repository"`
	} `json:"repo"`
}

var tagRE = regexp.MustCompile(`<[^>]*>`)

func (w webResult) repo() repo {
	r := w.Repo.Repository
	full := r.OwnerName + "/" + r.Name
	out := repo{
		FullName:        full,
		Name:            r.Name,
		Description:     html.UnescapeString(tagRE.ReplaceAllString(w.Description, "")),
		HTMLURL:         webURL + "/" + full,
		StargazersCount: w.Followers,
		UpdatedAt:       r.UpdatedAt,
		Topics:          w.Topics,
	}
	out.Owner.Login = r.OwnerName
	out.Owner.AvatarURL = "https://avatars.githubusercontent.com/u/" + strconv.FormatInt(r.OwnerID, 10) + "?v=4"
	return out
}

// scrape is page of the search for terms read from github.com's search pages, which hold webRows each.
func (d *Driver) scrape(ctx context.Context, q source.Query, terms string, page int) (topicPage, error) {
	start := (max(page, source.FirstPage) - source.FirstPage) * q.Len()
	var got topicPage
	var rows []repo
	for p := start/webRows + 1; p <= (start+q.Len()-1)/webRows+1; p++ {
		part, err := d.webPage(ctx, q, terms, p)
		if err != nil {
			return topicPage{}, err
		}
		got.total = part.total
		rows = append(rows, part.repos...)
		if len(part.repos) < webRows {
			break
		}
	}
	skip := start % webRows
	got.repos = rows[min(skip, len(rows)):min(skip+q.Len(), len(rows))]
	return got, nil
}

// webPage reads one github.com search page, no sooner than webGap after the last.
func (d *Driver) webPage(ctx context.Context, q source.Query, terms string, page int) (topicPage, error) {
	d.webMu.Lock()
	defer d.webMu.Unlock()
	if err := d.held(&d.webRefused); err != nil {
		return topicPage{}, err
	}
	if wait := time.Until(d.webLast.Add(cmp.Or(d.webGap, webGap))); wait > 0 {
		select {
		case <-ctx.Done():
			return topicPage{}, ctx.Err()
		case <-time.After(wait):
		}
	}
	defer func() { d.webLast = time.Now() }()
	params := url.Values{"q": {terms}, "type": {"repositories"}, "p": {strconv.Itoa(page)}}
	if by := sortParam(q.Sort, q.Text); by != "" {
		params.Set("s", by)
		params.Set("o", "desc")
	}
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cmp.Or(d.Web, webURL), "/")+"/search?"+params.Encode(), nil)
	if err != nil {
		return topicPage{}, err
	}
	req.Header.Set("User-Agent", appversion.UserAgent(q.Version))
	req.Header.Set("Accept", "application/json")
	resp, err := cmp.Or(d.HTTP, http.DefaultClient).Do(req)
	if err != nil {
		return topicPage{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return topicPage{}, d.refuse(&d.webRefused, resp)
	}
	if resp.StatusCode != http.StatusOK {
		return topicPage{}, fmt.Errorf("github.com search answered %s", resp.Status)
	}
	raw, err := fsx.ReadCapped(resp.Body, source.MaxBody)
	if err != nil {
		return topicPage{}, err
	}
	var parsed struct {
		Payload struct {
			Search struct {
				Count   int         `json:"result_count"`
				Results []webResult `json:"results"`
			} `json:"blackbirdSearchRoute"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return topicPage{}, fmt.Errorf("github.com search page: %w", err)
	}
	out := topicPage{total: parsed.Payload.Search.Count}
	for _, r := range parsed.Payload.Search.Results {
		out.repos = append(out.repos, r.repo())
	}
	return out, nil
}
