// Package vsmoddb is the Vintage Story Mod DB source driver, over the site's public API
// (https://mods.vintagestory.at/api). A mod's id is "<modid>-<assetid>": the API reads a mod by its modid, and its page
// is addressed by its assetid, and neither answers to the other.
package vsmoddb

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// BaseURL is the real API.
const BaseURL = "https://mods.vintagestory.at/api"

const siteURL = "https://mods.vintagestory.at"

// Driver searches the Mod DB. HTTP and URL are for tests; the zero value talks to the real API.
type Driver struct {
	HTTP *http.Client
	URL  string
}

var _ = source.Register(Driver{})

// ID is the catalog id.
func (Driver) ID() string { return "vsmoddb" }

// Name is the site's name.
func (Driver) Name() string { return "Vintage Story Mod DB" }

// Modes says Mortar downloads release files itself.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Download} }

// Hosts is the site's web host.
func (Driver) Hosts() []string { return []string{"mods.vintagestory.at"} }

// ID joins a mod's modid and assetid into the id Mortar keeps.
func ID(modID, assetID int) string { return strconv.Itoa(modID) + "-" + strconv.Itoa(assetID) }

// splitID is the modid and assetid of id; an id without an assetid still names its mod.
func splitID(id string) (modID, assetID string, ok bool) {
	modID, assetID, _ = strings.Cut(id, "-")
	if _, err := strconv.Atoi(modID); err != nil {
		return "", "", false
	}
	if _, err := strconv.Atoi(assetID); err != nil {
		assetID = ""
	}
	return modID, assetID, true
}

// ModPageURL is the mod's page; gameKey does not enter the address.
func (Driver) ModPageURL(_, id string) string {
	if _, asset, ok := splitID(id); ok && asset != "" {
		return siteURL + "/show/mod/" + asset
	}
	return ""
}

func (d Driver) get(ctx context.Context, path string, params url.Values, out any) error {
	return source.DoJSON(ctx, source.Request{
		Service: "the Vintage Story Mod DB", Client: d.HTTP, URL: strings.TrimRight(cmp.Or(d.URL, BaseURL), "/") + path, Params: params,
	}, out)
}

// MaxPageSize is a page of the listing; the API takes any limit.
func (Driver) MaxPageSize() int { return 100 }

// Sorts are the Mod DB's orders other than its own; it has no name order.
func (Driver) Sorts() []string {
	return []string{source.SortDownloads, source.SortEndorsements, source.SortUpdated, source.SortNewest}
}

func orderBy(sort string) string {
	switch sort {
	case source.SortDownloads:
		return "downloads"
	case source.SortEndorsements:
		return "follows"
	case source.SortUpdated:
		return "lastreleased"
	case source.SortNewest:
		return "asset.created"
	}
	return ""
}

type listedMod struct {
	ModID        int      `json:"modid"`
	AssetID      int      `json:"assetid"`
	Downloads    int      `json:"downloads"`
	Follows      int      `json:"follows"`
	Name         string   `json:"name"`
	Summary      string   `json:"summary"`
	Author       string   `json:"author"`
	URLAlias     string   `json:"urlalias"`
	Logo         string   `json:"logo"`
	Tags         []string `json:"tags"`
	LastReleased string   `json:"lastreleased"`
}

