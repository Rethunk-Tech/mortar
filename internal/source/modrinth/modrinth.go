// Package modrinth is the Modrinth source driver, over the public API v2. No catalog game lists it yet, so it is
// registered but no game uses it; a game opts in with a source entry whose key is a facet such as "categories:fabric"
// or "project_type:mod" (a bare word means a category).
package modrinth

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit"
	gomodrinth "github.com/Rethunk-Tech/go-modrinth"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

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

// api is the API client for one call, identifying Mortar as the site asks.
func (d Driver) api(version string) *gomodrinth.Client {
	return gomodrinth.New(gomodrinth.Config{BaseURL: d.URL, HTTP: d.HTTP, UserAgent: userAgent(version)})
}

// bounded holds one library call to the timeout every Mortar source request carries.
func bounded(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, source.RequestTimeout)
}

// apiErr reads the library's answer as the source error whose wording a player already sees, so a status reads
// "Modrinth answered …" and a refusal is the busy state the queue backs off on.
func apiErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, httpkit.ErrRateLimited) {
		return &source.BusyError{Source: "Modrinth"}
	}
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return &source.StatusError{Service: "Modrinth", Code: se.Code, Status: se.Status}
	}
	return err
}

// PageURL is the project's page address, whose last segment is its slug.
func (d Driver) PageURL(ctx context.Context, id string) (string, error) {
	ctx, cancel := bounded(ctx)
	defer cancel()
	project, err := d.api("").Project(ctx, id)
	if err != nil {
		return "", apiErr(err)
	}
	return d.ModPageURL("", project.Slug), nil
}

// ModPageURL is the project's page; id is the project id or slug, and gameKey does not enter the address.
func (Driver) ModPageURL(_, id string) string {
	if id == "" {
		return ""
	}
	return gomodrinth.PageURL("project", id)
}

// userAgent is the identification Modrinth requires of every client.
func userAgent(version string) string {
	version = cmp.Or(version, "dev")
	return "Rethunk-Tech/mortar/" + version + " (+https://mortar.rethunk.tech)"
}

// facet turns the catalog key into one facet group; a key without a colon is a category.
func facet(key string) string {
	if strings.Contains(key, ":") {
		return key
	}
	return "categories:" + key
}

// MaxPageSize is the search API's largest limit.
func (Driver) MaxPageSize() int { return gomodrinth.MaxPageSize }

// Sorts are Modrinth's search indexes other than relevance; it has no name sort.
func (Driver) Sorts() []string {
	return []string{source.SortDownloads, source.SortEndorsements, source.SortUpdated, source.SortNewest}
}

// sortIndex maps a Sort to Modrinth's index.
func sortIndex(sort string) string {
	switch sort {
	case source.SortDownloads:
		return gomodrinth.IndexDownloads
	case source.SortUpdated:
		return gomodrinth.IndexUpdated
	case source.SortEndorsements:
		return gomodrinth.IndexFollows
	case source.SortNewest:
		return gomodrinth.IndexNewest
	}
	return gomodrinth.IndexRelevance
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
	found, err := d.search(ctx, q, facets)
	if err != nil {
		return source.Page{}, apiErr(err)
	}
	latest := make([]string, 0, len(found.Hits))
	for _, h := range found.Hits {
		if h.Latest != "" {
			latest = append(latest, h.Latest)
		}
	}
	sizes := d.primarySizes(ctx, latest, q.Version)
	items := make([]source.Item, 0, len(found.Hits))
	for _, h := range found.Hits {
		items = append(items, source.Item{
			Source: d.ID(), ID: h.ProjectID, Name: h.Title, Summary: h.Description, Author: h.Author,
			Picture: h.IconURL, Endorsements: h.Follows, Downloads: h.Downloads, Updated: h.Modified,
			Size: sizes[h.Latest], URL: d.ModPageURL(q.Key, h.Slug),
		})
	}
	return source.Page{Total: found.Total, Items: items}, nil
}

func (d Driver) search(ctx context.Context, q source.Query, facets [][]string) (gomodrinth.SearchResult, error) {
	ctx, cancel := bounded(ctx)
	defer cancel()
	return d.api(q.Version).Search(ctx, gomodrinth.SearchQuery{
		Text: q.Text, Facets: facets, Index: sortIndex(q.Sort),
		Offset: max(q.Page-source.FirstPage, 0) * q.Len(), Limit: q.Len(),
	})
}

