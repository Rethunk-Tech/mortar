// Package paradox is the Paradox Mods (PDX Mods) source driver, over the public API behind mods.paradoxplaza.com. A
// mod's id is its numeric mod id; the API names the files by a second, opaque name the details answer holds. Mods are
// read and downloaded anonymously.
package paradox

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// BaseURL is the real API.
const BaseURL = "https://api.paradox-interactive.com"

const (
	siteURL    = "https://mods.paradoxplaza.com"
	contentURL = "https://modscontent.paradox-interactive.com"
	// platform is the build the details are read for: the API answers a mod only for an operating system it was
	// published for, and every mod lists Windows.
	platform = "Windows"
)

// games are the API's names for the games the driver serves; a mod id is global across Paradox's games, so a mod of
// another game is refused.
var games = []string{"cities_skylines_2"}

// Driver searches Paradox Mods. HTTP and URL are for tests; the zero value talks to the real API.
type Driver struct {
	HTTP *http.Client
	URL  string
}

var _ = source.Register(Driver{})

// ID is the catalog id.
func (Driver) ID() string { return "paradox" }

// Name is the site's name.
func (Driver) Name() string { return "Paradox Mods" }

// Modes says Mortar downloads the mod archives itself.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Download} }

// Hosts is the site's web host.
func (Driver) Hosts() []string { return []string{"mods.paradoxplaza.com"} }

var (
	modIDPattern = regexp.MustCompile(`^[0-9]{1,9}$`)
	// namePattern is the opaque file name of a mod, a UUID in practice; it enters a download address.
	namePattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
)

// ValidID reports whether id is a Paradox Mods mod id.
func ValidID(id string) bool { return modIDPattern.MatchString(id) }

// ModPageURL is the mod's page; gameKey does not enter the address.
func (Driver) ModPageURL(_, id string) string {
	if !modIDPattern.MatchString(id) {
		return ""
	}
	return siteURL + "/mods/" + id + "/" + platform
}

func (d Driver) do(ctx context.Context, method, path string, params url.Values, body, out any, version string) error {
	return source.DoJSON(ctx, source.Request{
		Service: "Paradox Mods", Client: d.HTTP, Method: method,
		URL: strings.TrimRight(cmp.Or(d.URL, BaseURL), "/") + path, Params: params, Body: body,
		UserAgent: source.UserAgentWithSite(version),
	}, out)
}

// answer is the envelope every call is wrapped in; a refusal can arrive as a 200.
type answer struct {
	Result       string `json:"result"`
	ErrorMessage string `json:"errorMessage"`
}

func (a answer) err() error {
	if a.Result == "OK" {
		return nil
	}
	return fmt.Errorf("the answer from Paradox Mods was: %s", cmp.Or(a.ErrorMessage, "refused"))
}

// MaxPageSize is the API's largest page.
func (Driver) MaxPageSize() int { return 100 }

// Sorts are the API's orders other than best match; it sorts by name ascending and everything else descending.
func (Driver) Sorts() []string {
	return []string{source.SortDownloads, source.SortEndorsements, source.SortUpdated, source.SortNewest, source.SortName}
}

func sortBy(sort string) (by, order string) {
	switch sort {
	case source.SortDownloads:
		return "popularity", "desc"
	case source.SortEndorsements:
		return "rating", "desc"
	case source.SortUpdated:
		return "updated", "desc"
	case source.SortNewest:
		return "created", "desc"
	case source.SortName:
		return "displayName", "asc"
	}
	return "", ""
}

type listedMod struct {
	ModID            string   `json:"modId"`
	DisplayName      string   `json:"displayName"`
	Author           string   `json:"author"`
	ShortDescription string   `json:"shortDescription"`
	InstalledCount   int      `json:"installedCount"`
	RatingsTotal     int      `json:"ratingsTotal"`
	Tags             []string `json:"tags"`
	Updated          string   `json:"updated"`
	Image            string   `json:"displayImagePath"`
}

