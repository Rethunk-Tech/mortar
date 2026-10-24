package components

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// KnownBrokenURL is where each game's signed known-broken list is published (Rethunk-Tech/mortar-known-broken, the
// rolling release "lists"): <game>.json and <game>.json.sig.
const KnownBrokenURL = "https://github.com/Rethunk-Tech/mortar-known-broken/releases/download/lists/"

const (
	maxKnownBroken = 4 << 20
	knownBrokenTTL = 24 * time.Hour
)

// BrokenIDs names one mod on each site it is on; one is enough.
type BrokenIDs struct {
	Thunderstore string `json:"thunderstore,omitempty"`
	SMAPI        string `json:"smapi,omitempty"`
	Nexus        int    `json:"nexus,omitempty"`
	CurseForge   int    `json:"curseforge,omitempty"`
	GitHub       string `json:"github,omitempty"`
}

// BrokenEntry is a mod known not to work: Through and every older version are broken, a newer one is taken to be
// fixed. GameVersions, when set, limits it to those game versions (constraints such as ">=50", spaces between).
type BrokenEntry struct {
	IDs          BrokenIDs `json:"ids"`
	Through      string    `json:"through"`
	GameVersions string    `json:"gameVersions,omitempty"`
	Reason       string    `json:"reason"`
	// Replacement is a package ("Namespace-Name") to use instead, optional.
	Replacement string `json:"replacement,omitempty"`
}

// BrokenList is one game's signed known-broken list.
type BrokenList struct {
	Game    string        `json:"game"`
	Serial  uint64        `json:"serial"`
	Entries []BrokenEntry `json:"entries"`
}

// BrokenMod is what a mod is known by, as a match against a list needs it.
type BrokenMod struct {
	ID         mod.ID
	NexusID    int
	CurseForge int
	GitHub     string
	Version    string
}

// Names reports whether the entry is about m, by any id they share.
func (e BrokenEntry) Names(m BrokenMod) bool {
	local := m.ID.Local()
	switch m.ID.Format() {
	case mod.FormatThunderstore:
		if e.IDs.Thunderstore != "" && strings.EqualFold(e.IDs.Thunderstore, local) {
			return true
		}
	case mod.FormatSMAPI:
		if e.IDs.SMAPI != "" && strings.EqualFold(e.IDs.SMAPI, local) {
			return true
		}
	}
	return (e.IDs.Nexus > 0 && e.IDs.Nexus == m.NexusID) ||
		(e.IDs.CurseForge > 0 && e.IDs.CurseForge == m.CurseForge) ||
		(e.IDs.GitHub != "" && strings.EqualFold(e.IDs.GitHub, m.GitHub))
}

// Breaks reports whether the entry flags version on gameVersion: the version is Through or older (a version that
// cannot be compared is not flagged), and the game version meets GameVersions (an unknown game version does).
func (e BrokenEntry) Breaks(version, gameVersion string) bool {
	c, ok := compareLoose(version, e.Through)
	if !ok || c > 0 {
		return false
	}
	return e.GameVersions == "" || gameVersion == "" || inConstraints(gameVersion, e.GameVersions)
}

// Find is the entry that flags m on gameVersion, if any.
func (l BrokenList) Find(m BrokenMod, gameVersion string) (BrokenEntry, bool) {
	for _, e := range l.Entries {
		if e.Names(m) && e.Breaks(m.Version, gameVersion) {
			return e, true
		}
	}
	return BrokenEntry{}, false
}

var digits = regexp.MustCompile(`[0-9]+`)

// compareLoose orders two versions by their runs of digits in turn ("49" < "50", "v72" > "v50", "1.2" = "1.2.0"); a
// version with no digits cannot be ordered. Games and mod sites write versions too loosely for semver.
func compareLoose(a, b string) (int, bool) {
	x, y := digits.FindAllString(a, -1), digits.FindAllString(b, -1)
	if len(x) == 0 || len(y) == 0 {
		return 0, false
	}
	for i := range max(len(x), len(y)) {
		var p, q int
		if i < len(x) {
			p, _ = strconv.Atoi(x[i])
		}
		if i < len(y) {
			q, _ = strconv.Atoi(y[i])
		}
		if p != q {
			return cmp.Compare(p, q), true
		}
	}
	return 0, true
}

func inConstraints(version, constraints string) bool {
	for c := range strings.FieldsSeq(constraints) {
		op := c[:1]
		if len(c) > 1 && c[1] == '=' {
			op = c[:2]
		}
		got, ok := compareLoose(version, strings.TrimPrefix(c, op))
		if !ok {
			return true
		}
		switch op {
		case "<":
			ok = got < 0
		case "<=":
			ok = got <= 0
		case ">":
			ok = got > 0
		case ">=":
			ok = got >= 0
		case "=":
			ok = got == 0
		}
		if !ok {
			return false
		}
	}
	return true
}

