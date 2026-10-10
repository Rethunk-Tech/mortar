// Package curseforge is the CurseForge source driver, over the Core API v1 with Mortar's own API key. A game opts in
// with a source entry whose gameId is the site's game id and whose key is the id of its "Mods" class. The key is set
// at release build time (apiKey) or, for dev and test builds, by MORTAR_CURSEFORGE_KEY; without either the driver
// reports itself unavailable and browse disables it. Authors can forbid distribution outside CurseForge: a file with
// no download address is never fetched around the API, and the queue hands the user the mod's page instead.
package curseforge

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// BaseURL is the real API.
const BaseURL = "https://api.curseforge.com/v1"

const siteURL = "https://www.curseforge.com"

// apiKey is set at release build time with -ldflags -X.
var apiKey string

// envKey names the variable a dev or test build reads the key from.
const envKey = "MORTAR_CURSEFORGE_KEY"

// ErrNoKey means this build carries no CurseForge API key.
var ErrNoKey = errors.New("this build has no CurseForge API key")

// NotDistributableError is a file whose author does not allow it to be downloaded outside CurseForge. PageURL is the
// mod's page, where the user downloads it.
type NotDistributableError struct{ Mod, PageURL string }

func (e *NotDistributableError) Error() string {
	return e.Mod + "'s author only allows downloads from CurseForge"
}

// Driver searches CurseForge. HTTP, URL, Key and GameSource are for tests; the zero value talks to the real API
// with the build's key and reads the game's entry from the catalog.
type Driver struct {
	HTTP *http.Client
	URL  string
	// Key returns the API key; nil reads the build's key.
	Key func() string
	// GameSource returns the game's CurseForge entry (game "" asks for any); nil reads the catalog.
	GameSource func(game string) (components.GameSource, bool)
}

var (
	_                    = source.Register(Driver{})
	_ source.Categorizer = Driver{}
)

// ID is the catalog id.
func (Driver) ID() string { return "curseforge" }

// Name is the site's name.
func (Driver) Name() string { return "CurseForge" }

// Modes says Mortar downloads files itself, and hands over to the site's page for a mod that forbids it.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Download, source.Handoff} }

// Hosts is the site's web host.
func (Driver) Hosts() []string { return []string{"curseforge.com"} }

// ModPageURL is the project's page; the site redirects the numeric id to the slugged address.
func (Driver) ModPageURL(_, id string) string {
	if id == "" {
		return ""
	}
	return siteURL + "/projects/" + url.PathEscape(id)
}

func (d Driver) key() string {
	if d.Key != nil {
		return strings.TrimSpace(d.Key())
	}
	if apiKey != "" {
		return strings.TrimSpace(apiKey)
	}
	return strings.TrimSpace(os.Getenv(envKey))
}

// Unavailable says why browse cannot use the source now, or "" when it can.
func (d Driver) Unavailable() string {
	if d.key() == "" {
		return ErrNoKey.Error()
	}
	return ""
}

func (d Driver) gameSource(game string) (components.GameSource, bool) {
	if d.GameSource != nil {
		return d.GameSource(game)
	}
	g, ok := components.Game(game)
	if !ok {
		return components.GameSource{}, false
	}
	return g.Source("curseforge")
}

// do calls the API. The key rides in a header only, so an error that quotes the address cannot leak it.
func (d Driver) do(ctx context.Context, method, path string, params url.Values, body, out any) error {
	key := d.key()
	if key == "" {
		return ErrNoKey
	}
	u := strings.TrimRight(cmp.Or(d.URL, BaseURL), "/") + path
	return source.DoJSON(ctx, source.Request{
		Service: "CurseForge", Client: d.HTTP, Method: method, URL: u, Params: params, Body: body,
		UserAgent: source.UserAgent(""),
		Header:    func(h http.Header) { h.Set("X-Api-Key", key) },
	}, out)
}

func (d Driver) get(ctx context.Context, path string, params url.Values, out any) error {
	return d.do(ctx, http.MethodGet, path, params, nil, out)
}

// CurseForge's sortField values (https://docs.curseforge.com/rest-api/#tocS_ModsSearchSortField).
const (
	sortFeatured  = 1
	sortPopular   = 2
	sortUpdated   = 3
	sortName      = 4
	sortDownloads = 6
	sortReleased  = 11
)

