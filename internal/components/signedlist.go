package components

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

const (
	maxSignedList = 4 << 20
	signedListTTL = 24 * time.Hour
)

// signedList is a kind of per-game list published apart from the catalog and signed with its key: <game>.json and
// <game>.json.sig under url, cached under prefix. decode checks a list for a game and gives its serial.
type signedList[T any] struct {
	prefix string
	url    string
	decode func(body []byte, game string) (T, uint64, error)
}

func (l signedList[T]) listName(game string) string { return l.prefix + game + ".json" }

// floorName is the file holding the highest serial accepted for game, kept apart from the cache entry so a cleared or
// expired cache cannot reopen the way to an older signed list.
func (l signedList[T]) floorName(game string) string { return "serial-floor-" + l.listName(game) }

type cachedSignedList struct {
	List      []byte `json:"list"`
	Signature []byte `json:"signature"`
}

// load fetches game's list from base (l.url when empty), cached a day; a list that fails its signature or is older
// than the one cached is refused, and the cached one serves while a fetch fails.
func (l signedList[T]) load(ctx context.Context, c *Client, cache *meta.Client, publicKey []byte, base, game string) (T, error) {
	var zero T
	trusted := func(e cachedSignedList) (T, uint64, bool) {
		if Verify(e.List, e.Signature, publicKey) != nil {
			return zero, 0, false
		}
		v, serial, err := l.decode(e.List, game)
		return v, serial, err == nil
	}
	floor, _ := meta.Peek[uint64](cache, l.floorName(game))
	oldSerial := floor
	if old, ok := meta.Peek[cachedSignedList](cache, l.listName(game)); ok {
		if _, serial, ok := trusted(old); ok {
			oldSerial = max(oldSerial, serial)
		}
	}
	entry, err := meta.CachedForBuild(cache, l.listName(game), signedListTTL, func() (cachedSignedList, error) {
		url := base + game + ".json"
		if base == "" {
			url = l.url + game + ".json"
		}
		list, err := c.get(ctx, url, maxSignedList)
		if err != nil {
			return cachedSignedList{}, err
		}
		sig, err := c.get(ctx, url+".sig", maxSignature)
		if err != nil {
			return cachedSignedList{}, err
		}
		e := cachedSignedList{List: list, Signature: sig}
		_, serial, ok := trusted(e)
		if !ok {
			return cachedSignedList{}, fmt.Errorf("%w: %s", ErrBadSignature, url)
		}
		if serial < oldSerial {
			return cachedSignedList{}, fmt.Errorf("%w: %s serial %d is older than %d", ErrRollback, url, serial, oldSerial)
		}
		if serial > floor {
			meta.Put(cache, l.floorName(game), serial)
		}
		return e, nil
	})
	if err != nil {
		return zero, err
	}
	v, serial, ok := trusted(entry)
	if !ok {
		return zero, errors.New("cached signed list does not verify")
	}
	if serial < floor {
		return zero, fmt.Errorf("%w: %s serial %d is older than %d", ErrRollback, l.listName(game), serial, floor)
	}
	return v, nil
}

var listSource struct {
	mu     sync.Mutex
	client *Client
	cache  *meta.Client
	key    []byte
}

// UseSignedLists sets where the per-game signed lists (known-broken mods, log suppressions) are read from: the
// catalog's client, cache and key.
func UseSignedLists(c *Client, cache *meta.Client, publicKey []byte) {
	listSource.mu.Lock()
	defer listSource.mu.Unlock()
	listSource.client, listSource.cache, listSource.key = c, cache, publicKey
	brokenLists.reset()
	suppressionLists.reset()
}

// listMemo holds each game's list for an hour, so callers on every screen do not each read the cache file.
// ponytail: one lock over every game's first fetch, so a slow fetch for one game holds the others; per-game locks if
// that is ever seen.
type listMemo[T any] struct {
	mu    sync.Mutex
	lists map[string]T
	at    map[string]time.Time
}

func (m *listMemo[T]) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lists, m.at = map[string]T{}, map[string]time.Time{}
}

func (m *listMemo[T]) get(ctx context.Context, l signedList[T], game string) T {
	listSource.mu.Lock()
	c, cache, key := listSource.client, listSource.cache, listSource.key
	listSource.mu.Unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	var zero T
	if c == nil {
		return zero
	}
	if v, ok := m.lists[game]; ok && time.Since(m.at[game]) < time.Hour {
		return v
	}
	v, err := l.load(ctx, c, cache, key, "", game)
	if err != nil && ctx.Err() != nil {
		// A cancelled caller says nothing about the list: the next one fetches again.
		return v
	}
	m.lists[game], m.at[game] = v, time.Now()
	return v
}