// Categories lists the site's category names across project types, sorted.
func (d Driver) Categories(ctx context.Context, _ string) ([]string, error) {
	ctx, cancel := bounded(ctx)
	defer cancel()
	tags, err := d.api("").Categories(ctx)
	if err != nil {
		return nil, apiErr(err)
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

// primarySizes is each version's primary file size by version id, read in one request; a failed read sizes none, so
// the cards leave the size out.
func (d Driver) primarySizes(ctx context.Context, ids []string, mortarVersion string) map[string]int64 {
	out := map[string]int64{}
	if len(ids) == 0 {
		return out
	}
	ctx, cancel := bounded(ctx)
	defer cancel()
	versions, err := d.api(mortarVersion).Versions(ctx, ids)
	if err != nil {
		log.Printf("modrinth: file sizes: %v", apiErr(err))
		return out
	}
	for _, v := range versions {
		if f, ok := v.PrimaryFile(); ok {
			out[v.ID] = f.Size
		}
	}
	return out
}

// versions lists the project's versions, newest first, only those for loaders when it names any.
func (d Driver) versions(ctx context.Context, project, mortarVersion string, loaders []string) ([]gomodrinth.Version, error) {
	ctx, cancel := bounded(ctx)
	defer cancel()
	return d.api(mortarVersion).ProjectVersions(ctx, project, loaders, nil)
}

// Resolve finds the project's version, matching its number or id; an empty version means the newest. The file is
// the version's primary one. A game's loaders narrow the versions considered, so a dependency is not the build for
// another loader.
func (d Driver) Resolve(ctx context.Context, project, version, mortarVersion string, loaders []string) (Resolved, error) {
	all, err := d.versions(ctx, project, mortarVersion, loaders)
	if err != nil {
		return Resolved{}, apiErr(err)
	}
	for _, v := range all {
		if version != "" && v.VersionNumber != version && v.ID != version {
			continue
		}
		f, ok := v.PrimaryFile()
		if !ok {
			return Resolved{}, fmt.Errorf("%s %s has no files", project, v.VersionNumber)
		}
		digest := ""
		if h := f.Hashes["sha512"]; h != "" {
			digest = "sha512:" + h
		}
		res := Resolved{ProjectID: v.ProjectID, Version: v.VersionNumber, FileName: f.Filename, URL: f.URL, Size: f.Size, Digest: digest}
		for _, dep := range v.Dependencies {
			if dep.Type == "required" && dep.ProjectID != "" {
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
		return nil, apiErr(err)
	}
	out := make([]string, len(all))
	for i, v := range all {
		out[i] = v.VersionNumber
	}
	return out, nil
}

// listed holds the required dependencies of versions an update check read, keyed by project id and version
// number, so Dependencies answers without another request. Oldest keys drop once the cap is hit.
var listed source.ListedStore

func remember(v gomodrinth.Version) {
	var ids []string
	for _, dep := range v.Dependencies {
		if dep.Type == "required" && dep.ProjectID != "" {
			ids = append(ids, dep.ProjectID)
		}
	}
	listed.Put(source.VersionRef{ID: v.ProjectID, Version: v.VersionNumber}, ids)
}

func hasFile(v gomodrinth.Version, sha512 string) bool {
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
	// One request per update channel: the version types a file accepts depend on its mod's channel.
	groups := map[int][]string{}
	for _, f := range files {
		if h, ok := strings.CutPrefix(f.Digest, "sha512:"); ok && h != "" {
			h = strings.ToLower(h)
			byHash[h] = f.Digest
			stability := source.Stability(f.Channel)
			groups[stability] = append(groups[stability], h)
		}
	}
	if len(byHash) == 0 {
		return out, nil
	}
	newest := map[string]gomodrinth.Version{}
	for _, stability := range slices.Sorted(maps.Keys(groups)) {
		var got map[string]gomodrinth.Version
		// The version types a channel accepts are not part of the library's lookup, so this request is sent here.
		if err := source.DoJSON(ctx, source.Request{
			Service: "Modrinth", Client: d.HTTP, Method: http.MethodPost,
			URL: strings.TrimRight(cmp.Or(d.URL, gomodrinth.BaseURL), "/") + "/version_files/update",
			Body: struct {
				Hashes       []string `json:"hashes"`
				Algorithm    string   `json:"algorithm"`
				Loaders      []string `json:"loaders"`
				GameVersions []string `json:"game_versions"`
				VersionTypes []string `json:"version_types"`
			}{slices.Sorted(slices.Values(groups[stability])), "sha512", usererr.NonNil(src.Loaders), usererr.NonNil(src.GameVersions), versionTypes[:stability]},
			UserAgent: userAgent(mortarVersion),
		}, &got); err != nil {
			return nil, err
		}
		maps.Copy(newest, got)
	}
	var older []string
	for h, v := range newest {
		h = strings.ToLower(h)
		digest, ok := byHash[h]
		if !ok || v.ID == "" || hasFile(v, h) {
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
	ctx, cancel := bounded(ctx)
	defer cancel()
	installed, err := d.api(mortarVersion).VersionsByHash(ctx, "sha512", hashes)
	if err != nil {
		return apiErr(err)
	}
	for _, v := range installed {
		remember(v)
	}
	return nil
}

// Dependencies answers from the versions update checks read: each known version's required projects, by id.
func (Driver) Dependencies(_ context.Context, _, _ string, refs []source.VersionRef) (map[source.VersionRef][]string, error) {
	return listed.GetAll(refs), nil
}

// versionTypes are Modrinth's version types from the most to the least stable, which a channel's stability cuts off.
var versionTypes = []string{"release", "beta", "alpha"}

// Details reads the project's page: its description (already Markdown), categories, every version number and the
// required projects of the newest version.
func (d Driver) Details(ctx context.Context, _, id, mortarVersion string) (source.Details, error) {
	pageCtx, cancel := bounded(ctx)
	project, err := d.api(mortarVersion).Project(pageCtx, id)
	cancel()
	if err != nil {
		return source.Details{}, apiErr(err)
	}
	all, err := d.versions(ctx, id, mortarVersion, nil)
	if err != nil {
		return source.Details{}, apiErr(err)
	}
	out := source.Details{
		Description: project.Body, Versions: []string{}, Dependencies: []string{},
		Categories: source.UniqueNames(append(project.Categories, project.AdditionalCategories...)),
	}
	for _, v := range all {
		out.Versions = append(out.Versions, v.VersionNumber)
		out.Released = append(out.Released, source.Released(v.DatePublished))
	}
	if len(all) > 0 {
		for _, dep := range all[0].Dependencies {
			if dep.Type == "required" && dep.ProjectID != "" {
				out.Dependencies = append(out.Dependencies, dep.ProjectID)
			}
		}
	}
	return out, nil
}
