// Package itch is the itch.io source driver, over the server-side API with the user's own API key. No catalog game
// lists it yet, so it is registered but no game uses it. The key lives in the OS keyring under "itch"; without one
// the driver reports ErrNeedsKey and browse disables it. A game's source key is searched as a keyword beside the
// player's text, since the API has no tag filter.
package itch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// BaseURL is the real API root; the key rides in an Authorization header and never in an address, so an error
// that quotes the address cannot leak it.
const BaseURL = "https://itch.io/api/1/x"

const (
	keyName = "itch"
	siteURL = "https://itch.io"
)

// ErrNeedsKey means no itch.io API key is stored.
var ErrNeedsKey = errors.New("needs an itch.io API key")

// ErrBadKey means itch.io rejected the key.
var ErrBadKey = errors.New("itch.io rejected the API key")

// Driver searches itch.io. HTTP, URL and Key are for tests; the zero value talks to the real API with the
// keyring's key.
type Driver struct {
	HTTP *http.Client
	URL  string
	// Key returns the API key; nil reads the keyring.
	Key func() (string, error)
}

var _ = source.Register(Driver{})

// ID is the catalog id.
func (Driver) ID() string { return "itch" }

// Name is the site's name.
func (Driver) Name() string { return "itch.io" }

// Modes says Mortar downloads uploads itself.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Download} }

// Hosts is the site's web host.
func (Driver) Hosts() []string { return []string{"itch.io"} }

// ModPageURL is the game page; id is the numeric game id, which the site redirects to the page, so no slug is
// needed. gameKey does not enter the address.
func (Driver) ModPageURL(_, id string) string {
	if id == "" {
		return ""
	}
	return siteURL + "/games/" + url.PathEscape(id)
}

func (d Driver) key() (string, error) {
	get := d.Key
	if get == nil {
		get = func() (string, error) { return secret.Get(keyName) }
	}
	k, err := get()
	if errors.Is(err, secret.ErrNotFound) || (err == nil && strings.TrimSpace(k) == "") {
		return "", ErrNeedsKey
	}
	return strings.TrimSpace(k), err
}

// Unavailable says why browse cannot use the source now, or "" when it can.
func (d Driver) Unavailable() string {
	if _, err := d.key(); errors.Is(err, ErrNeedsKey) {
		return ErrNeedsKey.Error()
	}
	return ""
}

// call GETs an API path with the key, which itch.io may answer with HTTP 200 and an errors list.
func (d Driver) call(ctx context.Context, key, path string, params url.Values, out any) error {
	base := d.URL
	if base == "" {
		base = BaseURL
	}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("User-Agent", source.UserAgent(""))
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests {
		return &source.BusyError{Source: "itch.io"}
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return ErrBadKey
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, source.MaxBody))
	if err != nil {
		return err
	}
	var env struct {
		Errors []string `json:"errors"`
	}
	if json.Unmarshal(body, &env) == nil && len(env.Errors) > 0 {
		if slices.Contains(env.Errors, "invalid key") {
			return ErrBadKey
		}
		return fmt.Errorf("itch.io: %s", strings.Join(env.Errors, "; "))
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("itch.io answered %s", resp.Status)
	}
	return json.Unmarshal(body, out)
}

type game struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ShortText   string `json:"short_text"`
	URL         string `json:"url"`
	CoverURL    string `json:"cover_url"`
	PublishedAt string `json:"published_at"`
	User        struct {
		Username string `json:"username"`
	} `json:"user"`
}

// Search lists games matching the key and text. The API has no sort or total, so Sort is ignored and Total counts
// only what is known: this page, plus a full page's worth more so browse offers a next page.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	key, err := d.key()
	if err != nil {
		return source.Page{}, err
	}
	params := url.Values{}
	params.Set("query", strings.TrimSpace(q.Key+" "+q.Text))
	params.Set("page", strconv.Itoa(max(q.Page, source.FirstPage)))
	var parsed struct {
		Games []game `json:"games"`
	}
	if err := d.call(ctx, key, "/search/games", params, &parsed); err != nil {
		return source.Page{}, err
	}
	items := make([]source.Item, 0, len(parsed.Games))
	for _, g := range parsed.Games {
		items = append(items, source.Item{
			Source: d.ID(), ID: strconv.Itoa(g.ID), Name: g.Title, Summary: g.ShortText, Author: g.User.Username,
			Picture: g.CoverURL, Updated: g.PublishedAt, URL: g.URL,
		})
	}
	total := (max(q.Page, source.FirstPage)-source.FirstPage)*source.PageSize + len(items)
	if len(items) >= source.PageSize {
		total += source.PageSize
	}
	return source.Page{Total: total, Items: items}, nil
}

// Resolved is what an install needs of one upload.
type Resolved struct {
	GameID   string
	UploadID string
	FileName string
	Size     int64
	// URL is the time-limited download address itch.io issued for this call.
	URL string
}

// Resolve picks an upload of the game, by id or filename, else the first one, and asks itch.io for its download
// address. Uploads flagged as demos or soundtracks are not mods and are passed over when choosing the first.
func (d Driver) Resolve(ctx context.Context, gameID, upload string) (Resolved, error) {
	key, err := d.key()
	if err != nil {
		return Resolved{}, err
	}
	var ups struct {
		Uploads []struct {
			ID       int    `json:"id"`
			Filename string `json:"filename"`
			Size     int64  `json:"size"`
			Type     string `json:"type"`
			Demo     bool   `json:"demo"`
		} `json:"uploads"`
	}
	if err := d.call(ctx, key, "/game/"+url.PathEscape(gameID)+"/uploads", nil, &ups); err != nil {
		return Resolved{}, err
	}
	for _, u := range ups.Uploads {
		id := strconv.Itoa(u.ID)
		switch {
		case upload != "":
			if id != upload && u.Filename != upload {
				continue
			}
		case u.Demo || u.Type == "soundtrack":
			continue
		}
		var dl struct {
			URL string `json:"url"`
		}
		if err := d.call(ctx, key, "/upload/"+id+"/download", nil, &dl); err != nil {
			return Resolved{}, err
		}
		return Resolved{GameID: gameID, UploadID: id, FileName: u.Filename, Size: u.Size, URL: dl.URL}, nil
	}
	return Resolved{}, fmt.Errorf("game %s has no matching upload on itch.io", gameID)
}

// Validate checks key against the API and returns the account's username.
func (d Driver) Validate(ctx context.Context, key string) (string, error) {
	var me struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := d.call(ctx, strings.TrimSpace(key), "/me", nil, &me); err != nil {
		return "", err
	}
	if me.User.Username == "" {
		return "itch.io", nil
	}
	return me.User.Username, nil
}
