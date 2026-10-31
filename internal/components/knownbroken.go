package components

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-semverish"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// KnownBrokenURL is where each game's signed known-broken list is published (Rethunk-Tech/mortar-known-broken, the
// rolling release "lists"): <game>.json and <game>.json.sig.
const KnownBrokenURL = "https://github.com/Rethunk-Tech/mortar-known-broken/releases/download/lists/"

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
	// Kind is BrokenKindBroken (the default), BrokenKindSuperseded or BrokenKindNotRecommended.
	Kind   string `json:"kind,omitempty"`
	Reason string `json:"reason"`
	// Replacement is a package ("Namespace-Name") to use instead, optional.
	Replacement string `json:"replacement,omitempty"`
	// DropIn, on a superseded entry with a Replacement, says the replacement does everything the mod did for the
	// packs that need it, so it meets their dependency on the mod.
	DropIn bool `json:"dropIn,omitempty"`
}

// Kinds of known-broken entry: it does not work, another mod does its job now, or it works but is best avoided.
const (
	BrokenKindBroken         = "broken"
	BrokenKindSuperseded     = "superseded"
	BrokenKindNotRecommended = "notRecommended"
)

// Status is the entry's kind, BrokenKindBroken when it names none.
func (e BrokenEntry) Status() string { return cmp.Or(e.Kind, BrokenKindBroken) }

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

// DropIns maps each Thunderstore package a drop-in entry names, lowercased "Namespace-Name", to its replacement.
func (l BrokenList) DropIns() map[string]string {
	out := map[string]string{}
	for _, e := range l.Entries {
		if e.DropIn && e.Status() == BrokenKindSuperseded && e.Replacement != "" && e.IDs.Thunderstore != "" {
			out[strings.ToLower(e.IDs.Thunderstore)] = e.Replacement
		}
	}
	return out
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

// compareLoose orders two versions as semver when both parse, so a prerelease sorts below its release and build
// metadata is ignored; otherwise by their runs of digits in turn ("49" < "50", "v72" > "v50"), since games and mod
// sites write versions too loosely for semver. A version with no digits cannot be ordered.
func compareLoose(a, b string) (int, bool) {
	if c, ok := semverish.Compare(a, b); ok {
		return c, true
	}
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
		if !slices.Contains([]string{BrokenKindBroken, BrokenKindSuperseded, BrokenKindNotRecommended}, e.Status()) {
			return BrokenList{}, fmt.Errorf("known-broken entry %d has an unknown kind %q", i, e.Kind)
		}
		if e.DropIn && (e.Status() != BrokenKindSuperseded || e.Replacement == "" || e.IDs.Thunderstore == "") {
			return BrokenList{}, fmt.Errorf("known-broken entry %d: dropIn needs kind superseded, a thunderstore id and a replacement", i)
		}
		for c := range strings.FieldsSeq(e.GameVersions) {
			if !versionConstraint.MatchString(c) {
				return BrokenList{}, fmt.Errorf("known-broken entry %d: game version constraint %q", i, c)
			}
		}
	}
	return l, nil
}

var knownBroken = signedList[BrokenList]{
	prefix: meta.KnownBrokenPrefix,
	url:    KnownBrokenURL,
	decode: func(body []byte, game string) (BrokenList, uint64, error) {
		l, err := DecodeBrokenList(body, game)
		return l, l.Serial, err
	},
}

var brokenLists listMemo[BrokenList]

// LoadKnownBroken fetches game's signed list, cached a day; a list that fails its signature or is older than the one
// cached is refused, and the cached one serves while a fetch fails. A game nobody published a list for has none.
func (c *Client) LoadKnownBroken(ctx context.Context, cache *meta.Client, publicKey []byte, game string) (BrokenList, error) {
	return knownBroken.load(ctx, c, cache, publicKey, c.KnownBrokenURL, game)
}

// KnownBrokenFor is game's known-broken list, empty when none is published, none could be fetched yet, or no source
// is set. Lists are read once an hour; the cache file is checked against a day.
func KnownBrokenFor(ctx context.Context, game string) BrokenList {
	return brokenLists.get(ctx, knownBroken, game)
}