// stamp turns the site's "2006-01-02 15:04:05" (UTC) into RFC 3339, as the other sources give it.
func stamp(s string) string {
	t, err := time.Parse(time.DateTime, s)
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// pageURL prefers the author's alias, the address the site itself links.
func pageURL(alias string, assetID int) string {
	if alias != "" {
		return siteURL + "/" + url.PathEscape(alias)
	}
	return siteURL + "/show/mod/" + strconv.Itoa(assetID)
}

// Search lists the site's mods. The API has no offset, only a limit, so page n asks for n pages and keeps the last.
// Included categories go to the site as tag ids; excluded ones are dropped from the page and counted in Hidden.
// ponytail: the total is unknown, so a full page reports one more hit than it shows and the next page is offered;
// read a count if the API ever gives one.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	page := max(q.Page-source.FirstPage, 0) + 1
	params := url.Values{"limit": {strconv.Itoa(page * q.Len())}}
	if q.Text != "" {
		params.Set("text", q.Text)
	}
	by := orderBy(q.Sort)
	if by == "" && q.Text != "" {
		// The site has no relevance order, and its own order puts a word found in any description ahead of the mod
		// named by the search; the most downloaded matches come first instead.
		by = "downloads"
	}
	if by != "" {
		params.Set("orderby", by)
		params.Set("orderdirection", "desc")
	}
	if len(q.Categories) > 0 {
		ids, err := d.tagIDs(ctx)
		if err != nil {
			return source.Page{}, err
		}
		for _, c := range q.Categories {
			if id, ok := ids[strings.ToLower(c)]; ok {
				params.Add("tagids[]", id)
			}
		}
	}
	var parsed struct {
		Mods []listedMod `json:"mods"`
	}
	if err := d.get(ctx, "/mods", params, &parsed); err != nil {
		return source.Page{}, err
	}
	total := len(parsed.Mods)
	if total == page*q.Len() {
		total++
	}
	start := min((page-1)*q.Len(), len(parsed.Mods))
	out := source.Page{Total: total, Items: []source.Item{}}
	for _, m := range parsed.Mods[start:] {
		if !source.CategoryMatch(m.Tags, nil, q.ExcludeCategories) {
			out.Hidden++
			continue
		}
		out.Items = append(out.Items, source.Item{
			Source: d.ID(), ID: ID(m.ModID, m.AssetID), Name: m.Name, Summary: m.Summary, Author: m.Author,
			Picture: m.Logo, Endorsements: m.Follows, Downloads: m.Downloads, Updated: stamp(m.LastReleased),
			URL: pageURL(m.URLAlias, m.AssetID),
		})
	}
	return out, nil
}

// tags caches the site's tag names and their ids by lower-cased name for the life of the process; the list changes
// rarely.
var tags struct {
	sync.Mutex
	ids   map[string]string
	names []string
}

func (d Driver) tagIDs(ctx context.Context) (map[string]string, error) {
	tags.Lock()
	defer tags.Unlock()
	if tags.ids != nil {
		return tags.ids, nil
	}
	var parsed struct {
		Tags []struct {
			ID   string `json:"tagid"`
			Name string `json:"name"`
		} `json:"tags"`
	}
	if err := d.get(ctx, "/tags", nil, &parsed); err != nil {
		return nil, err
	}
	ids := make(map[string]string, len(parsed.Tags))
	names := make([]string, 0, len(parsed.Tags))
	for _, t := range parsed.Tags {
		ids[strings.ToLower(t.Name)] = t.ID
		names = append(names, t.Name)
	}
	tags.ids, tags.names = ids, source.UniqueNames(names)
	return ids, nil
}

// Categories lists the site's tag names.
func (d Driver) Categories(ctx context.Context, _ string) ([]string, error) {
	if _, err := d.tagIDs(ctx); err != nil {
		return nil, err
	}
	tags.Lock()
	defer tags.Unlock()
	return tags.names, nil
}

type release struct {
	ReleaseID  int      `json:"releaseid"`
	MainFile   string   `json:"mainfile"`
	FileName   string   `json:"filename"`
	ModVersion string   `json:"modversion"`
	ModIDStr   string   `json:"modidstr"`
	Tags       []string `json:"tags"`
	Created    string   `json:"created"`
	Changelog  string   `json:"changelog"`
}

type modDetail struct {
	ModID    int       `json:"modid"`
	AssetID  int       `json:"assetid"`
	Name     string    `json:"name"`
	Author   string    `json:"author"`
	URLAlias string    `json:"urlalias"`
	Logo     string    `json:"logofile"`
	Text     string    `json:"text"`
	Tags     []string  `json:"tags"`
	Releases []release `json:"releases"`
}

// mod reads one mod by its modid, or by the mod id its modinfo.json names.
func (d Driver) mod(ctx context.Context, ref string) (modDetail, error) {
	var parsed struct {
		Mod *modDetail `json:"mod"`
	}
	if err := d.get(ctx, "/mod/"+url.PathEscape(ref), nil, &parsed); err != nil {
		return modDetail{}, err
	}
	if parsed.Mod == nil {
		return modDetail{}, fmt.Errorf("%s is not on the Vintage Story Mod DB", ref)
	}
	return *parsed.Mod, nil
}

// preRelease reports a version its author marks as a test build (1.0.0-rc.1, 2.0.0-pre.8, 1.2-dev.3).
func preRelease(version string) bool { return strings.Contains(version, "-") }

// newest is the newest release a player on stable builds should get: the newest stable one, else the newest of all.
// The site lists releases newest first.
func (m modDetail) newest() (release, bool) {
	for _, r := range m.Releases {
		if !preRelease(r.ModVersion) && r.MainFile != "" {
			return r, true
		}
	}
	for _, r := range m.Releases {
		if r.MainFile != "" {
			return r, true
		}
	}
	return release{}, false
}

