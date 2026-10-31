// Package curseforge is the CurseForge source driver, over the Core API v1 with Mortar's own API key. A game opts in
// with a source entry whose gameId is the site's game id and whose key is the id of its "Mods" class. The key is set
// at release build time (apiKey) or, for dev and test builds, by MORTAR_CURSEFORGE_KEY; without either the driver
// reports itself unavailable and browse disables it. Authors can forbid distribution outside CurseForge: a file with
// no download address is never fetched around the API, and the queue hands the user the mod's page instead.
package curseforge

import (
	"cmp"
	"context"
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

	cf "github.com/Rethunk-Tech/go-curseforge"
	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

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

// call runs one request through a client built for this call, so a key set while Mortar runs is used at once. The
// library's rate-limit and status errors become source's, which usererr and the queue classify.
func call[T any](d Driver, f func(*cf.Client) (T, error)) (T, error) {
	key := d.key()
	if key == "" {
		var zero T
		return zero, ErrNoKey
	}
	v, err := f(cf.New(cf.Config{APIKey: key, HTTP: d.HTTP, BaseURL: d.URL, UserAgent: source.UserAgentWithSite("")}))
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return v, &source.BusyError{Source: "CurseForge", Reset: rl.Reset}
	}
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return v, &source.StatusError{Service: "CurseForge", Code: se.Code, Status: se.Status}
	}
	return v, err
}

// projectID is a project id as the API takes it.
func projectID(id string) (int, error) {
	n, err := strconv.Atoi(id)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%q is not a CurseForge project id", id)
	}
	return n, nil
}

// Sorts are the ModsSearchSortField values the driver offers besides Featured, its best match.
func (Driver) Sorts() []string {
	return []string{source.SortEndorsements, source.SortDownloads, source.SortUpdated, source.SortNewest, source.SortName}
}

func sortField(sort string) int {
	switch sort {
	case source.SortDownloads:
		return cf.SortTotalDownloads
	case source.SortUpdated:
		return cf.SortLastUpdated
	case source.SortEndorsements:
		return cf.SortPopularity
	case source.SortNewest:
		return cf.SortReleasedDate
	case source.SortName:
		return cf.SortName
	}
	return cf.SortFeatured
}

// fileVersion is the name a file goes by in Mortar: its display name without the archive extension.
func fileVersion(f cf.File) string {
	return archive.StripExtension(cmp.Or(f.DisplayName, f.FileName, strconv.Itoa(f.ID)))
}

func required(f cf.File) []string {
	var ids []string
	for _, id := range f.Required() {
		ids = append(ids, strconv.Itoa(id))
	}
	return ids
}

// Search lists mods in the game's Mods class, or in every class the catalog lists for it, merged by the query's sort.
// Included categories go to the API as ids; it has no exclude parameter and ANDs several ids, so excluded ones are
// dropped from the page afterwards and counted in Hidden (the total still counts them).
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	gs, ok := d.gameSource(q.Game)
	if !ok || gs.GameID == 0 || q.Key == "" {
		return source.Page{}, fmt.Errorf("game %q has no CurseForge game and class id", q.Game)
	}
	index := max(q.Page-source.FirstPage, 0) * source.PageSize
	if index+source.PageSize > cf.MaxWindow {
		return source.Page{Total: cf.MaxWindow}, nil
	}
	classes := gs.Classes
	if len(classes) == 0 {
		classes = []string{q.Key}
	}
	var mods []cf.Mod
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
	hidden := 0
	for _, m := range mods {
		if !source.CategoryMatch(m.CategoryNames(), nil, q.ExcludeCategories) {
			hidden++
			continue
		}
		it := source.Item{
			Source: d.ID(), ID: strconv.Itoa(m.ID), Name: m.Name, Summary: m.Summary, Picture: m.Logo.ThumbnailURL,
			Endorsements: m.ThumbsUpCount, Downloads: m.DownloadCount, Updated: m.DateModified,
			URL: cmp.Or(m.Links.WebsiteURL, d.ModPageURL(strconv.Itoa(m.ClassID), strconv.Itoa(m.ID))), Repo: source.GitHubRepo(m.Links.SourceURL),
			External: m.Restricted(),
		}
		if len(m.Authors) > 0 {
			it.Author = m.Authors[0].Name
		}
		for _, f := range m.LatestFiles {
			if f.ID == m.MainFileID {
				it.Version = fileVersion(f)
				it.Size = f.FileLength
			}
		}
		items = append(items, it)
	}
	failed = d.classLabels(ctx, gs.GameID, failed)
	limit := cf.MaxWindow
	if len(classes) > 1 {
		limit = mergedCap
	}
	return source.Page{Total: min(total, limit), Items: items, Failed: failed, Hidden: hidden}, nil
}