// Sorts are the ModsSearchSortField values the driver offers besides Featured, its best match.
func (Driver) Sorts() []string {
	return []string{source.SortEndorsements, source.SortDownloads, source.SortUpdated, source.SortNewest, source.SortName}
}

func sortField(sort string) int {
	switch sort {
	case source.SortDownloads:
		return sortDownloads
	case source.SortUpdated:
		return sortUpdated
	case source.SortEndorsements:
		return sortPopular
	case source.SortNewest:
		return sortReleased
	case source.SortName:
		return sortName
	}
	return sortFeatured
}

// maxWindow is how deep the API pages: index + pageSize may not pass it.
const maxWindow = 10000

type modResp struct {
	ID                   int    `json:"id"`
	Name                 string `json:"name"`
	Summary              string `json:"summary"`
	DownloadCount        int    `json:"downloadCount"`
	ThumbsUpCount        int    `json:"thumbsUpCount"`
	DateModified         string `json:"dateModified"`
	DateReleased         string `json:"dateReleased"`
	ClassID              int    `json:"classId"`
	MainFileID           int    `json:"mainFileId"`
	AllowModDistribution *bool  `json:"allowModDistribution"`
	IsAvailable          bool   `json:"isAvailable"`
	Links                struct {
		WebsiteURL string `json:"websiteUrl"`
		SourceURL  string `json:"sourceUrl"`
	} `json:"links"`
	Authors []struct {
		Name string `json:"name"`
	} `json:"authors"`
	Logo struct {
		ThumbnailURL string `json:"thumbnailUrl"`
	} `json:"logo"`
	Categories []struct {
		Name string `json:"name"`
	} `json:"categories"`
	LatestFiles []fileResp `json:"latestFiles"`
}

type fileResp struct {
	ID          int    `json:"id"`
	ModID       int    `json:"modId"`
	IsAvailable bool   `json:"isAvailable"`
	DisplayName string `json:"displayName"`
	FileName    string `json:"fileName"`
	// ReleaseType is 1 release, 2 beta, 3 alpha.
	ReleaseType int `json:"releaseType"`
	Hashes      []struct {
		Value string `json:"value"`
		// Algo is 1 for SHA-1 and 2 for MD5.
		Algo int `json:"algo"`
	} `json:"hashes"`
	FileDate     string   `json:"fileDate"`
	FileLength   int64    `json:"fileLength"`
	DownloadURL  string   `json:"downloadUrl"`
	GameVersions []string `json:"gameVersions"`
	Dependencies []struct {
		ModID int `json:"modId"`
		// RelationType 3 is a required dependency.
		RelationType int `json:"relationType"`
	} `json:"dependencies"`
}

const (
	relationRequired = 3
	releaseStable    = 1
	hashSHA1         = 1
)

func (f fileResp) sha1() string {
	for _, h := range f.Hashes {
		if h.Algo == hashSHA1 && h.Value != "" {
			return strings.ToLower(h.Value)
		}
	}
	return ""
}

// version is the name a file goes by in Mortar: its display name without the archive extension.
func (f fileResp) version() string {
	name := cmp.Or(f.DisplayName, f.FileName, strconv.Itoa(f.ID))
	return archive.StripExtension(name)
}

func (f fileResp) required() []string {
	var ids []string
	for _, dep := range f.Dependencies {
		if dep.RelationType == relationRequired && dep.ModID > 0 {
			ids = append(ids, strconv.Itoa(dep.ModID))
		}
	}
	return ids
}

// newest picks the newest published file, preferring stable releases, and keeping to files for one of gameVersions
// when it names any.
func newest(files []fileResp, gameVersions []string) (fileResp, bool) {
	var best fileResp
	found := false
	for _, f := range files {
		if !f.IsAvailable || f.ID == 0 {
			continue
		}
		if len(gameVersions) > 0 && !slices.ContainsFunc(f.GameVersions, func(v string) bool { return slices.Contains(gameVersions, v) }) {
			continue
		}
		stable, bestStable := f.ReleaseType == releaseStable, best.ReleaseType == releaseStable
		if !found || (stable && !bestStable) || (stable == bestStable && f.FileDate > best.FileDate) {
			best, found = f, true
		}
	}
	return best, found
}

