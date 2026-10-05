// Package modrinth is the Modrinth source driver, over the public API v2. No catalog game lists it yet, so it is
// registered but no game uses it; a game opts in with a source entry whose key is a facet such as "categories:fabric"
// or "project_type:mod" (a bare word means a category).
package modrinth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// BaseURL is the real API.
const BaseURL = "https://api.modrinth.com/v2"

const siteURL = "https://modrinth.com"

// Driver searches Modrinth. HTTP and URL are for tests; the zero value talks to the real API.
type Driver struct {
	HTTP *http.Client
	URL  string
}

var _ = source.Register(Driver{})

// ID is the catalog id.
func (Driver) ID() string { return "modrinth" }

// Name is the site's name.
func (Driver) Name() string { return "Modrinth" }

// Modes says Mortar downloads version files itself.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Download} }

// Hosts is the site's web host.
func (Driver) Hosts() []string { return []string{"modrinth.com"} }

// ModPageURL is the project's page; id is the project id or slug, and gameKey does not enter the address.
func (Driver) ModPageURL(_, id string) string {
	if id == "" {
		return ""
	}
	return siteURL + "/project/" + url.PathEscape(id)
}

// userAgent is the identification Modrinth requires of every client.
func userAgent(version string) string {
	if version == "" {
		version = "dev"
	}
	return "Rethunk-Tech/mortar/" + version + " (+https://mortar.rethunk.tech)"
}

func (d Driver) get(ctx context.Context, path string, params url.Values, version string, out any) error {
	base := d.URL
	if base == "" {
		base = BaseURL
	}
	client := d.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	u := strings.TrimRight(base, "/") + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent(version))
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests {
		return source.ErrBusy
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("modrinth answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, source.MaxBody))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// facet turns the catalog key into one facet group; a key without a colon is a category.
func facet(key string) string {
	if strings.Contains(key, ":") {
		return key
	}
	return "categories:" + key
}

// sortIndex maps a Sort to Modrinth's index; Modrinth has no name sort, so that keeps relevance.
func sortIndex(sort string) string {
	switch sort {
	case source.SortDownloads:
		return "downloads"
	case source.SortUpdated:
		return "updated"
	case source.SortEndorsements:
		return "follows"
	}
	return "relevance"
}

type searchResp struct {
	TotalHits int `json:"total_hits"`
	Hits      []struct {
		ProjectID   string   `json:"project_id"`
		Slug        string   `json:"slug"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Author      string   `json:"author"`
		IconURL     string   `json:"icon_url"`
		Downloads   int      `json:"downloads"`
		Follows     int      `json:"follows"`
		Modified    string   `json:"date_modified"`
		Versions    []string `json:"versions"`
	} `json:"hits"`
}

// Search lists projects carrying the game's facet. Included categories are one OR group and each excluded category
// a negated facet; the game version in q.Version is Mortar's own, so it is not a filter.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	if q.Key == "" {
		return source.Page{}, fmt.Errorf("game %q has no Modrinth facet", q.Game)
	}
	facets := [][]string{{facet(q.Key)}}
	if len(q.Categories) > 0 {
		group := make([]string, len(q.Categories))
		for i, c := range q.Categories {
			group[i] = "categories:" + strings.ToLower(c)
		}
		facets = append(facets, group)
	}
	for _, c := range q.ExcludeCategories {
		facets = append(facets, []string{"categories!=" + strings.ToLower(c)})
	}
	raw, err := json.Marshal(facets)
	if err != nil {
		return source.Page{}, err
	}
	params := url.Values{}
	params.Set("query", q.Text)
	params.Set("facets", string(raw))
	params.Set("index", sortIndex(q.Sort))
	params.Set("limit", strconv.Itoa(source.PageSize))
	params.Set("offset", strconv.Itoa(max(q.Page-source.FirstPage, 0)*source.PageSize))
	var parsed searchResp
	if err := d.get(ctx, "/search", params, q.Version, &parsed); err != nil {
		return source.Page{}, err
	}
	items := make([]source.Item, 0, len(parsed.Hits))
	for _, h := range parsed.Hits {
		items = append(items, source.Item{
			Source: d.ID(), ID: h.ProjectID, Name: h.Title, Summary: h.Description, Author: h.Author,
			Picture: h.IconURL, Endorsements: h.Follows, Downloads: h.Downloads, Updated: h.Modified,
			URL: d.ModPageURL(q.Key, h.Slug),
		})
	}
	return source.Page{Total: parsed.TotalHits, Items: items}, nil
}

// Categories lists the site's category names across project types, sorted.
func (d Driver) Categories(ctx context.Context, _ string) ([]string, error) {
	var tags []struct {
		Name string `json:"name"`
	}
	if err := d.get(ctx, "/tag/category", nil, "", &tags); err != nil {
		return nil, err
	}
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return source.UniqueNames(names), nil
}

// Resolved is what an install needs of one project version.
type Resolved struct {
	ProjectID string
	Version   string
	FileName  string
	URL       string
	Size      int64
	// Digest is "sha512:<hex>" of the file, as Modrinth publishes it.
	Digest string
	// Dependencies are the projects this version requires, each at the version it names (empty means the newest).
	Dependencies []Dependency
}

// Dependency is a required project.
type Dependency struct{ ProjectID, VersionID string }

type versionResp struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	VersionNumber string `json:"version_number"`
	Dependencies  []struct {
		ProjectID      string `json:"project_id"`
		VersionID      string `json:"version_id"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependencies"`
	Files []struct {
		Hashes   map[string]string `json:"hashes"`
		URL      string            `json:"url"`
		Filename string            `json:"filename"`
		Primary  bool              `json:"primary"`
		Size     int64             `json:"size"`
	} `json:"files"`
}

func (d Driver) versions(ctx context.Context, project, mortarVersion string) ([]versionResp, error) {
	var out []versionResp
	err := d.get(ctx, "/project/"+url.PathEscape(project)+"/version", nil, mortarVersion, &out)
	return out, err
}

// Resolve finds the project's version, matching its number or id; an empty version means the newest. The file is
// the version's primary one.
func (d Driver) Resolve(ctx context.Context, project, version, mortarVersion string) (Resolved, error) {
	all, err := d.versions(ctx, project, mortarVersion)
	if err != nil {
		return Resolved{}, err
	}
	for _, v := range all {
		if version != "" && v.VersionNumber != version && v.ID != version {
			continue
		}
		if len(v.Files) == 0 {
			return Resolved{}, fmt.Errorf("%s %s has no files", project, v.VersionNumber)
		}
		f := v.Files[0]
		for _, c := range v.Files {
			if c.Primary {
				f = c
				break
			}
		}
		digest := ""
		if h := f.Hashes["sha512"]; h != "" {
			digest = "sha512:" + h
		}
		res := Resolved{ProjectID: v.ProjectID, Version: v.VersionNumber, FileName: f.Filename, URL: f.URL, Size: f.Size, Digest: digest}
		for _, dep := range v.Dependencies {
			if dep.DependencyType == "required" && dep.ProjectID != "" {
				res.Dependencies = append(res.Dependencies, Dependency{ProjectID: dep.ProjectID, VersionID: dep.VersionID})
			}
		}
		return res, nil
	}
	return Resolved{}, fmt.Errorf("%s %s is not on Modrinth", project, version)
}

// Versions lists the project's version numbers, newest first.
func (d Driver) Versions(ctx context.Context, project, mortarVersion string) ([]string, error) {
	all, err := d.versions(ctx, project, mortarVersion)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(all))
	for i, v := range all {
		out[i] = v.VersionNumber
	}
	return out, nil
}
