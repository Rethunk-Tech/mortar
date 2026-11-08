// Package paradox is the Paradox Mods (PDX Mods) source driver, over the public API behind mods.paradoxplaza.com. A
// mod's id is its numeric mod id; the API names the files by a second, opaque name the details answer holds. Mods are
// read and downloaded anonymously.
package paradox

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-paradoxmods"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// BaseURL is the real API.
const BaseURL = paradoxmods.BaseURL

// platform is the build the details are read for: the API answers a mod only for an operating system it was
// published for, and every mod lists Windows.
const platform = "Windows"

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

// namePattern is the opaque file name of a mod, a UUID in practice; it enters a download address.
var namePattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

// ValidID reports whether id is a Paradox Mods mod id.
func ValidID(id string) bool { return paradoxmods.ValidID(id) }

// ModPageURL is the mod's page; gameKey does not enter the address.
func (Driver) ModPageURL(_, id string) string {
	return paradoxmods.PageURL(id)
}

func (d Driver) client(version string) *paradoxmods.Client {
	return paradoxmods.New(paradoxmods.Config{
		BaseURL:   d.URL,
		HTTP:      d.HTTP,
		UserAgent: source.UserAgentWithSite(version),
	})
}

func mapErr(err error) error {
	if api, ok := errors.AsType[*paradoxmods.APIError](err); ok {
		return fmt.Errorf("the answer from Paradox Mods was: %s", cmp.Or(api.Message, "refused"))
	}
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return &source.BusyError{Source: "Paradox Mods", Reset: rl.Reset}
	}
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return &source.StatusError{Service: "Paradox Mods", Code: se.Code, Status: se.Status}
	}
	var syntax *json.SyntaxError
	var typed *json.UnmarshalTypeError
	if errors.As(err, &syntax) || errors.As(err, &typed) {
		return fmt.Errorf("paradox mods: %w", errors.Unwrap(err))
	}
	return err
}

// MaxPageSize is the API's largest page.
func (Driver) MaxPageSize() int { return paradoxmods.MaxPageSize }

// Sorts are the orders other than best match.
func (Driver) Sorts() []string {
	return []string{source.SortDownloads, source.SortEndorsements, source.SortUpdated, source.SortNewest, source.SortName}
}

func sortOf(sort string) paradoxmods.Sort {
	switch sort {
	case source.SortDownloads:
		return paradoxmods.SortDownloads
	case source.SortEndorsements:
		return paradoxmods.SortRating
	case source.SortUpdated:
		return paradoxmods.SortUpdated
	case source.SortNewest:
		return paradoxmods.SortNewest
	case source.SortName:
		return paradoxmods.SortName
	}
	return ""
}