// apiPageSize is the most mods one /mods/search request returns.
const apiPageSize = 50

// Search lists mods in the game's Mods class, or in every class the catalog lists for it, merged by the query's sort.
// Included categories go to the API as ids; it has no exclude parameter and ANDs several ids, so excluded ones are
// dropped from the page afterwards (the total still counts them).
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	gs, ok := d.gameSource(q.Game)
	if !ok || gs.GameID == 0 || q.Key == "" {
		return source.Page{}, fmt.Errorf("game %q has no CurseForge game and class id", q.Game)
	}
	index := max(q.Page-source.FirstPage, 0) * source.PageSize
	if index+source.PageSize > maxWindow {
		return source.Page{Total: maxWindow}, nil
	}
	classes := gs.Classes
	if len(classes) == 0 {
		classes = []string{q.Key}
	}
	var mods []modResp
	var failed []string
	var total int
	if len(classes) > 1 && index+source.PageSize > mergedCap {
		return source.Page{Total: mergedCap}, nil
	}
	if len(classes) == 1 {
		var err error
		if mods, total, err = d.searchClass(ctx, gs.GameID, classes[0], q, index, source.PageSize); err != nil {
			return source.Page{}, err
		}
	} else {
		var err error
		if mods, total, failed, err = d.searchClasses(ctx, gs.GameID, classes, q, index); err != nil {
			return source.Page{}, err
		}
	}
	items := make([]source.Item, 0, len(mods))
	for _, m := range mods {
		if !source.CategoryMatch(categoryNames(m), nil, q.ExcludeCategories) {
			continue
		}
		it := source.Item{
			Source: d.ID(), ID: strconv.Itoa(m.ID), Name: m.Name, Summary: m.Summary, Picture: m.Logo.ThumbnailURL,
			Endorsements: m.ThumbsUpCount, Downloads: m.DownloadCount, Updated: m.DateModified,
			URL: cmp.Or(m.Links.WebsiteURL, d.ModPageURL(strconv.Itoa(m.ClassID), strconv.Itoa(m.ID))), Repo: source.GitHubRepo(m.Links.SourceURL),
			External: m.AllowModDistribution != nil && !*m.AllowModDistribution,
		}
		if len(m.Authors) > 0 {
			it.Author = m.Authors[0].Name
		}
		for _, f := range m.LatestFiles {
			if f.ID == m.MainFileID {
				it.Version = f.version()
			}
		}
		items = append(items, it)
	}
	failed = d.classLabels(ctx, gs.GameID, failed)
	limit := maxWindow
	if len(classes) > 1 {
		limit = mergedCap
	}
	return source.Page{Total: min(total, limit), Items: items, Failed: failed}, nil
}

// classNames caches each game's class names (Mods, Create a Sim) for the process, keyed by API address and game.
var classNames sync.Map

// classLabels names the classes of a merged search that did not answer for the person reading the page: the site, and
// the class's own name where the API gives it. A name that cannot be read leaves the site alone.
func (d Driver) classLabels(ctx context.Context, gameID int, classes []string) []string {
	if len(classes) == 0 {
		return nil
	}
	key := cmp.Or(d.URL, BaseURL) + "|" + strconv.Itoa(gameID)
	var names map[string]string
	if v, ok := classNames.Load(key); ok {
		names, _ = v.(map[string]string)
	} else {
		var out struct {
			Data []category `json:"data"`
		}
		if err := d.get(ctx, "/categories", url.Values{"gameId": {strconv.Itoa(gameID)}, "classesOnly": {"true"}}, &out); err == nil {
			names = map[string]string{}
			for _, c := range out.Data {
				names[strconv.Itoa(c.ID)] = c.Name
			}
			classNames.Store(key, names)
		}
	}
	var labels []string
	for _, class := range classes {
		label := "CurseForge"
		if n := names[class]; n != "" {
			label += " (" + n + ")"
		}
		if !slices.Contains(labels, label) {
			labels = append(labels, label)
		}
	}
	return labels
}