// DecodeBrokenList parses and checks a list for game.
func DecodeBrokenList(body []byte, game string) (BrokenList, error) {
	var l BrokenList
	if err := json.Unmarshal(body, &l); err != nil {
		return BrokenList{}, err
	}
	if l.Game != game || l.Serial == 0 {
		return BrokenList{}, fmt.Errorf("known-broken list for %q has game %q, serial %d", game, l.Game, l.Serial)
	}
	for i, e := range l.Entries {
		if e.Through == "" || e.Reason == "" || e.IDs == (BrokenIDs{}) {
			return BrokenList{}, fmt.Errorf("known-broken entry %d needs ids, through and a reason", i)
		}
		for c := range strings.FieldsSeq(e.GameVersions) {
			if !versionConstraint.MatchString(c) {
				return BrokenList{}, fmt.Errorf("known-broken entry %d: game version constraint %q", i, c)
			}
		}
	}
	return l, nil
}

type cachedBrokenList struct {
	List      []byte `json:"list"`
	Signature []byte `json:"signature"`
}

// LoadKnownBroken fetches game's signed list, cached a day; a list that fails its signature or is older than the one
// cached is refused, and the cached one serves while a fetch fails. A game nobody published a list for has none.
func (c *Client) LoadKnownBroken(ctx context.Context, cache *meta.Client, publicKey []byte, game string) (BrokenList, error) {
	name := meta.KnownBrokenPrefix + game + ".json"
	trusted := func(e cachedBrokenList) (BrokenList, bool) {
		if Verify(e.List, e.Signature, publicKey) != nil {
			return BrokenList{}, false
		}
		l, err := DecodeBrokenList(e.List, game)
		return l, err == nil
	}
	var oldSerial uint64
	if old, ok := meta.Peek[cachedBrokenList](cache, name); ok {
		if l, ok := trusted(old); ok {
			oldSerial = l.Serial
		}
	}
	entry, err := meta.CachedForBuild(cache, meta.KnownBrokenPrefix+game+".json", knownBrokenTTL, func() (cachedBrokenList, error) {
		url := cmp.Or(c.KnownBrokenURL, KnownBrokenURL) + game + ".json"
		list, err := c.get(ctx, url, maxKnownBroken)
		if err != nil {
			return cachedBrokenList{}, err
		}
		sig, err := c.get(ctx, url+".sig", maxSignature)
		if err != nil {
			return cachedBrokenList{}, err
		}
		e := cachedBrokenList{List: list, Signature: sig}
		l, ok := trusted(e)
		if !ok {
			return cachedBrokenList{}, fmt.Errorf("%w: %s", ErrBadSignature, url)
		}
		if l.Serial < oldSerial {
			return cachedBrokenList{}, fmt.Errorf("%w: %s serial %d is older than %d", ErrRollback, url, l.Serial, oldSerial)
		}
		return e, nil
	})
	if err != nil {
		return BrokenList{}, err
	}
	l, ok := trusted(entry)
	if !ok {
		return BrokenList{}, errors.New("cached known-broken list does not verify")
	}
	return l, nil
}

var brokenSource struct {
	mu     sync.Mutex
	client *Client
	cache  *meta.Client
	key    []byte
	lists  map[string]BrokenList
	at     map[string]time.Time
}

// UseKnownBroken sets where KnownBrokenFor reads lists from: the catalog's client, cache and key.
func UseKnownBroken(c *Client, cache *meta.Client, publicKey []byte) {
	brokenSource.mu.Lock()
	defer brokenSource.mu.Unlock()
	brokenSource.client, brokenSource.cache, brokenSource.key = c, cache, publicKey
	brokenSource.lists, brokenSource.at = map[string]BrokenList{}, map[string]time.Time{}
}

// KnownBrokenFor is game's known-broken list, empty when none is published, none could be fetched yet, or no source
// is set. Lists are read once an hour; the cache file is checked against a day.
// ponytail: one lock over every game's first fetch, so a slow fetch for one game holds the others; per-game locks if
// that is ever seen.
func KnownBrokenFor(ctx context.Context, game string) BrokenList {
	brokenSource.mu.Lock()
	defer brokenSource.mu.Unlock()
	if brokenSource.client == nil {
		return BrokenList{}
	}
	if l, ok := brokenSource.lists[game]; ok && time.Since(brokenSource.at[game]) < time.Hour {
		return l
	}
	l, _ := brokenSource.client.LoadKnownBroken(ctx, brokenSource.cache, brokenSource.key, game)
	brokenSource.lists[game], brokenSource.at[game] = l, time.Now()
	return l
}
