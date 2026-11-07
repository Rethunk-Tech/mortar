// Package modio is the mod.io source driver over go-modio, signed in with the player's own read-only personal access
// token (mod.io/me/access), kept in the OS keyring under "modio"; without one the driver reports ErrNeedsToken and
// browse disables it. A game's source key is its numeric mod.io game id.
package modio

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
	gomodio "github.com/Rethunk-Tech/go-modio"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const (
	keyName = "modio"
	// platform picks each mod's Windows file: every game modded through mod.io here runs its Windows build, under
	// Proton on Linux.
	platform = "windows"
)

// ErrNeedsToken means no mod.io access token is stored.
var ErrNeedsToken = errors.New("needs a mod.io access token")

// ErrAPIKey means the player pasted the API key mod.io also shows on the access page, which cannot sign in.
var ErrAPIKey = errors.New("that is a mod.io API key; create a read-only personal access token on the same page and paste that")

// apiKeyShape is an API key's form; an access token is far longer.
var apiKeyShape = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

// ErrBadToken means mod.io rejected the token: it was revoked, expired or mistyped, and tokens do not refresh.
var ErrBadToken = errors.New("mod.io rejected the access token, sign in again")

// gameKey is a numeric mod.io game id.
var gameKey = regexp.MustCompile(`^[0-9]{1,10}$`)

// userIDs remembers the account id of each token, since the API host is the account's own (u-{id}.modapi.io).
var userIDs sync.Map

// Driver searches mod.io. HTTP, URL and Token are for tests; the zero value talks to the real API with the keyring's
// token. URL, when set, replaces the API root.
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
func (Driver) MaxPageSize() int { return gomodio.MaxPageSize }

// Sorts are the orders the API sorts by besides best match; "endorsements" is its weighted rating.
func (Driver) Sorts() []string {
	return []string{source.SortDownloads, source.SortEndorsements, source.SortUpdated, source.SortNewest, source.SortName}
}

func sortField(sort string) string {
	switch sort {
	case source.SortDownloads:
		return gomodio.SortDownloads
	case source.SortEndorsements:
		return gomodio.SortRating
	case source.SortUpdated:
		return gomodio.SortUpdated
	case source.SortNewest:
		return gomodio.SortNewest
	case source.SortName:
		return gomodio.SortName
	}
	return gomodio.SortPopular
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

func (d Driver) config(token string) gomodio.Config {
	return gomodio.Config{Token: token, Platform: platform, HTTP: d.HTTP, BaseURL: d.URL, UserAgent: source.UserAgentWithSite("")}
}

// wrap names the failures the UI treats specially: a rejected token and a rate limit.
func wrap[T any](v T, err error) (T, error) {
	if gomodio.Denied(err) {
		return v, ErrBadToken
	}
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return v, &source.BusyError{Source: "mod.io", Reset: rl.Reset}
	}
	return v, err
}

// client builds the library client for the stored token, on the account's own host.
func (d Driver) client(ctx context.Context) (*gomodio.Client, error) {
	token, err := d.token()
	if err != nil {
		return nil, err
	}
	cfg := d.config(token)
	if d.URL == "" {
		id, ok := userIDs.Load(token)
		if !ok {
			me, err := wrap(gomodio.New(cfg).Me(ctx))
			if err != nil {
				return nil, err
			}
			id = me.ID
			userIDs.Store(token, id)
		}
		cfg.UserID, _ = id.(int)
	}
	return gomodio.New(cfg), nil
}

// game returns the client and the numeric game id for key, which names a path, so nothing else may reach it.
func (d Driver) game(ctx context.Context, key string) (*gomodio.Client, int, error) {
	if !gameKey.MatchString(key) {
		return nil, 0, fmt.Errorf("%q is not a mod.io game id", key)
	}
	c, err := d.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	id, _ := strconv.Atoi(key)
	return c, id, nil
}

func (d Driver) item(m gomodio.Mod) source.Item {
	it := source.Item{
		Source: d.ID(), ID: strconv.Itoa(m.ID), Name: m.Name, Summary: m.Summary, Author: m.SubmittedBy.Username,
		Picture: m.Logo.Thumb, Endorsements: m.Stats.RatingsPositive, Downloads: m.Stats.Downloads,
		// A mod's page is named by the game's and the mod's slugs, which only the API's answer gives.
		Size: m.Modfile.Filesize, URL: source.WebLink(m.ProfileURL), Adult: m.Explicit(),
	}
	if m.Modfile.ID != 0 {
		it.Version = m.Modfile.VersionName()
	}
	if m.DateUpdated > 0 {
		it.Updated = time.Unix(m.DateUpdated, 0).UTC().Format(time.RFC3339)
	}
	return it
}

// Search lists the game's live mods matching the text, filtered by tag; the game version is Mortar's own, so it is
// not a filter.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	c, gid, err := d.game(ctx, q.Key)
	if err != nil {
		return source.Page{}, err
	}
	list, err := wrap(c.Search(ctx, gomodio.SearchQuery{
		GameID: gid, Text: strings.TrimSpace(q.Text), Sort: sortField(q.Sort), TagsIn: q.Categories,
		TagsNotIn: q.ExcludeCategories, Limit: q.Len(), Offset: max(q.Page-source.FirstPage, 0) * q.Len(),
	}))
	if err != nil {
		return source.Page{}, err
	}
	items := make([]source.Item, 0, len(list.Data))
	for _, m := range list.Data {
		items = append(items, d.item(m))
	}
	return source.Page{Total: list.Total, Items: items}, nil
}