// Search lists the game's mods. The API has no category list, but filters by tags and a mod carries its own, so the
// tags a hit shows are what Browse can filter on. It keeps hits carrying all of the included tags; excluded ones are
// dropped from the page and counted in Hidden.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	if !slices.Contains(games, q.Key) {
		return source.Page{}, fmt.Errorf("game %q has no Paradox Mods name", q.Game)
	}
	sort := sortOf(q.Sort)
	if sort == "" && q.Text == "" {
		sort = paradoxmods.SortBest
	}
	list, err := d.client(q.Version).Search(ctx, paradoxmods.SearchQuery{
		Game: q.Key, Text: q.Text, Page: max(q.Page, source.FirstPage), Size: min(q.Len(), paradoxmods.MaxPageSize),
		Tags: q.Categories, Sort: sort,
	})
	if err != nil {
		return source.Page{}, mapErr(err)
	}
	out := source.Page{Total: list.Total, Items: []source.Item{}}
	for _, m := range list.Mods {
		if !paradoxmods.ValidID(m.ID) {
			continue
		}
		if !source.CategoryMatch(m.Tags, nil, q.ExcludeCategories) {
			out.Hidden++
			continue
		}
		out.Items = append(out.Items, source.Item{
			Source: d.ID(), ID: m.ID, Name: m.DisplayName, Summary: m.ShortDescription, Author: m.Author,
			Picture: source.WebLink(m.Image), Endorsements: m.RatingsTotal, Downloads: m.InstalledCount,
			Updated: m.Updated, URL: d.ModPageURL(q.Key, m.ID), Category: firstOf(m.Tags),
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

// mod reads one mod's details, refusing a mod of a game the driver does not serve.
func (d Driver) mod(ctx context.Context, id, mortarVersion string) (paradoxmods.Mod, error) {
	if !paradoxmods.ValidID(id) {
		return paradoxmods.Mod{}, fmt.Errorf("%q is not a Paradox Mods id", id)
	}
	m, err := d.client(mortarVersion).Mod(ctx, "", id, platform)
	if err != nil {
		return paradoxmods.Mod{}, mapErr(err)
	}
	if !slices.Contains(games, m.Game) {
		return paradoxmods.Mod{}, fmt.Errorf("mod %s is not a Cities: Skylines II mod", id)
	}
	return m, nil
}

// versions lists the mod's versions, newest first.
func (d Driver) versions(ctx context.Context, id, mortarVersion string) ([]paradoxmods.Version, error) {
	if !paradoxmods.ValidID(id) {
		return nil, fmt.Errorf("%q is not a Paradox Mods id", id)
	}
	all, err := d.client(mortarVersion).Versions(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return all, nil
}

// Resolved is what an install needs of one version.
type Resolved struct {
	ID, Name, Version, FileName, URL string
	// VersionID is the number the download address carries; Resolve takes it as a version too.
	VersionID    int
	Size         int64
	Dependencies []string
}

// Resolve finds the mod's version whose number or name is version; an empty version means the newest. The API lists
// the requirements of the newest version only, so an older version is given those.
func (d Driver) Resolve(ctx context.Context, id, wanted string) (Resolved, error) {
	m, err := d.mod(ctx, id, "")
	if err != nil {
		return Resolved{}, err
	}
	if !namePattern.MatchString(m.Name) {
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
		pick = slices.IndexFunc(all, func(v paradoxmods.Version) bool { return strconv.Itoa(v.ID) == wanted })
		if pick < 0 {
			pick = slices.IndexFunc(all, func(v paradoxmods.Version) bool { return v.Name == wanted })
		}
	}
	if pick < 0 {
		return Resolved{}, fmt.Errorf("%s has no version %s on Paradox Mods", m.DisplayName, cmp.Or(wanted, "to download"))
	}
	v := all[pick]
	return Resolved{
		ID: id, Name: m.DisplayName, Version: v.Label(), VersionID: v.ID, Size: v.Size,
		FileName: fmt.Sprintf("%s_%d.zip", m.Name, v.ID), URL: paradoxmods.DownloadURL(m.Game, m.Name, v.ID),
		Dependencies: m.Required(),
	}, nil
}

// Items answers for installed mods by id with each one's newest version, one small request per mod, so an update check
// compares versions without searching by name.
func (d Driver) Items(ctx context.Context, _, mortarVersion string, ids []string) (map[string]source.Item, error) {
	out := map[string]source.Item{}
	for _, id := range ids {
		if !paradoxmods.ValidID(id) {
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
	entries := slices.Clone(m.Changelog)
	slices.SortFunc(entries, func(a, b paradoxmods.ChangeEntry) int { return cmp.Compare(b.ModVersion, a.ModVersion) })
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
		Description: text(m.LongDescription), Versions: []string{}, Dependencies: m.Required(),
		Categories: source.UniqueNames(m.Tags),
	}
	for _, v := range all {
		out.Versions = append(out.Versions, v.Label())
		out.Released = append(out.Released, source.Released(v.Created))
	}
	if len(m.Changelog) > 0 {
		newest := slices.MaxFunc(m.Changelog, func(a, b paradoxmods.ChangeEntry) int { return cmp.Compare(a.ModVersion, b.ModVersion) })
		out.Changelog = text(newest.Details)
	}
	return out, nil
}
