// Package vsmoddb is the Vintage Story Mod DB source driver, over the site's public API
// (https://mods.vintagestory.at/api). A mod's id is "<modid>-<assetid>": the API reads a mod by its modid, and its page
// is addressed by its assetid, and neither answers to the other.
package vsmoddb

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-vintagestory/moddb"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

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
func ID(modID, assetID int) string { return moddb.ID(modID, assetID) }

// ModPageURL is the mod's page; gameKey does not enter the address.
func (Driver) ModPageURL(_, id string) string {
	if _, asset, ok := moddb.SplitID(id); ok && asset != "" {
		return moddb.SiteURL + "/show/mod/" + asset
	}
	return ""
}

// client is the API client for this driver's HTTP and URL. Passing http.DefaultClient rather than nil keeps Mortar's
// own transport, whose user agent the app sets.
func (d Driver) client() *moddb.Client {
	return moddb.New(moddb.Config{BaseURL: d.URL, HTTP: cmp.Or(d.HTTP, http.DefaultClient)})
}

// busy turns the API's 429 into the source's busy error, which the download queue waits on. The client gives the wait
// only when the site's Retry-After named one.
func busy(err error) error {
	var rl *httpkit.RateLimitError
	switch {
	case errors.As(err, &rl):
		return &source.BusyError{Source: "the Vintage Story Mod DB", Reset: rl.Reset}
	case errors.Is(err, httpkit.ErrRateLimited):
		return &source.BusyError{Source: "the Vintage Story Mod DB"}
	}
	return err
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
		return moddb.OrderDownloads
	case source.SortEndorsements:
		return moddb.OrderFollows
	case source.SortUpdated:
		return moddb.OrderLastReleased
	case source.SortNewest:
		return moddb.OrderCreated
	}
	return ""
}

// stamp turns the site's "2006-01-02 15:04:05" (UTC) into RFC 3339, as the other sources give it.
func stamp(s string) string {
	t, err := time.Parse(time.DateTime, s)
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// Search lists the site's mods. The API has no offset and returns the whole matching list in one response, so the
// list is fetched once per query and every page is cut from that copy (see listCached).
// Included categories go to the site as tag ids; excluded ones are dropped from the page and counted in Hidden.
// Total is the rows through this page plus one when more follow; the API gives no total.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	page := max(q.Page-source.FirstPage, 0) + 1
	query := moddb.ListQuery{Text: q.Text, OrderBy: orderBy(q.Sort)}
	if len(q.Categories) > 0 {
		ids, err := d.tagIDs(ctx)
		if err != nil {
			return source.Page{}, err
		}
		for _, c := range q.Categories {
			if id, ok := ids[strings.ToLower(c)]; ok {
				query.Tags = append(query.Tags, id)
			}
		}
		// An included category the site has no tag for matches nothing; searching without one would match everything.
		if len(query.Tags) == 0 {
			return source.Page{Items: []source.Item{}}, nil
		}
	}
	mods, err := d.listCached(ctx, query)
	if err != nil {
		return source.Page{}, err
	}
	total := min(len(mods), page*q.Len()+1)
	end := min(page*q.Len(), total)
	start := min((page-1)*q.Len(), end)
	out := source.Page{Total: total, Items: []source.Item{}}
	for _, m := range mods[start:end] {
		if !source.CategoryMatch(m.Tags, nil, q.ExcludeCategories) {
			out.Hidden++
			continue
		}
		out.Items = append(out.Items, source.Item{
			Source: d.ID(), ID: ID(m.ModID, m.AssetID), Name: m.Name, Summary: m.Summary, Author: m.Author,
			Picture: m.Logo, Endorsements: m.Follows, Downloads: m.Downloads, Updated: stamp(m.LastReleased),
			URL: moddb.PageURL(m.URLAlias, m.AssetID),
		})
	}
	return out, nil
}

// now is the clock the caches read; tests replace it.
var now = time.Now

const (
	// modListTTL is how long a fetched mod list serves pages.
	modListTTL = time.Hour
	// tagsTTL is how long the tag list is reused; it matches the other sources' category lists.
	tagsTTL = 24 * time.Hour
)

// lists holds one entry per upstream query; an entry's mutex is held across its fetch so concurrent pages share it.
var lists struct {
	sync.Mutex
	m map[string]*listEntry
}

type listEntry struct {
	sync.Mutex
	mods  []moddb.ListedMod
	until time.Time
}