// Categories lists the game's visible tag names.
func (d Driver) Categories(ctx context.Context, key string) ([]string, error) {
	c, gid, err := d.game(ctx, key)
	if err != nil {
		return nil, err
	}
	names, err := wrap(c.Tags(ctx, gid))
	return source.UniqueNames(names), err
}

func modID(id string) (int, error) {
	n, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("%q is not a mod.io mod id", id)
	}
	return n, nil
}

// Details reads the mod's description, tags, its files' versions newest first, the newest file's changelog and the
// names of the mods it depends on. mod.io dependencies include credit links, so they are only listed.
func (d Driver) Details(ctx context.Context, key, id, _ string) (source.Details, error) {
	c, gid, err := d.game(ctx, key)
	if err != nil {
		return source.Details{}, err
	}
	mid, err := modID(id)
	if err != nil {
		return source.Details{}, err
	}
	m, err := wrap(c.Mod(ctx, gid, mid))
	if err != nil {
		return source.Details{}, err
	}
	files, err := wrap(c.Files(ctx, gid, mid))
	if err != nil {
		return source.Details{}, err
	}
	deps, err := wrap(c.Dependencies(ctx, gid, mid, false))
	if err != nil {
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
		out.Versions = append(out.Versions, f.VersionName())
	}
	if len(files) > 0 {
		out.Changelog = strings.TrimSpace(files[0].Changelog)
	}
	for _, dep := range deps {
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
	c, gid, err := d.game(ctx, key)
	if err != nil {
		return Resolved{}, err
	}
	mid, err := modID(id)
	if err != nil {
		return Resolved{}, err
	}
	m, err := wrap(c.Mod(ctx, gid, mid))
	if err != nil {
		return Resolved{}, err
	}
	f := m.Modfile
	if version != "" && version != strconv.Itoa(f.ID) && version != f.VersionName() {
		files, err := wrap(c.Files(ctx, gid, mid))
		if err != nil {
			return Resolved{}, err
		}
		i := slices.IndexFunc(files, func(f gomodio.File) bool { return version == strconv.Itoa(f.ID) || version == f.VersionName() })
		if i < 0 {
			return Resolved{}, fmt.Errorf("mod.io mod %s has no file %q", id, version)
		}
		f = files[i]
	}
	if !f.Downloadable() {
		return Resolved{}, fmt.Errorf("mod.io mod %s has no file to download", id)
	}
	// The pinned go-modio accepts an http address and a missing digest, either of which would install unchecked.
	if !strings.HasPrefix(strings.ToLower(f.Download.BinaryURL), "https://") || f.MD5() == "" {
		return Resolved{}, fmt.Errorf("mod.io mod %s has no https download address and an MD5 for the file", id)
	}
	return Resolved{
		ModID: id, Name: m.Name, Version: f.VersionName(), FileName: f.Filename, FileID: f.ID, Size: f.Filesize,
		URL: f.Download.BinaryURL, MD5: f.MD5(),
	}, nil
}

// Latest reads the current file of every installed mod, a hundred mods a request. mod.io has no release channels, so
// a mod whose current file's MD5 differs from the installed one has an update.
func (d Driver) Latest(ctx context.Context, src components.GameSource, _ string, files []source.InstalledFile) (map[string]source.Latest, error) {
	out := map[string]source.Latest{}
	c, gid, err := d.game(ctx, src.Key)
	if err != nil {
		return nil, err
	}
	for chunk := range slices.Chunk(files, gomodio.MaxPageSize) {
		ids := make([]int, 0, len(chunk))
		for _, f := range chunk {
			if n, err := strconv.Atoi(f.ID); err == nil {
				ids = append(ids, n)
			}
		}
		list, err := wrap(c.Search(ctx, gomodio.SearchQuery{GameID: gid, IDs: ids}))
		if err != nil {
			return nil, err
		}
		for _, f := range chunk {
			i := slices.IndexFunc(list.Data, func(m gomodio.Mod) bool { return strconv.Itoa(m.ID) == f.ID })
			if i < 0 {
				continue
			}
			m := list.Data[i]
			if m.Modfile.ID == 0 || "md5:"+m.Modfile.MD5() == strings.ToLower(f.Digest) {
				continue
			}
			out[f.Digest] = source.Latest{
				ProjectID: f.ID, Version: m.Modfile.VersionName(), VersionID: strconv.Itoa(m.Modfile.ID), URL: source.WebLink(m.ProfileURL),
			}
		}
	}
	return out, nil
}

// Validate checks token against the API and returns the account's username.
func (d Driver) Validate(ctx context.Context, token string) (string, error) {
	if apiKeyShape.MatchString(strings.TrimSpace(token)) {
		return "", ErrAPIKey
	}
	me, err := wrap(gomodio.New(d.config(strings.TrimSpace(token))).Me(ctx))
	if err != nil {
		return "", err
	}
	return cmp.Or(me.Username, "mod.io"), nil
}