// classNames caches each game's class names (Mods, Create a Sim) for the process, keyed by API address and game.
var classNames sync.Map

// classLabels names the classes of a merged search that did not answer for the person reading the page: the site, and
// the class's own name where the API gives it. A name that cannot be read leaves the site alone.
func (d Driver) classLabels(ctx context.Context, gameID int, classes []string) []string {
	if len(classes) == 0 {
		return nil
	}
	key := cmp.Or(d.URL, cf.BaseURL) + "|" + strconv.Itoa(gameID)
	var names map[string]string
	if v, ok := classNames.Load(key); ok {
		names, _ = v.(map[string]string)
	} else {
		classes, err := call(d, func(c *cf.Client) ([]cf.Category, error) { return c.Classes(ctx, gameID) })
		if err == nil {
			names = map[string]string{}
			for _, c := range classes {
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
func (d Driver) searchClass(ctx context.Context, gameID int, class string, q source.Query, index, size int) ([]cf.Mod, int, error) {
	classID, err := strconv.Atoi(class)
	if err != nil {
		return nil, 0, fmt.Errorf("CurseForge class %q is not a number", class)
	}
	sq := cf.SearchQuery{
		GameID: gameID, ClassID: classID, Text: q.Text, SortField: sortField(q.Sort), Ascending: q.Sort == source.SortName,
		Index: index, PageSize: size,
	}
	if len(q.Categories) > 0 {
		cats, err := d.classCategories(ctx, gameID, class)
		if err != nil {
			return nil, 0, err
		}
		for _, c := range cats {
			if slices.ContainsFunc(q.Categories, func(n string) bool { return strings.EqualFold(n, c.Name) }) {
				sq.CategoryIDs = append(sq.CategoryIDs, c.ID)
			}
		}
		if len(sq.CategoryIDs) == 0 {
			return nil, 0, nil
		}
	}
	res, err := call(d, func(c *cf.Client) (cf.SearchResult, error) { return c.Search(ctx, sq) })
	if err != nil {
		return nil, 0, err
	}
	for i := range res.Mods {
		if res.Mods[i].ClassID == 0 {
			res.Mods[i].ClassID = classID
		}
	}
	return res.Mods, res.Pagination.TotalCount, nil
}

// mergedCap is how deep a merged multi-class search goes: past it a page is empty and the total stops here. A merged
// page needs every class's leading rows, so depth is what costs requests.
const mergedCap = 1000

// rowsTTL is how long a class's fetched rows serve later pages of the same search.
const rowsTTL = 10 * time.Minute

// classRows are the leading rows of one class's search. at is when the first of them was fetched, which is their age
// however often they are read; last is when rows were last added, which orders eviction.
type classRows struct {
	mods  []cf.Mod
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
func (d Driver) classRowsTo(ctx context.Context, gameID int, class string, q source.Query, end int) ([]cf.Mod, int, error) {
	key := strings.Join([]string{cmp.Or(d.URL, cf.BaseURL), strconv.Itoa(gameID), class, q.Text, q.Sort, strings.Join(q.Categories, ",")}, "|")
	rowsMu.Lock()
	held, ok := rowsCache[key]
	if ok && time.Since(held.at) > rowsTTL {
		delete(rowsCache, key)
		held = classRows{}
	}
	rowsMu.Unlock()
	fetched := false
	for len(held.mods) < end && (len(held.mods) == 0 || len(held.mods) < held.total) {
		rows, n, err := d.searchClass(ctx, gameID, class, q, len(held.mods), min(cf.MaxPageSize, end-len(held.mods)))
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
// page needs every class's leading rows. The merge takes the best of the classes' next rows and keeps each class's own
// order, which the API's sort need not match exactly (Popularity is not the thumbs-up count), so the merged order of
// the first rows never changes as later pages read deeper; ties go by rank, then class.
func (d Driver) searchClasses(ctx context.Context, gameID int, classes []string, q source.Query, index int) (mods []cf.Mod, total int, failed []string, err error) {
	end := index + source.PageSize
	var lanes [][]cf.Mod
	var errs []error
	for _, class := range classes {
		got, n, err := d.classRowsTo(ctx, gameID, class, q, end)
		if err != nil {
			errs = append(errs, fmt.Errorf("class %s: %w", class, err))
			failed = append(failed, class)
			continue
		}
		total += n
		lanes = append(lanes, got)
	}
	if len(failed) == len(classes) {
		return nil, 0, nil, errors.Join(errs...)
	}
	next := make([]int, len(lanes))
	var all []cf.Mod
	for len(all) < end {
		best := -1
		for c, lane := range lanes {
			if next[c] >= len(lane) {
				continue
			}
			if best < 0 {
				best = c
				continue
			}
			if cmp.Or(compareBySort(q.Sort, lane[next[c]], lanes[best][next[best]]), cmp.Compare(next[c], next[best])) < 0 {
				best = c
			}
		}
		if best < 0 {
			break
		}
		all = append(all, lanes[best][next[best]])
		next[best]++
	}
	if len(failed) > 0 {
		log.Printf("curseforge: search of %d classes: %d answered with %d mods, %d failed (%s): %v",
			len(classes), len(classes)-len(failed), len(all), len(failed), strings.Join(failed, ","), errors.Join(errs...))
	}
	if index >= len(all) {
		return nil, total, failed, nil
	}
	return slices.Clone(all[index:]), total, failed, nil
}

// compareBySort orders two mods as the API's sortField does: negative when a comes first.
func compareBySort(sort string, a, b cf.Mod) int {
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

// categoryLists caches each class's categories for the process, keyed by API address, game and class.
var categoryLists sync.Map

func (d Driver) classCategories(ctx context.Context, gameID int, classID string) ([]cf.Category, error) {
	key := cmp.Or(d.URL, cf.BaseURL) + "|" + strconv.Itoa(gameID) + "|" + classID
	if v, ok := categoryLists.Load(key); ok {
		if cats, ok := v.([]cf.Category); ok {
			return cats, nil
		}
	}
	class, err := strconv.Atoi(classID)
	if err != nil {
		return nil, fmt.Errorf("CurseForge class %q is not a number", classID)
	}
	cats, err := call(d, func(c *cf.Client) ([]cf.Category, error) { return c.Categories(ctx, gameID, class) })
	if err != nil {
		return nil, err
	}
	categoryLists.Store(key, cats)
	return cats, nil
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

func (d Driver) mod(ctx context.Context, id string) (cf.Mod, error) {
	n, err := projectID(id)
	if err != nil {
		return cf.Mod{}, err
	}
	return call(d, func(c *cf.Client) (cf.Mod, error) { return c.Mod(ctx, n) })
}

// files lists the mod's newest published files, at most one API page.
func (d Driver) files(ctx context.Context, id string) ([]cf.File, error) {
	n, err := projectID(id)
	if err != nil {
		return nil, err
	}
	res, err := call(d, func(c *cf.Client) (cf.FilesResult, error) { return c.Files(ctx, n, cf.FilesQuery{}) })
	return res.Files, err
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
	key := cmp.Or(d.URL, cf.BaseURL) + "|" + id
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

// Details reads the mod's description, its file names and the required mods of its newest file. The long description
// is HTML on the site and becomes Markdown here; the summary stands in when the project has none.
func (d Driver) Details(ctx context.Context, _, id, _ string) (source.Details, error) {
	m, err := d.mod(ctx, id)
	if err != nil {
		return source.Details{}, err
	}
	files, err := d.files(ctx, id)
	if err != nil {
		return source.Details{}, err
	}
	n, _ := strconv.Atoi(id)
	long, err := call(d, func(c *cf.Client) (string, error) { return c.Description(ctx, n) })
	if err != nil {
		return source.Details{}, err
	}
	slices.SortStableFunc(files, func(a, b cf.File) int { return strings.Compare(b.FileDate, a.FileDate) })
	out := source.Details{Description: cmp.Or(source.HTMLMarkdown(long), m.Summary), Versions: []string{}, Dependencies: []string{}, Categories: []string{}}
	for _, f := range files {
		if f.IsAvailable {
			out.Versions = append(out.Versions, fileVersion(f))
			out.Released = append(out.Released, source.Released(f.FileDate))
		}
	}
	if f, ok := cf.Newest(files, nil, source.StabilityRelease); ok {
		out.Dependencies = append(out.Dependencies, required(f)...)
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
	// mod accepted id, so it is a positive number.
	n, _ := strconv.Atoi(id)
	f, err := d.pick(ctx, n, version)
	if err != nil {
		return Resolved{}, err
	}
	page := cmp.Or(m.Links.WebsiteURL, d.ModPageURL("", id))
	if m.Restricted() {
		return Resolved{}, &NotDistributableError{Mod: m.Name, PageURL: page}
	}
	link := f.DownloadURL
	if link == "" {
		link, err = call(d, func(c *cf.Client) (string, error) { return c.DownloadURL(ctx, n, f.ID) })
		if cf.Denied(err) || errors.Is(err, cf.ErrNoDownload) {
			return Resolved{}, &NotDistributableError{Mod: m.Name, PageURL: page}
		}
		if err != nil {
			return Resolved{}, err
		}
	}
	if link == "" {
		return Resolved{}, &NotDistributableError{Mod: m.Name, PageURL: page}
	}
	res := Resolved{
		ModID: id, Name: m.Name, Version: fileVersion(f), VersionID: strconv.Itoa(f.ID), FileName: f.FileName,
		URL: link, Size: f.FileLength, Dependencies: required(f),
	}
	if h := f.SHA1(); h != "" {
		res.Digest = "sha1:" + h
	}
	return res, nil
}

func (d Driver) pick(ctx context.Context, id int, version string) (cf.File, error) {
	if n, err := strconv.Atoi(version); err == nil && n > 0 {
		return call(d, func(c *cf.Client) (cf.File, error) { return c.File(ctx, id, n) })
	}
	files, err := d.files(ctx, strconv.Itoa(id))
	if err != nil {
		return cf.File{}, err
	}
	if version != "" {
		files = slices.DeleteFunc(files, func(f cf.File) bool { return fileVersion(f) != version && f.DisplayName != version })
	}
	if f, ok := cf.Newest(files, nil, source.StabilityRelease); ok {
		return f, nil
	}
	return cf.File{}, fmt.Errorf("%d %s is not on CurseForge", id, version)
}

// listed holds the required dependencies of files an update check read, keyed by mod id and version, so
// Dependencies answers without another request. Oldest keys drop once the cap is hit.
var listed source.ListedStore

func remember(f cf.File) {
	listed.Put(source.VersionRef{ID: strconv.Itoa(f.ModID), Version: fileVersion(f)}, required(f))
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
		mods, err := call(d, func(c *cf.Client) ([]cf.Mod, error) { return c.Mods(ctx, chunk) })
		if err != nil {
			return nil, err
		}
		for _, m := range mods {
			for _, have := range byMod[m.ID] {
				f, ok := cf.Newest(m.LatestFiles, src.GameVersions, source.Stability(have.Channel))
				if !ok {
					continue
				}
				remember(f)
				if have.Digest == "sha1:"+f.SHA1() || have.Version == fileVersion(f) {
					continue
				}
				out[have.Digest] = source.Latest{
					ProjectID: strconv.Itoa(m.ID), Version: fileVersion(f), VersionID: strconv.Itoa(f.ID),
					URL: cmp.Or(m.Links.WebsiteURL, Driver{}.ModPageURL("", strconv.Itoa(m.ID))),
				}
			}
		}
	}
	return out, nil
}

// Dependencies answers from the files update checks read: each known file's required mods, by id.
func (Driver) Dependencies(_ context.Context, _, _ string, refs []source.VersionRef) (map[source.VersionRef][]string, error) {
	return listed.GetAll(refs), nil
}

// Changelog reads the file changelogs of the files published after the installed one, up to latest, newest first. The
// API serves each as HTML, which becomes plain text. Without the installed file in the newest page of files, the
// newest ones up to the cap stand in.
func (d Driver) Changelog(ctx context.Context, _, id, installed, latest, _ string) ([]source.Note, error) {
	files, err := d.files(ctx, id)
	if err != nil {
		return nil, err
	}
	files = slices.DeleteFunc(files, func(f cf.File) bool { return !f.IsAvailable })
	slices.SortStableFunc(files, func(a, b cf.File) int { return strings.Compare(b.FileDate, a.FileDate) })
	since, until := "", "\xff"
	for _, f := range files {
		switch {
		case fileVersion(f) == installed:
			since = f.FileDate
		case fileVersion(f) == latest || strconv.Itoa(f.ID) == latest:
			until = f.FileDate
		}
	}
	out := []source.Note{}
	for _, f := range files {
		if f.FileDate <= since || f.FileDate > until {
			continue
		}
		if len(out) == source.MaxNotes {
			break
		}
		body, err := call(d, func(c *cf.Client) (string, error) { return c.Changelog(ctx, f.ModID, f.ID) })
		if err != nil {
			return nil, err
		}
		date, _, _ := strings.Cut(f.FileDate, "T")
		out = append(out, source.Note{Version: fileVersion(f), Date: date, Body: source.HTMLText(body)})
	}
	return out, nil
}

// ReportTarget is the project's issue tracker, its source repository when that is on GitHub, else its page. A tracker on
// GitHub reports to the repository so the report arrives prefilled.
func (d Driver) ReportTarget(ctx context.Context, _, id, _ string) (source.ReportTarget, error) {
	m, err := d.mod(ctx, id)
	if err != nil {
		return source.ReportTarget{}, err
	}
	repo := cmp.Or(source.GitHubRepo(m.Links.IssuesURL), source.GitHubRepo(m.Links.SourceURL))
	return source.ReportTarget{
		Repo: repo, URL: cmp.Or(secureLink(m.Links.IssuesURL), secureLink(m.Links.WebsiteURL), d.ModPageURL("", id)),
	}, nil
}

// secureLink is raw when it is an https address without credentials, else empty: the site supplies it and the
// window opens it.
func secureLink(raw string) string {
	u, err := url.Parse(source.WebLink(raw))
	if err != nil || u.Scheme != "https" || u.User != nil {
		return ""
	}
	return raw
}