// Search lists the game's mods. The API has no category list, but filters by tags and a mod carries its own, so the
// tags a hit shows are what Browse can filter on. It keeps hits carrying all of the included tags; excluded ones are
// dropped from the page and counted in Hidden.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	if !slices.Contains(games, q.Key) {
		return source.Page{}, fmt.Errorf("game %q has no Paradox Mods name", q.Game)
	}
	body := map[string]any{"gameName": q.Key, "page": max(q.Page, source.FirstPage), "pageSize": min(q.Len(), d.MaxPageSize())}
	if q.Text != "" {
		body["searchQuery"] = q.Text
	}
	if len(q.Categories) > 0 {
		body["tags"] = q.Categories
	}
	by, order := sortBy(q.Sort)
	if by == "" && q.Text == "" {
		by, order = "best", "desc"
	}
	if by != "" {
		body["sortBy"], body["orderBy"] = by, order
	}
	var parsed struct {
		answer
		TotalCount int         `json:"totalCount"`
		Mods       []listedMod `json:"mods"`
	}
	if err := d.do(ctx, http.MethodPost, "/mods", nil, body, &parsed, q.Version); err != nil {
		return source.Page{}, err
	}
	if err := parsed.err(); err != nil {
		return source.Page{}, err
	}
	out := source.Page{Total: parsed.TotalCount, Items: []source.Item{}}
	for _, m := range parsed.Mods {
		if !modIDPattern.MatchString(m.ModID) {
			continue
		}
		if !source.CategoryMatch(m.Tags, nil, q.ExcludeCategories) {
			out.Hidden++
			continue
		}
		out.Items = append(out.Items, source.Item{
			Source: d.ID(), ID: m.ModID, Name: m.DisplayName, Summary: m.ShortDescription, Author: m.Author,
			Picture: source.WebLink(m.Image), Endorsements: m.RatingsTotal, Downloads: m.InstalledCount,
			Updated: m.Updated, URL: d.ModPageURL(q.Key, m.ModID), Category: firstOf(m.Tags),
		})
	}
	return out, nil
}

func firstOf(list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[0]
}

type requirement struct {
	ProductType string `json:"productType"`
	ID          string `json:"id"`
}

type changeEntry struct {
	ModVersion  int    `json:"modVersion"`
	Released    string `json:"released"`
	Details     string `json:"details"`
	UserVersion string `json:"userModVersion"`
}

type modDetail struct {
	answer
	Detail struct {
		Game             string        `json:"game"`
		Name             string        `json:"name"`
		ModID            string        `json:"modId"`
		DisplayName      string        `json:"displayName"`
		Author           string        `json:"author"`
		LongDescription  string        `json:"longDescription"`
		ModVersion       string        `json:"modVersion"`
		UserModVersion   string        `json:"userModVersion"`
		Tags             []string      `json:"tags"`
		Requirements     []requirement `json:"requirements"`
		Changelog        []changeEntry `json:"changelog"`
		DisplayImagePath string        `json:"displayImagePath"`
	} `json:"modDetail"`
}

// requiredMods are the ids of the mods the newest version needs; the API lists other products too (the game's DLC).
func (m modDetail) requiredMods() []string {
	out := []string{}
	for _, r := range m.Detail.Requirements {
		if r.ProductType == "MOD" && modIDPattern.MatchString(r.ID) && r.ID != m.Detail.ModID {
			out = append(out, r.ID)
		}
	}
	return out
}

// mod reads one mod's details, refusing a mod of a game the driver does not serve.
func (d Driver) mod(ctx context.Context, id, mortarVersion string) (modDetail, error) {
	if !modIDPattern.MatchString(id) {
		return modDetail{}, fmt.Errorf("%q is not a Paradox Mods id", id)
	}
	var parsed modDetail
	if err := d.do(ctx, http.MethodGet, "/mods", url.Values{"modId": {id}, "os": {platform}}, nil, &parsed, mortarVersion); err != nil {
		return modDetail{}, err
	}
	if err := parsed.err(); err != nil {
		return modDetail{}, err
	}
	if !slices.Contains(games, parsed.Detail.Game) {
		return modDetail{}, fmt.Errorf("mod %s is not a Cities: Skylines II mod", id)
	}
	return parsed, nil
}

// version is one published version: ID is the number the download address carries, Name what the author calls it
// (empty when the author set none, which Label then replaces with the number).
type version struct {
	ID      int    `json:"id"`
	Created string `json:"created"`
	Size    int64  `json:"size"`
	Name    string `json:"userVersion"`
}

// Label is the version as Mortar shows and stores it.
func (v version) Label() string { return cmp.Or(v.Name, strconv.Itoa(v.ID)) }

// versions lists the mod's versions, newest first.
func (d Driver) versions(ctx context.Context, id, mortarVersion string) ([]version, error) {
	if !modIDPattern.MatchString(id) {
		return nil, fmt.Errorf("%q is not a Paradox Mods id", id)
	}
	var parsed struct {
		answer
		ModVersions []version `json:"modVersions"`
	}
	if err := d.do(ctx, http.MethodGet, "/mods/versions", url.Values{"modId": {id}}, nil, &parsed, mortarVersion); err != nil {
		return nil, err
	}
	if err := parsed.err(); err != nil {
		return nil, err
	}
	slices.SortFunc(parsed.ModVersions, func(a, b version) int { return cmp.Compare(b.ID, a.ID) })
	return parsed.ModVersions, nil
}

// Resolved is what an install needs of one version.
type Resolved struct {
	ID, Name, Version, FileName, URL string
	// VersionID is the number the download address carries; Resolve takes it as a version too.
	VersionID    int
	Size         int64
	Dependencies []string
}