// searchClass asks one class for size mods from index, and returns them with the class's total.
func (d Driver) searchClass(ctx context.Context, gameID int, class string, q source.Query, index, size int) ([]modResp, int, error) {
	order := "desc"
	if q.Sort == source.SortName {
		order = "asc"
	}
	params := url.Values{}
	params.Set("gameId", strconv.Itoa(gameID))
	params.Set("classId", class)
	params.Set("searchFilter", q.Text)
	params.Set("sortField", strconv.Itoa(sortField(q.Sort)))
	params.Set("sortOrder", order)
	params.Set("index", strconv.Itoa(index))
	params.Set("pageSize", strconv.Itoa(size))
	if len(q.Categories) > 0 {
		cats, err := d.classCategories(ctx, gameID, class)
		if err != nil {
			return nil, 0, err
		}
		var ids []int
		for _, c := range cats {
			if slices.ContainsFunc(q.Categories, func(n string) bool { return strings.EqualFold(n, c.Name) }) {
				ids = append(ids, c.ID)
			}
		}
		if len(ids) == 0 {
			return nil, 0, nil
		}
		raw, err := json.Marshal(ids)
		if err != nil {
			return nil, 0, err
		}
		params.Set("categoryIds", string(raw))
	}
	var parsed struct {
		Data       []modResp `json:"data"`
		Pagination struct {
			TotalCount int `json:"totalCount"`
		} `json:"pagination"`
	}
	if err := d.get(ctx, "/mods/search", params, &parsed); err != nil {
		return nil, 0, err
	}
	for i := range parsed.Data {
		if parsed.Data[i].ClassID == 0 {
			parsed.Data[i].ClassID, _ = strconv.Atoi(class)
		}
	}
	return parsed.Data, parsed.Pagination.TotalCount, nil
}

// mergedCap is how deep a merged multi-class search goes: past it a page is empty and the total stops here. A merged
// page needs every class's leading rows, so depth is what costs requests.
const mergedCap = 1000

// rowsTTL is how long a class's fetched rows serve later pages of the same search.
const rowsTTL = 10 * time.Minute

// classRows are the leading rows of one class's search. at is when the first of them was fetched, which is their age
// however often they are read; last is when rows were last added, which orders eviction.
type classRows struct {
	mods  []modResp
	total int
	at    time.Time
	last  time.Time
}

// rowsCap is the most class searches the cache holds. One merged search holds one entry per class (five at most in the
// catalog), so 64 keeps the open search and several earlier ones a player steps back to; typing or changing the
// filters adds entries, and the least recently fetched go first.
const rowsCap = 64

var (
	rowsMu    sync.Mutex
	rowsCache = map[string]classRows{}
)

// storeRows keeps held under key, dropping entries past their age and then the least recently fetched past the cap.
func storeRows(key string, held classRows) {
	rowsMu.Lock()
	defer rowsMu.Unlock()
	rowsCache[key] = held
	for k, v := range rowsCache {
		if time.Since(v.at) > rowsTTL {
			delete(rowsCache, k)
		}
	}
	for len(rowsCache) > rowsCap {
		oldest := ""
		for k, v := range rowsCache {
			if oldest == "" || v.last.Before(rowsCache[oldest].last) {
				oldest = k
			}
		}
		delete(rowsCache, oldest)
	}
}

// classRowsTo returns the first `end` rows of one class for the query, reading from the API only the rows the cache
// does not hold yet, in API-sized requests. Rows past their age are read again from the start of the class. The total
// is the class's own.
func (d Driver) classRowsTo(ctx context.Context, gameID int, class string, q source.Query, end int) ([]modResp, int, error) {
	key := strings.Join([]string{cmp.Or(d.URL, BaseURL), strconv.Itoa(gameID), class, q.Text, q.Sort, strings.Join(q.Categories, ",")}, "|")
	rowsMu.Lock()
	held, ok := rowsCache[key]
	if ok && time.Since(held.at) > rowsTTL {
		delete(rowsCache, key)
		held = classRows{}
	}
	rowsMu.Unlock()
	fetched := false
	for len(held.mods) < end && (len(held.mods) == 0 || len(held.mods) < held.total) {
		rows, n, err := d.searchClass(ctx, gameID, class, q, len(held.mods), min(apiPageSize, end-len(held.mods)))
		if err != nil {
			return nil, 0, err
		}
		if len(held.mods) == 0 {
			held.at = time.Now()
		}
		held.mods, held.total, fetched = slices.Concat(held.mods, rows), n, true
		if len(rows) == 0 {
			break
		}
	}
	if fetched {
		held.last = time.Now()
		storeRows(key, held)
	}
	return held.mods[:min(end, len(held.mods))], held.total, nil
}

