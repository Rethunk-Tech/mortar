// Package itch is the itch.io source driver, over the server-side API with the user's own API key. No catalog game
// lists it yet, so it is registered but no game uses it. The key lives in the OS keyring under "itch"; without one
// the driver reports ErrNeedsKey and browse disables it. A game's source key is searched as a keyword beside the
// player's text, since the API has no tag filter.
package itch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-itchio"

	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const (
	keyName = "itch"
	siteURL = "https://itch.io"
)

// BaseURL is the server-side API root.
const BaseURL = itchio.BaseURL

// ErrNeedsKey means no itch.io API key is stored.
var ErrNeedsKey = itchio.ErrNeedsKey

// ErrBadKey means itch.io rejected the key.
var ErrBadKey = itchio.ErrBadKey

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

// Modes lists both ways: with the player's API key Mortar downloads uploads itself, and without one the player opens
// a pasted game page, saves the file there and adds it.
func (Driver) Modes() []source.Acquire { return []source.Acquire{source.Download, source.Handoff} }

// Hosts is the site's web host.
func (Driver) Hosts() []string { return []string{"itch.io"} }

// ModPageURL is the game page; id is a page name as ParsePageURL returns it, or the numeric game id of a search
// result, which the site redirects to the page, so no slug is needed. gameKey does not enter the address.
func (Driver) ModPageURL(_, id string) string {
	if strings.Contains(id, "/") {
		return PageURL(id)
	}
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

func (d Driver) client(key string) *itchio.Client {
	return itchio.New(itchio.Config{Key: key, BaseURL: d.URL, HTTP: d.HTTP})
}

// mapErr keeps HTTP failures in Mortar's source.StatusError so the player still sees "itch.io" on the message.
func mapErr(err error) error {
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return &source.StatusError{Service: "itch.io", Code: se.Code, Status: se.Status}
	}
	return err
}

// Unavailable says why browse cannot use the source now, or "" when it can.
func (d Driver) Unavailable() string {
	if _, err := d.key(); errors.Is(err, ErrNeedsKey) {
		return ErrNeedsKey.Error()
	}
	return ""
}

// Search lists games matching the key and text. The API answers every hit at once, ignoring a page number, and has
// no sort, so Sort is ignored and a later page is empty.
func (d Driver) Search(ctx context.Context, q source.Query) (source.Page, error) {
	// A game that lists itch.io with no key is there for pasted page links only, so there is nothing to search.
	if q.Key == "" || q.Page > source.FirstPage {
		return source.Page{}, nil
	}
	key, err := d.key()
	if err != nil {
		return source.Page{}, err
	}
	games, err := d.client(key).SearchGames(ctx, q.Key+" "+q.Text, q.Page)
	if err != nil {
		return source.Page{}, mapErr(err)
	}
	items := make([]source.Item, 0, len(games))
	for _, g := range games {
		items = append(items, source.Item{
			Source: d.ID(), ID: strconv.Itoa(g.ID), Name: g.Title, Summary: g.ShortText, Author: g.User.Username,
			Picture: g.CoverURL, Updated: g.PublishedAt, URL: g.URL,
		})
	}
	return source.Page{Total: len(items), Items: items}, nil
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
	c := d.client(key)
	uploads, err := c.Uploads(ctx, gameID)
	if err != nil {
		return Resolved{}, mapErr(err)
	}
	for _, u := range uploads {
		id := strconv.Itoa(u.ID)
		switch {
		case upload != "":
			if id != upload && u.Filename != upload {
				continue
			}
		case u.Demo || u.Type == "soundtrack":
			continue
		}
		link, err := c.DownloadURL(ctx, id)
		if err != nil {
			return Resolved{}, mapErr(err)
		}
		return Resolved{GameID: gameID, UploadID: id, FileName: u.Filename, Size: u.Size, URL: link}, nil
	}
	return Resolved{}, fmt.Errorf("game %s has no matching upload on itch.io", gameID)
}

// Validate checks key against the API and returns the account's username.
func (d Driver) Validate(ctx context.Context, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", ErrBadKey
	}
	me, err := d.client(key).Me(ctx)
	if err != nil {
		return "", mapErr(err)
	}
	if me.Username == "" {
		return "itch.io", nil
	}
	return me.Username, nil
}
