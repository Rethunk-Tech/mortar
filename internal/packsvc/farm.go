package packsvc

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// farmGame is the game Share's mod list text is read for, and the one whose mods SMAPI tells apart by unique id.
const farmGame = "stardew"

// maxFarmBytes bounds a pasted list; a profile of a thousand mods is far smaller.
const (
	maxFarmBytes = 1 << 20
	maxFarmMods  = 1000
)

// States of a FarmRow.
const (
	FarmMissing   = "missing"
	FarmOff       = "off"
	FarmDifferent = "different"
)

// FarmMod is a mod the host runs: the id the game's loader reports to peers (SMAPI's unique id, a Thunderstore
// package's Namespace-Name, a Vintage Story mod id), its version, and where it came from.
type FarmMod struct {
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	// Source is empty for a mod that has no download source (a local archive), which a guest installs by hand.
	Source profile.Source `json:"source"`
}

// FarmList is what a guest needs to join a host's game. SMAPI sends every loaded mod's id, name and version to its
// peers and enforces no match itself (SMultiplayer.GetContextSyncMessageFields); many mods refuse a peer on another
// version, and the manifest has no client-only flag. So the list holds every enabled mod and content pack but SMAPI's
// bundled ones, and a guest ignores a mod of its own that the list lacks.
type FarmList struct {
	Game string    `json:"game"`
	Name string    `json:"name"`
	Mods []FarmMod `json:"mods"`
}

// FarmRow is one mod the guest must still bring in line. Mine is the guest's version, empty when it has none.
type FarmRow struct {
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Mine    string `json:"mine"`
	State   string `json:"state"`
	Fixable bool   `json:"fixable"`
}

// FarmCheck is the list compared with a profile; Rows is empty when the profile already matches.
type FarmCheck struct {
	Host string    `json:"host"`
	Rows []FarmRow `json:"rows"`
}

// FarmFix is what FixFarm queued; Manual names the mods with no source to download from.
type FarmFix struct {
	Queued int      `json:"queued"`
	Manual []string `json:"manual"`
}

// farmSupported is whether the game's mods can be matched exactly across computers: Stardew by SMAPI unique id, a
// Thunderstore game by package id, Vintage Story by the mod id its modinfo.json gives.
func farmSupported(gameID string) bool {
	if gameID == farmGame {
		return true
	}
	i := slices.IndexFunc(game.Catalog(), func(g components.GameInfo) bool { return g.ID == gameID })
	if i < 0 {
		return false
	}
	g := game.Catalog()[i]
	_, thunderstore := g.Source(profile.KindThunderstore)
	_, vintage := g.Source(profile.KindVSModDB)
	return thunderstore || vintage
}

func errNoFarm() error {
	return usererr.New(usererr.OtherGame, "mod lists are not offered for this game")
}

// farmID is the id a mod is matched by across computers. A Vintage Story mod is its modinfo.json id (its component
// id names a store item, which differs per computer); a mod with no id of its own is matched by name.
func farmID(c profile.Component) mod.ID {
	switch {
	case c.ModID != "":
		return mod.NewID(mod.FormatModInfo, c.ModID)
	case c.ID.Format() == mod.FormatFolder:
		return mod.NewID(formatName, strings.ToLower(c.Name))
	}
	return c.ID
}

// sameFarmID compares ids as the games do: Thunderstore package ids and mod ids ignore case, as SMAPI's do.
func sameFarmID(a, b mod.ID) bool {
	if a.Format() == b.Format() && (a.Format() == mod.FormatThunderstore || a.Format() == mod.FormatModInfo) {
		return strings.EqualFold(a.Local(), b.Local())
	}
	return mod.Equal(a, b)
}

// ExportFarm lists the mods of a profile that a guest's must match.
func (s *Service) ExportFarm(gameID, profileID string) (FarmList, error) {
	if !farmSupported(gameID) {
		return FarmList{}, errNoFarm()
	}
	p, err := s.find(gameID, profileID)
	if err != nil {
		return FarmList{}, err
	}
	out := FarmList{Game: gameID, Name: p.Name, Mods: []FarmMod{}}
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		for _, c := range e.Mods {
			if e.Enabled(c.ID) {
				out.Mods = append(out.Mods, FarmMod{ID: farmID(c), Name: c.Name, Version: c.Version, Source: shareable(e.Source)})
			}
		}
	}
	return out, nil
}

