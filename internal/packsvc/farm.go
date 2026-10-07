package packsvc

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
)

// farmGame is the game whose multiplayer the list serves.
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

// FarmMod is a mod the host's farm runs: the id SMAPI reports to peers, its version, and where it came from.
type FarmMod struct {
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	// Source is empty for a mod that has no download source (a local archive), which a guest installs by hand.
	Source profile.Source `json:"source"`
}

// FarmList is what a guest needs to join a host's farm. SMAPI sends every loaded mod's id, name and version to its
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

// ExportFarm lists the mods of a Stardew profile that a guest's must match.
func (s *Service) ExportFarm(gameID, profileID string) (FarmList, error) {
	if gameID != farmGame {
		return FarmList{}, errors.New("multiplayer lists are for Stardew Valley")
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
				out.Mods = append(out.Mods, FarmMod{ID: c.ID, Name: c.Name, Version: c.Version, Source: shareable(e.Source)})
			}
		}
	}
	return out, nil
}

// shareable keeps the fields that name a download and drops the rest (pictures, counts, digests).
func shareable(s profile.Source) profile.Source {
	return profile.Source{Kind: s.Kind, Name: s.Name, ModID: s.ModID, FileID: s.FileID, Version: s.Version, Repo: s.Repo, Tag: s.Tag, Asset: s.Asset}
}

// ParseFarm reads a list a host copied out, refusing one for another game.
func ParseFarm(text string) (FarmList, error) {
	if len(text) > maxFarmBytes {
		return FarmList{}, errors.New("the list is too large")
	}
	var l FarmList
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &l); err != nil || len(l.Mods) == 0 || len(l.Mods) > maxFarmMods {
		return FarmList{}, errors.New("this is not a farm list")
	}
	if l.Game != farmGame {
		return FarmList{}, errors.New("multiplayer lists are for Stardew Valley")
	}
	return l, nil
}

// CheckFarm compares the host's list with the profile and returns the mods that are missing, switched off or on
// another version.
func (s *Service) CheckFarm(gameID, profileID, text string) (FarmCheck, error) {
	l, err := ParseFarm(text)
	if err != nil {
		return FarmCheck{}, err
	}
	if gameID != farmGame {
		return FarmCheck{}, errors.New("multiplayer lists are for Stardew Valley")
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
		e, c, ok := findComponent(p, m.ID)
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

func findComponent(p profile.Profile, id mod.ID) (profile.Entry, profile.Component, bool) {
	for _, e := range p.Entries {
		if i := slices.IndexFunc(e.Mods, func(c profile.Component) bool { return mod.Equal(c.ID, id) }); i >= 0 {
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
	l, err := ParseFarm(text)
	if err != nil {
		return FarmFix{}, err
	}
	if gameID != farmGame {
		return FarmFix{}, errors.New("multiplayer lists are for Stardew Valley")
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
		if row.State == FarmOff || (len(ids) > 0 && !slices.ContainsFunc(ids, func(id mod.ID) bool { return mod.Equal(id, row.ID) })) {
			continue
		}
		i := slices.IndexFunc(l.Mods, func(m FarmMod) bool { return mod.Equal(m.ID, row.ID) })
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
			e, _, _ := findComponent(p, row.ID)
			req.Kind, req.CurrentKey = queue.KindUpdate, e.Key
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