// Resolved is what an install needs of one release.
type Resolved struct {
	ID, Name, Version, FileName, URL, Picture string
	ReleaseID                                 int
}

// Resolve finds the mod's release whose version or release id is version; an empty version means the newest stable
// release. id is a Mortar id, a modid, or the mod id a modinfo.json names.
func (d Driver) Resolve(ctx context.Context, id, version string) (Resolved, error) {
	ref := id
	if modID, _, ok := splitID(id); ok {
		ref = modID
	}
	m, err := d.mod(ctx, ref)
	if err != nil {
		return Resolved{}, err
	}
	pick, ok := m.newest()
	if version != "" {
		ok = false
		for _, r := range m.Releases {
			if r.ModVersion == version || strconv.Itoa(r.ReleaseID) == version {
				pick, ok = r, r.MainFile != ""
				break
			}
		}
	}
	if !ok {
		return Resolved{}, fmt.Errorf("%s has no release %s on the Vintage Story Mod DB", m.Name, cmp.Or(version, "to download"))
	}
	return Resolved{
		ID: ID(m.ModID, m.AssetID), Name: m.Name, Version: pick.ModVersion, FileName: pick.FileName, URL: pick.MainFile,
		Picture: m.Logo, ReleaseID: pick.ReleaseID,
	}, nil
}

// Items answers for installed mods by id with each one's newest stable release, one request per mod, so an update
// check compares versions without searching by name.
func (d Driver) Items(ctx context.Context, _, _ string, ids []string) (map[string]source.Item, error) {
	out := map[string]source.Item{}
	for _, id := range ids {
		modID, _, ok := splitID(id)
		if !ok {
			continue
		}
		m, err := d.mod(ctx, modID)
		if err != nil {
			return out, err
		}
		r, ok := m.newest()
		if !ok {
			continue
		}
		out[strings.ToLower(id)] = source.Item{
			Source: d.ID(), ID: ID(m.ModID, m.AssetID), Name: m.Name, Author: m.Author, Version: r.ModVersion,
			Picture: m.Logo, URL: pageURL(m.URLAlias, m.AssetID),
		}
	}
	return out, nil
}

// Changelog reads the changelogs of the mod's releases newer than installed, up to latest, newest first. The site
// lists releases newest first and writes each changelog as HTML.
func (d Driver) Changelog(ctx context.Context, _, id, installed, latest, _ string) ([]source.Note, error) {
	ref := id
	if modID, _, ok := splitID(id); ok {
		ref = modID
	}
	m, err := d.mod(ctx, ref)
	if err != nil {
		return nil, err
	}
	out := []source.Note{}
	started := latest == ""
	for _, r := range m.Releases {
		if r.ModVersion == installed {
			break
		}
		started = started || r.ModVersion == latest
		if !started || len(out) == source.MaxNotes {
			continue
		}
		date, _, _ := strings.Cut(r.Created, " ")
		out = append(out, source.Note{Version: r.ModVersion, Date: date, Body: source.HTMLText(r.Changelog)})
	}
	return out, nil
}

// ReportTarget is the mod's page, where the Mod DB takes comments and links its author.
func (d Driver) ReportTarget(ctx context.Context, _, id, _ string) (source.ReportTarget, error) {
	ref := id
	if modID, _, ok := splitID(id); ok {
		ref = modID
	}
	m, err := d.mod(ctx, ref)
	if err != nil {
		return source.ReportTarget{}, err
	}
	return source.ReportTarget{URL: pageURL(m.URLAlias, m.AssetID)}, nil
}

// Details reads the mod's page: its description (HTML on the site, Markdown here), its tags, every release's version
// and the changelog of the release a player on stable builds gets. The site lists no dependencies apart from the mod
// archive.
func (d Driver) Details(ctx context.Context, _, id, _ string) (source.Details, error) {
	ref := id
	if modID, _, ok := splitID(id); ok {
		ref = modID
	}
	m, err := d.mod(ctx, ref)
	if err != nil {
		return source.Details{}, err
	}
	out := source.Details{
		Description: source.HTMLMarkdown(m.Text), Versions: []string{}, Dependencies: []string{},
		Categories: slices.DeleteFunc(slices.Clone(m.Tags), func(t string) bool { return t == "" }),
	}
	for _, r := range m.Releases {
		out.Versions = append(out.Versions, r.ModVersion)
	}
	if r, ok := m.newest(); ok {
		out.Changelog = source.HTMLMarkdown(r.Changelog)
	}
	return out, nil
}