// downloadURL is the anonymous archive address of version n of the mod file name; its host is fixed.
func downloadURL(game, name string, n int) string {
	return fmt.Sprintf("%s/%s/%s/content/sources/%s_%d.zip", contentURL, game, name, name, n)
}

// Resolve finds the mod's version whose number or name is version; an empty version means the newest. The API lists
// the requirements of the newest version only, so an older version is given those.
func (d Driver) Resolve(ctx context.Context, id, wanted string) (Resolved, error) {
	m, err := d.mod(ctx, id, "")
	if err != nil {
		return Resolved{}, err
	}
	if !namePattern.MatchString(m.Detail.Name) {
		return Resolved{}, fmt.Errorf("mod %s has no usable file name", id)
	}
	all, err := d.versions(ctx, id, "")
	if err != nil {
		return Resolved{}, err
	}
	pick := -1
	if wanted == "" && len(all) > 0 {
		pick = 0
	}
	if wanted != "" {
		pick = slices.IndexFunc(all, func(v version) bool { return strconv.Itoa(v.ID) == wanted })
		if pick < 0 {
			pick = slices.IndexFunc(all, func(v version) bool { return v.Name == wanted })
		}
	}
	if pick < 0 {
		return Resolved{}, fmt.Errorf("%s has no version %s on Paradox Mods", m.Detail.DisplayName, cmp.Or(wanted, "to download"))
	}
	v := all[pick]
	return Resolved{
		ID: id, Name: m.Detail.DisplayName, Version: v.Label(), VersionID: v.ID, Size: v.Size,
		FileName: fmt.Sprintf("%s_%d.zip", m.Detail.Name, v.ID), URL: downloadURL(m.Detail.Game, m.Detail.Name, v.ID),
		Dependencies: m.requiredMods(),
	}, nil
}

// Items answers for installed mods by id with each one's newest version, one small request per mod, so an update check
// compares versions without searching by name.
func (d Driver) Items(ctx context.Context, _, mortarVersion string, ids []string) (map[string]source.Item, error) {
	out := map[string]source.Item{}
	for _, id := range ids {
		if !modIDPattern.MatchString(id) {
			continue
		}
		all, err := d.versions(ctx, id, mortarVersion)
		if err != nil {
			return out, err
		}
		if len(all) == 0 {
			continue
		}
		out[strings.ToLower(id)] = source.Item{
			Source: d.ID(), ID: id, Version: all[0].Label(), Size: all[0].Size, Updated: all[0].Created, URL: d.ModPageURL("", id),
		}
	}
	return out, nil
}

// Changelog reads the notes of the versions newer than installed, up to latest, newest first.
func (d Driver) Changelog(ctx context.Context, _, id, installed, latest, mortarVersion string) ([]source.Note, error) {
	m, err := d.mod(ctx, id, mortarVersion)
	if err != nil {
		return nil, err
	}
	entries := slices.Clone(m.Detail.Changelog)
	slices.SortFunc(entries, func(a, b changeEntry) int { return cmp.Compare(b.ModVersion, a.ModVersion) })
	out := []source.Note{}
	started := latest == ""
	for _, e := range entries {
		label := cmp.Or(e.UserVersion, strconv.Itoa(e.ModVersion))
		if label == installed {
			break
		}
		started = started || label == latest
		if !started || len(out) == source.MaxNotes {
			continue
		}
		date, _, _ := strings.Cut(e.Released, " ")
		out = append(out, source.Note{Version: label, Date: date, Body: text(e.Details)})
	}
	return out, nil
}

// text is an author's text as Markdown: authors write Markdown, and some pages hold HTML instead.
func text(s string) string {
	if strings.Contains(s, "<") {
		return source.HTMLMarkdown(s)
	}
	return s
}

// Details reads the mod's page: its description, tags, every version's label and the newest version's notes and
// required mods.
func (d Driver) Details(ctx context.Context, _, id, mortarVersion string) (source.Details, error) {
	m, err := d.mod(ctx, id, mortarVersion)
	if err != nil {
		return source.Details{}, err
	}
	all, err := d.versions(ctx, id, mortarVersion)
	if err != nil {
		return source.Details{}, err
	}
	out := source.Details{
		Description: text(m.Detail.LongDescription), Versions: []string{}, Dependencies: m.requiredMods(),
		Categories: source.UniqueNames(m.Detail.Tags),
	}
	for _, v := range all {
		out.Versions = append(out.Versions, v.Label())
	}
	if len(m.Detail.Changelog) > 0 {
		newest := slices.MaxFunc(m.Detail.Changelog, func(a, b changeEntry) int { return cmp.Compare(a.ModVersion, b.ModVersion) })
		out.Changelog = text(newest.Details)
	}
	return out, nil
}