// shareable keeps the fields that name a download and drops the rest (pictures, counts, digests).
func shareable(s profile.Source) profile.Source {
	return profile.Source{Kind: s.Kind, Name: s.Name, ModID: s.ModID, FileID: s.FileID, Version: s.Version, Repo: s.Repo, Tag: s.Tag, Asset: s.Asset}
}

// formatName marks the id of a mod read from Share's mod list text, which carries names and no ids.
const formatName = "name"

// farmNexusDomain is the game's name in a Nexus Mods page address.
const farmNexusDomain = "stardewvalley"

var (
	listLink   = regexp.MustCompile(`^\[((?:\\.|[^\\\]])*)\]\((\S*?)\)(?: (\S+))?$`)
	listBare   = regexp.MustCompile(`^(.*?)(?: (\d\S*))?$`)
	listPlain  = regexp.MustCompile(`^(.+?)(?: v(\S+))?(?: - (https?://\S+))?$`)
	listEscape = regexp.MustCompile("\\\\([\\\\`*_\\[\\]()])")
	nexusPage  = regexp.MustCompile(`^https://www\.nexusmods\.com/([^/]+)/mods/(\d+)`)
	githubPage = regexp.MustCompile(`^https://github\.com/([^/\s]+/[^/\s]+)$`)
)

// listLine reads one line of Share's mod list text in its Markdown (also Discord) or plain form. A plain line
// counts only with a version or an address, which is what tells a mod from a heading.
func listLine(line string) (name, version, page string, ok bool) {
	if md, found := strings.CutPrefix(line, "- "); found {
		m := listLink.FindStringSubmatch(md)
		if m == nil {
			m = append(listBare.FindStringSubmatch(md), "")
			m[2], m[3] = "", m[2]
		}
		return listEscape.ReplaceAllString(m[1], "$1"), m[3], m[2], m[1] != ""
	}
	m := listPlain.FindStringSubmatch(line)
	if m == nil {
		return "", "", "", false
	}
	return m[1], m[2], m[3], m[2] != "" || m[3] != ""
}

// parseModList reads the enabled mods of Share's mod list text. A mod listed on a Nexus Mods or GitHub page can be
// downloaded from it; a list of pages for another Nexus game is that game's.
func parseModList(text string) FarmList {
	l := FarmList{Game: farmGame, Mods: []FarmMod{}}
	seen := map[mod.ID]bool{}
	off := false
	for line := range strings.Lines(text) {
		line = strings.TrimSpace(line)
		name, version, page, ok := listLine(line)
		// ponytail: the headings are matched in English, the one language Mortar ships; with a second locale,
		// treat the second heading as the switched-off group.
		if !ok {
			off = off && !strings.EqualFold(line, "Enabled") || strings.EqualFold(line, "Disabled")
			continue
		}
		m := FarmMod{ID: mod.NewID(formatName, strings.ToLower(name)), Name: name, Version: version}
		if n := nexusPage.FindStringSubmatch(page); n != nil {
			if n[1] != farmNexusDomain {
				l.Game = n[1]
			}
			id, _ := strconv.Atoi(n[2])
			m.Source = profile.Source{Kind: profile.KindNexus, ModID: id}
		} else if g := githubPage.FindStringSubmatch(page); g != nil {
			m.Source = profile.Source{Kind: profile.KindGitHub, Repo: g[1]}
		}
		if !off && !seen[m.ID] && len(l.Mods) <= maxFarmMods {
			seen[m.ID] = true
			l.Mods = append(l.Mods, m)
		}
	}
	return l
}

// ParseFarm reads a list a host copied out, either the multiplayer list or Share's mod list text (Stardew's).
func ParseFarm(text string) (FarmList, error) {
	if len(text) > maxFarmBytes {
		return FarmList{}, usererr.New(usererr.Invalid, "the list is too large")
	}
	var l FarmList
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &l); err != nil {
		l = parseModList(text)
	}
	if len(l.Mods) == 0 || len(l.Mods) > maxFarmMods {
		return FarmList{}, usererr.New(usererr.Invalid, "this is not a mod list")
	}
	return l, nil
}

// parseFarmFor is ParseFarm for a guest of gameID, refusing a list made for another game.
func parseFarmFor(gameID, text string) (FarmList, error) {
	if !farmSupported(gameID) {
		return FarmList{}, errNoFarm()
	}
	l, err := ParseFarm(text)
	if err != nil {
		return FarmList{}, err
	}
	if l.Game != gameID {
		return FarmList{}, usererr.New(usererr.OtherGame, "this list is for another game")
	}
	// A list names its own download sources; one the game's catalog does not offer is dropped, leaving the mod to
	// install by hand.
	if i := slices.IndexFunc(game.Catalog(), func(g components.GameInfo) bool { return g.ID == gameID }); i >= 0 {
		for j := range l.Mods {
			if _, ok := game.Catalog()[i].Source(l.Mods[j].Source.Kind); !ok {
				l.Mods[j].Source = profile.Source{}
			}
		}
	}
	return l, nil
}