func (d Driver) listCached(ctx context.Context, q moddb.ListQuery) ([]moddb.ListedMod, error) {
	// The key is the upstream query, so two pages of it share one fetch; tags are ids, so their order is not a
	// difference worth a second fetch.
	tags := slices.Sorted(slices.Values(q.Tags))
	key := strings.TrimRight(cmp.Or(d.URL, moddb.BaseURL), "/") + "?" +
		url.Values{"text": {q.Text}, "orderby": {q.OrderBy}, "tags": tags}.Encode()
	lists.Lock()
	if lists.m == nil {
		lists.m = map[string]*listEntry{}
	}
	e := lists.m[key]
	if e == nil {
		e = &listEntry{}
		lists.m[key] = e
	}
	lists.Unlock()
	e.Lock()
	defer e.Unlock()
	if e.mods != nil && now().Before(e.until) {
		return e.mods, nil
	}
	mods, err := d.client().Mods(ctx, q)
	if err != nil {
		return nil, busy(err)
	}
	e.mods, e.until = mods, now().Add(modListTTL)
	return e.mods, nil
}

// tags caches the site's tag names and their ids by lower-cased name for tagsTTL.
var tags struct {
	sync.Mutex
	ids   map[string]string
	names []string
	until time.Time
}

func (d Driver) tagIDs(ctx context.Context) (map[string]string, error) {
	tags.Lock()
	defer tags.Unlock()
	if tags.ids != nil && now().Before(tags.until) {
		return tags.ids, nil
	}
	ids, err := d.client().Tags(ctx)
	if err != nil {
		return nil, busy(err)
	}
	names := slices.Sorted(maps.Keys(ids))
	tags.ids, tags.names, tags.until = ids, names, now().Add(tagsTTL)
	return ids, nil
}

// Categories lists the site's tag names.
func (d Driver) Categories(ctx context.Context, _ string) ([]string, error) {
	if _, err := d.tagIDs(ctx); err != nil {
		return nil, err
	}
	tags.Lock()
	defer tags.Unlock()
	return slices.Clone(tags.names), nil
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
	if modID, _, ok := moddb.SplitID(id); ok {
		ref = modID
	}
	m, err := d.client().Mod(ctx, ref)
	if err != nil {
		return Resolved{}, busy(err)
	}
	pick, ok := m.Newest(false)
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
		modID, _, ok := moddb.SplitID(id)
		if !ok {
			continue
		}
		m, err := d.client().Mod(ctx, modID)
		if err != nil {
			return out, busy(err)
		}
		r, ok := m.Newest(false)
		if !ok {
			continue
		}
		out[strings.ToLower(id)] = source.Item{
			Source: d.ID(), ID: ID(m.ModID, m.AssetID), Name: m.Name, Author: m.Author, Version: r.ModVersion,
			Picture: m.Logo, URL: moddb.PageURL(m.URLAlias, m.AssetID),
		}
	}
	return out, nil
}

// Changelog reads the changelogs of the mod's releases newer than installed, up to latest, newest first. The site
// lists releases newest first and writes each changelog as HTML.
func (d Driver) Changelog(ctx context.Context, _, id, installed, latest, _ string) ([]source.Note, error) {
	ref := id
	if modID, _, ok := moddb.SplitID(id); ok {
		ref = modID
	}
	m, err := d.client().Mod(ctx, ref)
	if err != nil {
		return nil, busy(err)
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
	if modID, _, ok := moddb.SplitID(id); ok {
		ref = modID
	}
	m, err := d.client().Mod(ctx, ref)
	if err != nil {
		return source.ReportTarget{}, busy(err)
	}
	return source.ReportTarget{URL: moddb.PageURL(m.URLAlias, m.AssetID)}, nil
}

// Details reads the mod's page: its description (HTML on the site, Markdown here), its tags, every release's version
// and the changelog of the release a player on stable builds gets. The site lists no dependencies apart from the mod
// archive.
func (d Driver) Details(ctx context.Context, _, id, _ string) (source.Details, error) {
	ref := id
	if modID, _, ok := moddb.SplitID(id); ok {
		ref = modID
	}
	m, err := d.client().Mod(ctx, ref)
	if err != nil {
		return source.Details{}, busy(err)
	}
	out := source.Details{
		Description: source.HTMLMarkdown(m.Text), Versions: []string{}, Dependencies: []string{},
		Categories: slices.DeleteFunc(slices.Clone(m.Tags), func(t string) bool { return t == "" }),
	}
	for _, r := range m.Releases {
		out.Versions = append(out.Versions, r.ModVersion)
		out.Released = append(out.Released, source.Released(stamp(r.Created)))
	}
	if r, ok := m.Newest(false); ok {
		out.Changelog = source.HTMLMarkdown(r.Changelog)
	}
	return out, nil
}