// searchClasses merges the classes' results by the query's sort and returns the page at index, never deeper than
// mergedCap. Each class is read from its start to the end of the page (rows already read are reused), because a merged
// page needs every class's leading rows; ties keep each class's own order, interleaved by rank then class.
func (d Driver) searchClasses(ctx context.Context, gameID int, classes []string, q source.Query, index int) (mods []modResp, total int, failed []string, err error) {
	end := index + source.PageSize
	type ranked struct {
		mod         modResp
		rank, class int
	}
	var all []ranked
	var errs []error
	for ci, class := range classes {
		got, n, err := d.classRowsTo(ctx, gameID, class, q, end)
		if err != nil {
			errs = append(errs, fmt.Errorf("class %s: %w", class, err))
			failed = append(failed, class)
			continue
		}
		total += n
		for r, m := range got {
			all = append(all, ranked{m, r, ci})
		}
	}
	slices.SortStableFunc(all, func(a, b ranked) int {
		if c := compareBySort(q.Sort, a.mod, b.mod); c != 0 {
			return c
		}
		return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(a.class, b.class))
	})
	if len(failed) == len(classes) {
		return nil, 0, nil, errors.Join(errs...)
	}
	if len(failed) > 0 {
		log.Printf("curseforge: search of %d classes: %d answered with %d mods, %d failed (%s): %v",
			len(classes), len(classes)-len(failed), len(all), len(failed), strings.Join(failed, ","), errors.Join(errs...))
	}
	if index >= len(all) {
		return nil, total, failed, nil
	}
	out := make([]modResp, 0, min(end, len(all))-index)
	for _, r := range all[index:min(end, len(all))] {
		out = append(out, r.mod)
	}
	return out, total, failed, nil
}

// compareBySort orders two mods as the API's sortField does: negative when a comes first.
func compareBySort(sort string, a, b modResp) int {
	switch sort {
	case source.SortDownloads:
		return cmp.Compare(b.DownloadCount, a.DownloadCount)
	case source.SortEndorsements:
		return cmp.Compare(b.ThumbsUpCount, a.ThumbsUpCount)
	case source.SortUpdated:
		return cmp.Compare(b.DateModified, a.DateModified)
	case source.SortNewest:
		return cmp.Compare(b.DateReleased, a.DateReleased)
	case source.SortName:
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	}
	return 0
}

func categoryNames(m modResp) []string {
	names := make([]string, len(m.Categories))
	for i, c := range m.Categories {
		names[i] = c.Name
	}
	return names
}

type category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// categoryLists caches each class's categories for the process, keyed by API address, game and class.
var categoryLists sync.Map

func (d Driver) classCategories(ctx context.Context, gameID int, classID string) ([]category, error) {
	key := cmp.Or(d.URL, BaseURL) + "|" + strconv.Itoa(gameID) + "|" + classID
	if v, ok := categoryLists.Load(key); ok {
		if cats, ok := v.([]category); ok {
			return cats, nil
		}
	}
	var out struct {
		Data []category `json:"data"`
	}
	params := url.Values{"gameId": {strconv.Itoa(gameID)}, "classId": {classID}}
	if err := d.get(ctx, "/categories", params, &out); err != nil {
		return nil, err
	}
	categoryLists.Store(key, out.Data)
	return out.Data, nil
}

// Categories lists the category names of the Mods class key, sorted. The catalog's game for a class is found by
// scanning the games' CurseForge entries, since the API wants the game id too.
func (d Driver) Categories(ctx context.Context, key string) ([]string, error) {
	var sources []components.GameSource
	if d.GameSource != nil {
		if gs, ok := d.GameSource(""); ok {
			sources = append(sources, gs)
		}
	} else {
		for _, g := range components.Games() {
			if gs, ok := g.Source("curseforge"); ok {
				sources = append(sources, gs)
			}
		}
	}
	var names []string
	for _, gs := range sources {
		if gs.Key != key || gs.GameID == 0 {
			continue
		}
		classes := gs.Classes
		if len(classes) == 0 {
			classes = []string{key}
		}
		for _, class := range classes {
			cats, err := d.classCategories(ctx, gs.GameID, class)
			if err != nil {
				return nil, err
			}
			for _, c := range cats {
				names = append(names, c.Name)
			}
		}
	}
	return source.UniqueNames(names), nil
}