// CheckFarm compares the host's list with the profile and returns the mods that are missing, switched off or on
// another version.
func (s *Service) CheckFarm(gameID, profileID, text string) (FarmCheck, error) {
	l, err := parseFarmFor(gameID, text)
	if err != nil {
		return FarmCheck{}, err
	}
	p, err := s.find(gameID, profileID)
	if err != nil {
		return FarmCheck{}, err
	}
	return FarmCheck{Host: l.Name, Rows: farmRows(l, p)}, nil
}

func farmRows(l FarmList, p profile.Profile) []FarmRow {
	rows := []FarmRow{}
	for _, m := range l.Mods {
		e, c, ok := findComponent(p, m)
		row := FarmRow{ID: m.ID, Name: m.Name, Host: m.Version, Fixable: installable(m.Source)}
		switch {
		case !ok:
			row.State = FarmMissing
		case !sameVersion(c.Version, m.Version):
			row.State, row.Mine = FarmDifferent, c.Version
		case !e.Enabled(c.ID):
			row.State, row.Mine, row.Fixable = FarmOff, c.Version, false
		default:
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

// findComponent finds the profile's mod with the host's id. A host's mod known only by name is the mod of that
// name, the one on the host's version when several share it.
func findComponent(p profile.Profile, m FarmMod) (profile.Entry, profile.Component, bool) {
	if m.ID.Format() == formatName {
		var (
			entry profile.Entry
			named profile.Component
			found bool
		)
		for _, e := range p.Entries {
			for _, c := range e.Mods {
				if strings.EqualFold(strings.TrimSpace(c.Name), m.Name) && (!found || sameVersion(c.Version, m.Version)) {
					entry, named, found = e, c, true
				}
			}
		}
		return entry, named, found
	}
	for _, e := range p.Entries {
		if i := slices.IndexFunc(e.Mods, func(c profile.Component) bool { return sameFarmID(farmID(c), m.ID) }); i >= 0 {
			return e, e.Mods[i], true
		}
	}
	return profile.Entry{}, profile.Component{}, false
}

func sameVersion(a, b string) bool {
	if c, ok := meta.CompareVersions(a, b); ok {
		return c == 0
	}
	return a == b
}

func installable(s profile.Source) bool {
	_, ok := restoreRequest("", "", profile.Entry{Source: s})
	return ok
}

// FixFarm queues the host's file of each missing mod and each mod on another version, all of them when ids is empty,
// through the same download queue an import uses; a different version replaces the guest's entry, as an update does.
func (s *Service) FixFarm(ctx context.Context, gameID, profileID, text string, ids []mod.ID) (FarmFix, error) {
	l, err := parseFarmFor(gameID, text)
	if err != nil {
		return FarmFix{}, err
	}
	p, err := s.find(gameID, profileID)
	if err != nil {
		return FarmFix{}, err
	}
	res := FarmFix{Manual: []string{}}
	var reqs []queue.Request
	// Several mods of one download are fixed by one request.
	queued := map[profile.Source]bool{}
	for _, row := range farmRows(l, p) {
		if row.State == FarmOff || (len(ids) > 0 && !slices.ContainsFunc(ids, func(id mod.ID) bool { return sameFarmID(id, row.ID) })) {
			continue
		}
		i := slices.IndexFunc(l.Mods, func(m FarmMod) bool { return sameFarmID(m.ID, row.ID) })
		src := shareable(l.Mods[i].Source)
		req, ok := restoreRequest(gameID, profileID, profile.Entry{Source: src})
		if queued[src] {
			continue
		}
		if !ok {
			res.Manual = append(res.Manual, row.Name)
			continue
		}
		if row.State == FarmDifferent {
			e, _, _ := findComponent(p, l.Mods[i])
			// The host's version is the one to reach, though the guest's may be newer.
			req.Kind, req.CurrentKey, req.Exact = queue.KindUpdate, e.Key, true
		}
		queued[src] = true
		reqs = append(reqs, req)
	}
	if len(reqs) == 0 {
		return res, nil
	}
	items, err := s.Queue.Add(ctx, reqs)
	res.Queued = len(items)
	return res, err
}
