// Package modrinth is the Modrinth source driver, over the public API v2. No catalog game lists it yet, so it is
// registered but no game uses it; a game opts in with a source entry whose key is a facet such as "categories:fabric"
// or "project_type:mod" (a bare word means a category).
package modrinth

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/components"
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
	return d.do(ctx, http.MethodGet, path, params, nil, version, out)
}

func (d Driver) post(ctx context.Context, path string, body any, version string, out any) error {
	return d.do(ctx, http.MethodPost, path, nil, body, version, out)
}

func (d Driver) do(ctx context.Context, method, path string, params url.Values, body any, version string, out any) error {
	u := strings.TrimRight(cmp.Or(d.URL, BaseURL), "/") + path
	return source.DoJSON(ctx, source.Request{
		Service: "Modrinth", Client: d.HTTP, Method: method, URL: u, Params: params, Body: body,
		UserAgent: userAgent(version),
	}, out)
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

// versions lists the project's versions, only those for loaders when it names any.
func (d Driver) versions(ctx context.Context, project, mortarVersion string, loaders []string) ([]versionResp, error) {
	var out []versionResp
	var params url.Values
	if len(loaders) > 0 {
		encoded, err := json.Marshal(loaders)
		if err != nil {
			return nil, err
		}
		params = url.Values{"loaders": {string(encoded)}}
	}
	err := d.get(ctx, "/project/"+url.PathEscape(project)+"/version", params, mortarVersion, &out)
	return out, err
}

// Resolve finds the project's version, matching its number or id; an empty version means the newest. The file is
// the version's primary one. A game's loaders narrow the versions considered, so a dependency is not the build for
// another loader.
func (d Driver) Resolve(ctx context.Context, project, version, mortarVersion string, loaders []string) (Resolved, error) {
	all, err := d.versions(ctx, project, mortarVersion, loaders)
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
	all, err := d.versions(ctx, project, mortarVersion, nil)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(all))
	for i, v := range all {
		out[i] = v.VersionNumber
	}
	return out, nil
}

// listed holds the required dependencies of every version an update check read, keyed by project id and version
// number, so Dependencies answers without another request.
var listed sync.Map

func remember(v versionResp) {
	var ids []string
	for _, dep := range v.Dependencies {
		if dep.DependencyType == "required" && dep.ProjectID != "" {
			ids = append(ids, dep.ProjectID)
		}
	}
	listed.Store(source.VersionRef{ID: v.ProjectID, Version: v.VersionNumber}, ids)
}

func (v versionResp) hasFile(sha512 string) bool {
	for _, f := range v.Files {
		if strings.EqualFold(f.Hashes["sha512"], sha512) {
			return true
		}
	}
	return false
}

// Latest asks for the newest version of every file at once through POST /version_files/update, which takes the files'
// hashes and the loaders and game versions to keep to (https://docs.modrinth.com/api/operations/getlatestversionsfromhashes/).
// The installed versions of the files that have a newer one are then read through POST /version_files
// (https://docs.modrinth.com/api/operations/versionsfromhashes/), so Dependencies knows both sides.
func (d Driver) Latest(ctx context.Context, src components.GameSource, mortarVersion string, files []source.InstalledFile) (map[string]source.Latest, error) {
	out := map[string]source.Latest{}
	// Without loaders the site would answer with the newest file of any loader, which may not run in the game.
	if len(src.Loaders) == 0 {
		return out, nil
	}
	byHash := map[string]string{}
	for _, f := range files {
		if h, ok := strings.CutPrefix(f.Digest, "sha512:"); ok && h != "" {
			byHash[strings.ToLower(h)] = f.Digest
		}
	}
	if len(byHash) == 0 {
		return out, nil
	}
	hashes := slices.Sorted(maps.Keys(byHash))
	var newest map[string]versionResp
	if err := d.post(ctx, "/version_files/update", struct {
		Hashes       []string `json:"hashes"`
		Algorithm    string   `json:"algorithm"`
		Loaders      []string `json:"loaders"`
		GameVersions []string `json:"game_versions"`
	}{hashes, "sha512", nonNil(src.Loaders), nonNil(src.GameVersions)}, mortarVersion, &newest); err != nil {
		return nil, err
	}
	var older []string
	for h, v := range newest {
		h = strings.ToLower(h)
		digest, ok := byHash[h]
		if !ok || v.ID == "" || v.hasFile(h) {
			continue
		}
		remember(v)
		out[digest] = source.Latest{
			ProjectID: v.ProjectID, Version: v.VersionNumber, VersionID: v.ID,
			URL: siteURL + "/project/" + url.PathEscape(v.ProjectID) + "/version/" + url.PathEscape(v.ID),
		}
		older = append(older, h)
	}
	if len(older) > 0 {
		// The installed versions only add their dependencies to the answer; without them those stay unknown.
		_ = d.rememberInstalled(ctx, older, mortarVersion)
	}
	return out, nil
}

func (d Driver) rememberInstalled(ctx context.Context, hashes []string, mortarVersion string) error {
	slices.Sort(hashes)
	var installed map[string]versionResp
	if err := d.post(ctx, "/version_files", struct {
		Hashes    []string `json:"hashes"`
		Algorithm string   `json:"algorithm"`
	}{hashes, "sha512"}, mortarVersion, &installed); err != nil {
		return err
	}
	for _, v := range installed {
		remember(v)
	}
	return nil
}

// Dependencies answers from the versions update checks read: each known version's required projects, by id.
func (Driver) Dependencies(_ context.Context, _, _ string, refs []source.VersionRef) (map[source.VersionRef][]string, error) {
	out := map[source.VersionRef][]string{}
	for _, r := range refs {
		if ids, ok := listed.Load(r); ok {
			out[r], _ = ids.([]string)
		}
	}
	return out, nil
}

// nonNil keeps an absent filter an empty JSON list, which the API reads as no filter, rather than null.
func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