func (d Driver) mod(ctx context.Context, id string) (modResp, error) {
	var out struct {
		Data modResp `json:"data"`
	}
	err := d.get(ctx, "/mods/"+url.PathEscape(id), nil, &out)
	return out.Data, err
}

// files lists the mod's newest published files, at most one API page.
func (d Driver) files(ctx context.Context, id string) ([]fileResp, error) {
	var out struct {
		Data []fileResp `json:"data"`
	}
	err := d.get(ctx, "/mods/"+url.PathEscape(id)+"/files", url.Values{"pageSize": {"50"}}, &out)
	return out.Data, err
}

type projectName struct {
	name string
	at   time.Time
}

var (
	namesMu sync.Mutex
	names   = map[string]projectName{}
)

// ProjectName is the name of project id, kept for ten minutes so a preview of many shared files asks once per project.
func (d Driver) ProjectName(ctx context.Context, id string) (string, error) {
	key := cmp.Or(d.URL, BaseURL) + "|" + id
	namesMu.Lock()
	held, ok := names[key]
	namesMu.Unlock()
	if ok && time.Since(held.at) <= rowsTTL {
		return held.name, nil
	}
	m, err := d.mod(ctx, id)
	if err != nil {
		return "", err
	}
	namesMu.Lock()
	names[key] = projectName{name: m.Name, at: time.Now()}
	namesMu.Unlock()
	return m.Name, nil
}

// Details reads the mod's summary, its file names and the required mods of its newest file. The site's long
// description is HTML, which Details does not carry, so the summary stands in for it.
func (d Driver) Details(ctx context.Context, _, id, _ string) (source.Details, error) {
	m, err := d.mod(ctx, id)
	if err != nil {
		return source.Details{}, err
	}
	files, err := d.files(ctx, id)
	if err != nil {
		return source.Details{}, err
	}
	slices.SortStableFunc(files, func(a, b fileResp) int { return strings.Compare(b.FileDate, a.FileDate) })
	out := source.Details{Description: m.Summary, Versions: []string{}, Dependencies: []string{}, Categories: []string{}}
	for _, f := range files {
		if f.IsAvailable {
			out.Versions = append(out.Versions, f.version())
		}
	}
	if f, ok := newest(files, nil); ok {
		out.Dependencies = append(out.Dependencies, f.required()...)
	}
	for _, c := range m.Categories {
		out.Categories = append(out.Categories, c.Name)
	}
	return out, nil
}

// Resolved is what an install needs of one file.
type Resolved struct {
	ModID string
	// Name is the mod's name.
	Name string
	// Version is the file's display name, and VersionID its numeric id, which names it exactly.
	Version   string
	VersionID string
	FileName  string
	URL       string
	Size      int64
	// Digest is "sha1:<hex>", the only strong-enough hash the site publishes besides MD5. It names the file for update
	// checks; Mortar does not verify it, since the linter bans the weak hash.
	Digest string
	// Dependencies are the mods the file requires, by id; each resolves to its newest file.
	Dependencies []string
}

// Resolve finds the mod's file. version is a file id or a display name; empty means the newest. A file the author
// does not allow outside CurseForge fails as *NotDistributableError.
func (d Driver) Resolve(ctx context.Context, id, version string) (Resolved, error) {
	m, err := d.mod(ctx, id)
	if err != nil {
		return Resolved{}, err
	}
	f, err := d.pick(ctx, id, version)
	if err != nil {
		return Resolved{}, err
	}
	page := cmp.Or(m.Links.WebsiteURL, d.ModPageURL("", id))
	if m.AllowModDistribution != nil && !*m.AllowModDistribution {
		return Resolved{}, &NotDistributableError{Mod: m.Name, PageURL: page}
	}
	link := f.DownloadURL
	if link == "" {
		link, err = d.downloadURL(ctx, id, f.ID)
		if err != nil {
			if se, ok := errors.AsType[*source.StatusError](err); ok && se.Auth() {
				return Resolved{}, &NotDistributableError{Mod: m.Name, PageURL: page}
			}
			return Resolved{}, err
		}
	}
	if link == "" {
		return Resolved{}, &NotDistributableError{Mod: m.Name, PageURL: page}
	}
	res := Resolved{
		ModID: id, Name: m.Name, Version: f.version(), VersionID: strconv.Itoa(f.ID), FileName: f.FileName,
		URL: link, Size: f.FileLength, Dependencies: f.required(),
	}
	if h := f.sha1(); h != "" {
		res.Digest = "sha1:" + h
	}
	return res, nil
}

