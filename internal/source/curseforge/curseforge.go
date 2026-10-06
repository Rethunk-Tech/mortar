// Package curseforge is the CurseForge source driver, over the Core API v1 with Mortar's own API key. A game opts in
// with a source entry whose gameId is the site's game id and whose key is the id of its "Mods" class. The key is set
// at release build time (apiKey) or, for dev and test builds, by MORTAR_CURSEFORGE_KEY; without either the driver
// reports itself unavailable and browse disables it. Authors can forbid distribution outside CurseForge: a file with
// no download address is never fetched around the API, and the queue hands the user the mod's page instead.
package curseforge

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

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
	// GameSource returns the game's CurseForge entry; nil reads the catalog.
	GameSource func(game string) (components.GameSource, bool)
}

var _ = source.Register(Driver{})

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
	base := cmp.Or(d.URL, BaseURL)
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
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, payload)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", key)
	req.Header.Set("User-Agent", source.UserAgent(""))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		return source.Busy("CurseForge", resp)
	case http.StatusUnauthorized, http.StatusForbidden:
		return &statusError{code: resp.StatusCode}
	default:
		return fmt.Errorf("curseforge answered %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, source.MaxBody))
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// statusError is a 401 or 403, which a caller may read as "not allowed" for one request.
type statusError struct{ code int }

func (e *statusError) Error() string { return fmt.Sprintf("curseforge answered HTTP %d", e.code) }

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
)

func sortField(sort string) int {
	switch sort {
	case source.SortDownloads:
		return sortDownloads
	case source.SortUpdated:
		return sortUpdated
	case source.SortEndorsements:
		return sortPopular
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
	for _, ext := range []string{".zip", ".7z", ".rar"} {
		name = strings.TrimSuffix(name, ext)
	}
	return name
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

// Search lists mods in the game's Mods class. Category filters are not applied: the API filters by category id,
// and the catalog names none.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	gs, ok := d.gameSource(q.Game)
	if !ok || gs.GameID == 0 || q.Key == "" {
		return source.Page{}, fmt.Errorf("game %q has no CurseForge game and class id", q.Game)
	}
	index := max(q.Page-source.FirstPage, 0) * source.PageSize
	if index+source.PageSize > maxWindow {
		return source.Page{Total: maxWindow}, nil
	}
	order := "desc"
	if q.Sort == source.SortName {
		order = "asc"
	}
	params := url.Values{}
	params.Set("gameId", strconv.Itoa(gs.GameID))
	params.Set("classId", q.Key)
	params.Set("searchFilter", q.Text)
	params.Set("sortField", strconv.Itoa(sortField(q.Sort)))
	params.Set("sortOrder", order)
	params.Set("index", strconv.Itoa(index))
	params.Set("pageSize", strconv.Itoa(source.PageSize))
	var parsed struct {
		Data       []modResp `json:"data"`
		Pagination struct {
			TotalCount int `json:"totalCount"`
		} `json:"pagination"`
	}
	if err := d.get(ctx, "/mods/search", params, &parsed); err != nil {
		return source.Page{}, err
	}
	items := make([]source.Item, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		it := source.Item{
			Source: d.ID(), ID: strconv.Itoa(m.ID), Name: m.Name, Summary: m.Summary, Picture: m.Logo.ThumbnailURL,
			Endorsements: m.ThumbsUpCount, Downloads: m.DownloadCount, Updated: m.DateModified,
			URL: cmp.Or(m.Links.WebsiteURL, d.ModPageURL(q.Key, strconv.Itoa(m.ID))), Repo: source.GitHubRepo(m.Links.SourceURL),
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
	return source.Page{Total: min(parsed.Pagination.TotalCount, maxWindow), Items: items}, nil
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
			if _, denied := errors.AsType[*statusError](err); denied {
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
