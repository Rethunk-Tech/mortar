// Package modio is the mod.io source driver, over the REST API v1 with the player's own read-only OAuth access token
// (mod.io/me/access), kept in the OS keyring under "modio"; without one the driver reports ErrNeedsToken and browse
// disables it. A game's source key is its numeric mod.io game id.
package modio

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// AccountURL is the API root for calls that name no game; a game's own calls go to its g-<id>.modapi.io host.
const AccountURL = "https://api.mod.io/v1"

const (
	keyName = "modio"
	// platform picks each mod's Windows file: every game modded through mod.io here runs its Windows build, under
	// Proton on Linux.
	platform = "windows"
	// maturityExplicit is the maturity_option bit for explicit content.
	maturityExplicit = 8
)

// ErrNeedsToken means no mod.io access token is stored.
var ErrNeedsToken = errors.New("needs a mod.io access token")

// ErrBadToken means mod.io rejected the token: it was revoked, expired or mistyped, and tokens do not refresh.
var ErrBadToken = errors.New("mod.io rejected the access token, sign in again")

// gameKey is a numeric mod.io game id; it names the API host, so nothing else may reach the address.
var gameKey = regexp.MustCompile(`^[0-9]{1,10}$`)

// Driver searches mod.io. HTTP, URL and Token are for tests; the zero value talks to the real API with the keyring's
// token. URL, when set, replaces both the game hosts and AccountURL.
type Driver struct {
	HTTP  *http.Client
	URL   string
	Token func() (string, error)
}

var _ = source.Register(Driver{})

// ID is the catalog id.
func (Driver) ID() string { return "modio" }

// Name is the site's name.
func (Driver) Name() string { return "mod.io" }

// Modes says Mortar downloads mod files itself.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Download} }

// Hosts is the site's web host.
func (Driver) Hosts() []string { return []string{"mod.io"} }

// MaxPageSize is the API's largest _limit.
func (Driver) MaxPageSize() int { return 100 }

// Sorts are the orders the API sorts by besides best match; "endorsements" is its weighted rating.
func (Driver) Sorts() []string {
	return []string{source.SortDownloads, source.SortEndorsements, source.SortUpdated, source.SortNewest, source.SortName}
}

func sortField(sort string) string {
	switch sort {
	case source.SortDownloads:
		return "-downloads_total"
	case source.SortEndorsements:
		return "-rating"
	case source.SortUpdated:
		return "-date_updated"
	case source.SortNewest:
		return "-date_live"
	case source.SortName:
		return "name"
	}
	return "-popular"
}

func (d Driver) token() (string, error) {
	get := d.Token
	if get == nil {
		get = func() (string, error) { return secret.Get(keyName) }
	}
	t, err := get()
	if errors.Is(err, secret.ErrNotFound) || (err == nil && strings.TrimSpace(t) == "") {
		return "", ErrNeedsToken
	}
	return strings.TrimSpace(t), err
}

// Unavailable says why browse cannot use the source now, or "" when it can.
func (d Driver) Unavailable() string {
	if _, err := d.token(); errors.Is(err, ErrNeedsToken) {
		return ErrNeedsToken.Error()
	}
	return ""
}

func (d Driver) base(key string) (string, error) {
	if !gameKey.MatchString(key) {
		return "", fmt.Errorf("%q is not a mod.io game id", key)
	}
	return cmp.Or(d.URL, "https://g-"+key+".modapi.io/v1"), nil
}

// call GETs root+path with the token. A 429 is a *source.BusyError carrying the API's retry-after.
func (d Driver) call(ctx context.Context, token, root, path string, params url.Values, out any) error {
	err := source.DoJSON(ctx, source.Request{
		Service: "mod.io", Client: d.HTTP, URL: strings.TrimRight(root, "/") + path, Params: params,
		UserAgent: source.UserAgentWithSite(""),
		Header: func(h http.Header) {
			h.Set("Authorization", "Bearer "+token)
			h.Set("X-Modio-Platform", platform)
		},
	}, out)
	if se, ok := errors.AsType[*source.StatusError](err); ok && se.Code == http.StatusUnauthorized {
		return ErrBadToken
	}
	return err
}

// game calls a path under the game's API host.
func (d Driver) game(ctx context.Context, key, path string, params url.Values, out any) error {
	token, err := d.token()
	if err != nil {
		return err
	}
	root, err := d.base(key)
	if err != nil {
		return err
	}
	return d.call(ctx, token, root, "/games/"+key+path, params, out)
}