func (d Driver) pick(ctx context.Context, id, version string) (fileResp, error) {
	if n, err := strconv.Atoi(version); err == nil && n > 0 {
		var out struct {
			Data fileResp `json:"data"`
		}
		if err := d.get(ctx, "/mods/"+url.PathEscape(id)+"/files/"+strconv.Itoa(n), nil, &out); err != nil {
			return fileResp{}, err
		}
		return out.Data, nil
	}
	files, err := d.files(ctx, id)
	if err != nil {
		return fileResp{}, err
	}
	if version != "" {
		files = slices.DeleteFunc(files, func(f fileResp) bool { return f.version() != version && f.DisplayName != version })
	}
	if f, ok := newest(files, nil); ok {
		return f, nil
	}
	return fileResp{}, fmt.Errorf("%s %s is not on CurseForge", id, version)
}

// downloadURL asks the API for the file's address; a refusal means the author forbids third-party downloads.
func (d Driver) downloadURL(ctx context.Context, id string, file int) (string, error) {
	var out struct {
		Data string `json:"data"`
	}
	err := d.get(ctx, "/mods/"+url.PathEscape(id)+"/files/"+strconv.Itoa(file)+"/download-url", nil, &out)
	return out.Data, err
}

// listed holds the required dependencies of every file an update check read, keyed by mod id and version, so
// Dependencies answers without another request.
var listed sync.Map

func remember(f fileResp) {
	listed.Store(source.VersionRef{ID: strconv.Itoa(f.ModID), Version: f.version()}, f.required())
}

// Latest reads each installed mod's newest file through POST /mods, which carries the latest files and their
// dependencies for a batch of mods. A mod whose newest file is the installed one, by SHA-1 or by name, is absent. The
// choice of newest is stable releases only, so an installed beta can be offered an older release.
func (d Driver) Latest(ctx context.Context, src components.GameSource, _ string, files []source.InstalledFile) (map[string]source.Latest, error) {
	out := map[string]source.Latest{}
	var ids []int
	byMod := map[int][]source.InstalledFile{}
	for _, f := range files {
		id, err := strconv.Atoi(f.ID)
		if err != nil || id <= 0 {
			continue
		}
		if _, seen := byMod[id]; !seen {
			ids = append(ids, id)
		}
		byMod[id] = append(byMod[id], f)
	}
	slices.Sort(ids)
	const batch = 100
	for chunk := range slices.Chunk(ids, batch) {
		var resp struct {
			Data []modResp `json:"data"`
		}
		if err := d.do(ctx, http.MethodPost, "/mods", nil, struct {
			ModIDs []int `json:"modIds"`
		}{chunk}, &resp); err != nil {
			return nil, err
		}
		for _, m := range resp.Data {
			f, ok := newest(m.LatestFiles, src.GameVersions)
			if !ok {
				continue
			}
			remember(f)
			for _, have := range byMod[m.ID] {
				if have.Digest == "sha1:"+f.sha1() || have.Version == f.version() {
					continue
				}
				out[have.Digest] = source.Latest{
					ProjectID: strconv.Itoa(m.ID), Version: f.version(), VersionID: strconv.Itoa(f.ID),
					URL: cmp.Or(m.Links.WebsiteURL, Driver{}.ModPageURL("", strconv.Itoa(m.ID))),
				}
			}
		}
	}
	return out, nil
}

// Dependencies answers from the files update checks read: each known file's required mods, by id.
func (Driver) Dependencies(_ context.Context, _, _ string, refs []source.VersionRef) (map[source.VersionRef][]string, error) {
	out := map[source.VersionRef][]string{}
	for _, r := range refs {
		if ids, ok := listed.Load(r); ok {
			out[r], _ = ids.([]string)
		}
	}
	return out, nil
}