type modfile struct {
	ID        int    `json:"id"`
	Filesize  int64  `json:"filesize"`
	Filename  string `json:"filename"`
	Version   string `json:"version"`
	Changelog string `json:"changelog"`
	Filehash  struct {
		MD5 string `json:"md5"`
	} `json:"filehash"`
	Download struct {
		BinaryURL string `json:"binary_url"`
	} `json:"download"`
}

// version is the file's own version, or its id when the author left it blank.
func (f modfile) version() string { return cmp.Or(strings.TrimSpace(f.Version), strconv.Itoa(f.ID)) }

type mod struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	ProfileURL  string `json:"profile_url"`
	DateUpdated int64  `json:"date_updated"`
	Maturity    int    `json:"maturity_option"`
	SubmittedBy struct {
		Username string `json:"username"`
	} `json:"submitted_by"`
	Logo struct {
		Thumb string `json:"thumb_320x180"`
	} `json:"logo"`
	Stats struct {
		Downloads       int `json:"downloads_total"`
		RatingsPositive int `json:"ratings_positive"`
	} `json:"stats"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
	Modfile modfile `json:"modfile"`
}

type modList struct {
	Data  []mod `json:"data"`
	Total int   `json:"result_total"`
}

func (d Driver) item(m mod) source.Item {
	it := source.Item{
		Source: d.ID(), ID: strconv.Itoa(m.ID), Name: m.Name, Summary: m.Summary, Author: m.SubmittedBy.Username,
		Picture: m.Logo.Thumb, Endorsements: m.Stats.RatingsPositive, Downloads: m.Stats.Downloads,
		// A mod's page is named by the game's and the mod's slugs, which only the API's answer gives.
		Size: m.Modfile.Filesize, URL: source.WebLink(m.ProfileURL),
		Adult: m.Maturity&maturityExplicit != 0,
	}
	if m.Modfile.ID != 0 {
		it.Version = m.Modfile.version()
	}
	if m.DateUpdated > 0 {
		it.Updated = time.Unix(m.DateUpdated, 0).UTC().Format(time.RFC3339)
	}
	return it
}

// Search lists the game's live mods matching the text, filtered by tag; the game version is Mortar's own, so it is
// not a filter.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	params := url.Values{}
	if t := strings.TrimSpace(q.Text); t != "" {
		params.Set("_q", t)
	}
	params.Set("_sort", sortField(q.Sort))
	params.Set("_limit", strconv.Itoa(q.Len()))
	params.Set("_offset", strconv.Itoa(max(q.Page-source.FirstPage, 0)*q.Len()))
	if len(q.Categories) > 0 {
		params.Set("tags-in", strings.Join(q.Categories, ","))
	}
	if len(q.ExcludeCategories) > 0 {
		params.Set("tags-not-in", strings.Join(q.ExcludeCategories, ","))
	}
	var parsed modList
	if err := d.game(ctx, q.Key, "/mods", params, &parsed); err != nil {
		return source.Page{}, err
	}
	items := make([]source.Item, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		items = append(items, d.item(m))
	}
	return source.Page{Total: parsed.Total, Items: items}, nil
}

// Categories lists the game's visible tag names.
func (d Driver) Categories(ctx context.Context, key string) ([]string, error) {
	var parsed struct {
		Data []struct {
			Hidden bool     `json:"hidden"`
			Tags   []string `json:"tags"`
		} `json:"data"`
	}
	if err := d.game(ctx, key, "/tags", nil, &parsed); err != nil {
		return nil, err
	}
	var names []string
	for _, g := range parsed.Data {
		if !g.Hidden {
			names = append(names, g.Tags...)
		}
	}
	return source.UniqueNames(names), nil
}

func (d Driver) files(ctx context.Context, key, id string) ([]modfile, error) {
	var parsed struct {
		Data []modfile `json:"data"`
	}
	params := url.Values{"_sort": {"-date_added"}, "_limit": {"100"}}
	err := d.game(ctx, key, "/mods/"+url.PathEscape(id)+"/files", params, &parsed)
	return parsed.Data, err
}

// Details reads the mod's description, tags, its files' versions newest first, the newest file's changelog and the
// names of the mods it depends on. mod.io dependencies include credit links, so they are only listed.
func (d Driver) Details(ctx context.Context, key, id, _ string) (source.Details, error) {
	var m mod
	if err := d.game(ctx, key, "/mods/"+url.PathEscape(id), nil, &m); err != nil {
		return source.Details{}, err
	}
	files, err := d.files(ctx, key, id)
	if err != nil {
		return source.Details{}, err
	}
	var deps modList
	if err := d.game(ctx, key, "/mods/"+url.PathEscape(id)+"/dependencies", nil, &deps); err != nil {
		return source.Details{}, err
	}
	out := source.Details{
		Description: source.HTMLMarkdown(m.Description), Versions: []string{}, Dependencies: []string{}, Categories: []string{},
	}
	for _, t := range m.Tags {
		out.Categories = append(out.Categories, t.Name)
	}
	out.Categories = source.UniqueNames(out.Categories)
	for _, f := range files {
		out.Versions = append(out.Versions, f.version())
	}
	if len(files) > 0 {
		out.Changelog = strings.TrimSpace(files[0].Changelog)
	}
	for _, dep := range deps.Data {
		out.Dependencies = append(out.Dependencies, dep.Name)
	}
	return out, nil
}

// Resolved is what an install needs of one mod file.
type Resolved struct {
	ModID, Name, Version, FileName string
	FileID                         int
	Size                           int64
	// URL is the download address read for this call, since mod.io may expire it.
	URL string
	// MD5 is the hex digest mod.io publishes for the file, its only hash.
	MD5 string
}

// Resolve reads the mod's file named by version (a file id or version number), or its current file when version is
// empty, fresh for each download.
func (d Driver) Resolve(ctx context.Context, key, id, version string) (Resolved, error) {
	var m mod
	if err := d.game(ctx, key, "/mods/"+url.PathEscape(id), nil, &m); err != nil {
		return Resolved{}, err
	}
	f := m.Modfile
	if version != "" && version != strconv.Itoa(f.ID) && version != f.version() {
		files, err := d.files(ctx, key, id)
		if err != nil {
			return Resolved{}, err
		}
		i := slices.IndexFunc(files, func(f modfile) bool { return version == strconv.Itoa(f.ID) || version == f.version() })
		if i < 0 {
			return Resolved{}, fmt.Errorf("mod.io mod %s has no file %q", id, version)
		}
		f = files[i]
	}
	if f.ID == 0 || source.WebLink(f.Download.BinaryURL) == "" {
		return Resolved{}, fmt.Errorf("mod.io mod %s has no file to download", id)
	}
	return Resolved{
		ModID: id, Name: m.Name, Version: f.version(), FileName: f.Filename, FileID: f.ID, Size: f.Filesize,
		URL: f.Download.BinaryURL, MD5: strings.ToLower(f.Filehash.MD5),
	}, nil
}

// latestBatch is how many mods one update request names, the API's largest page.
const latestBatch = 100

// Latest reads the current file of every installed mod, a hundred mods a request. mod.io has no release channels, so
// a mod whose current file's MD5 differs from the installed one has an update.
func (d Driver) Latest(ctx context.Context, src components.GameSource, _ string, files []source.InstalledFile) (map[string]source.Latest, error) {
	out := map[string]source.Latest{}
	for start := 0; start < len(files); start += latestBatch {
		chunk := files[start:min(start+latestBatch, len(files))]
		ids := make([]string, len(chunk))
		for i, f := range chunk {
			ids[i] = f.ID
		}
		var parsed modList
		params := url.Values{"id-in": {strings.Join(ids, ",")}, "_limit": {strconv.Itoa(latestBatch)}}
		if err := d.game(ctx, src.Key, "/mods", params, &parsed); err != nil {
			return nil, err
		}
		for _, f := range chunk {
			i := slices.IndexFunc(parsed.Data, func(m mod) bool { return strconv.Itoa(m.ID) == f.ID })
			if i < 0 {
				continue
			}
			m := parsed.Data[i]
			if m.Modfile.ID == 0 || "md5:"+strings.ToLower(m.Modfile.Filehash.MD5) == strings.ToLower(f.Digest) {
				continue
			}
			out[f.Digest] = source.Latest{
				ProjectID: f.ID, Version: m.Modfile.version(), VersionID: strconv.Itoa(m.Modfile.ID), URL: source.WebLink(m.ProfileURL),
			}
		}
	}
	return out, nil
}

// Validate checks token against the API and returns the account's username.
func (d Driver) Validate(ctx context.Context, token string) (string, error) {
	var me struct {
		Username string `json:"username"`
	}
	if err := d.call(ctx, strings.TrimSpace(token), cmp.Or(d.URL, AccountURL), "/me", nil, &me); err != nil {
		return "", err
	}
	return cmp.Or(me.Username, "mod.io"), nil
}
