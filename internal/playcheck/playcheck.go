// Package playcheck lists what the player should know before Play, for the window's "Before you play" dialog and
// `mortar play --check`.
package playcheck

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
)

// NameCap is how many names a group lists; Count carries the full total.
const NameCap = 5

// Group is one warning bucket. Kind is missing, conflicts, updates, broken, longPaths, lastProfile, changes, saveMods
// or saveModsOff. The window words NameParts, Changes and the Save fields in the user's language, so they come
// structured beside the English Names the CLI prints.
type Group struct {
	Kind  string   `json:"kind"`
	Count int      `json:"count"`
	Names []string `json:"names"`
	// NameTitles is parallel to Names where a name stands in for a longer one, such as "Unknown mod" for a bare id.
	NameTitles []string `json:"nameTitles,omitempty"`
	// NameParts is parallel to Names for conflicts: the mods each one names, empty when the name is the asset target.
	NameParts [][]string `json:"nameParts,omitempty"`
	// Changes are the first NameCap profile changes of a changes group.
	Changes []profile.HistoryItem `json:"changes,omitempty"`
	// SaveFolder, SaveFarm and SaveName identify the save of a lastProfile group.
	SaveFolder      string `json:"saveFolder,omitempty"`
	SaveFarm        string `json:"saveFarm,omitempty"`
	SaveName        string `json:"saveName,omitempty"`
	ProfileName     string `json:"profileName,omitempty"`
	SwitchProfileID string `json:"switchProfileId,omitempty"`
}

// Input is everything the groups are drawn from.
type Input struct {
	ProfileID string
	Profile   profile.Profile
	Problems  problems.Result
	Updates   problems.UpdatesResult
	// SmapiNever hides SMAPI prerelease updates.
	SmapiNever bool
	// Installed names the profile's mods, for a requirement that is installed but disabled or outdated.
	Installed []profile.Mod
	// Fit is the newest save, zero when there is none.
	Fit savessvc.Fit
	// ProfileName is a profile's name by id, empty when it is gone.
	ProfileName func(id string) string
	// Changes is the profile's changes since its last run, nil when it has none or they could not be read.
	Changes *profile.HistoryDiff
}

// Groups lists the warnings in the order the dialog shows them.
func Groups(in Input) []Group {
	var out []Group
	add := func(g Group) {
		if g.Count > 0 {
			out = append(out, g)
		}
	}
	add(missingGroup(in.Problems.Missing, in.Installed))
	add(conflictsGroup(in.Problems.AssetConflicts))
	add(listGroup("updates", updateNames(in)))
	add(listGroup("broken", brokenNames(in.Problems.Broken)))
	add(listGroup("longPaths", longPathNames(in.Problems.LongPaths)))
	out = append(out, saveGroups(in)...)
	if in.Changes != nil {
		add(changesGroup(*in.Changes))
	}
	return out
}

func listGroup(kind string, labels []string) Group {
	return Group{Kind: kind, Count: len(labels), Names: labels[:min(len(labels), NameCap)]}
}

func missingGroup(rows []problems.Missing, installed []profile.Mod) Group {
	g := Group{Kind: "missing"}
	seen := map[string]bool{}
	titled := false
	var titles []string
	for _, m := range rows {
		// A row exists per dependent mod, and an optional, outside or unorderable requirement is a note.
		if !m.Counted() || seen[m.ID.Fold()] {
			continue
		}
		seen[m.ID.Fold()] = true
		g.Count++
		if len(g.Names) >= NameCap {
			continue
		}
		name, title := missingName(m, installed)
		g.Names = append(g.Names, name)
		titles = append(titles, title)
		titled = titled || title != ""
	}
	if titled {
		g.NameTitles = titles
	}
	return g
}

// missingName is an installed copy's name first (a disabled or outdated one), then the page that offers it, the id
// only last, which reads "Unknown mod" with the id as its title.
func missingName(m problems.Missing, installed []profile.Mod) (name, title string) {
	page := ""
	if m.Where != nil {
		page = strings.TrimSpace(m.Where.PageName)
	}
	known := m.ID.Local()
	for _, i := range installed {
		if i.ID.Fold() == m.ID.Fold() {
			known = i.Name
			break
		}
	}
	name = known
	if page != "" && (m.Listed || known == m.ID.Local()) {
		name = page
	}
	if page == "" && name == m.ID.Local() {
		return "Unknown mod", name
	}
	return name, ""
}

func conflictsGroup(conflicts []framework.AssetConflict) Group {
	g := Group{Kind: "conflicts"}
	var parts [][]string
	for _, c := range conflicts {
		if c.Cosmetic {
			continue
		}
		g.Count++
		if len(g.Names) >= NameCap {
			continue
		}
		names := slices.DeleteFunc(slices.Clone(c.Names), func(n string) bool { return n == "" })
		g.Names = append(g.Names, cmp.Or(strings.Join(names, ", "), c.Target))
		parts = append(parts, names)
	}
	if slices.ContainsFunc(parts, func(p []string) bool { return len(p) > 0 }) {
		g.NameParts = parts
	}
	return g
}

func updateNames(in Input) []string {
	var out []string
	for _, u := range in.Updates.Updates {
		if (in.SmapiNever && u.Unofficial) || u.Version == "" {
			continue
		}
		if e, ok := entryByKey(in.Profile, u.Key); ok && (e.Pinned || (u.Source != "" && slices.Contains(e.SkipSources, u.Source)) || (e.SkipVersion != "" && e.SkipVersion == u.Version)) {
			continue
		}
		out = append(out, u.Name)
	}
	return out
}

func entryByKey(p profile.Profile, key string) (profile.Entry, bool) {
	for _, e := range p.Entries {
		if e.Key == key {
			return e, true
		}
	}
	return profile.Entry{}, false
}

func brokenNames(rows []problems.Broken) []string {
	var out []string
	for _, b := range rows {
		if b.Status == "broken" || b.Status == "obsolete" || b.Status == "cycle" {
			out = append(out, b.Name)
		}
	}
	return out
}

func longPathNames(rows []problems.LongPath) []string {
	var out []string
	for _, l := range rows {
		out = append(out, l.Name)
	}
	return out
}

// saveGroups are the newest save's warnings: the profile that last played it when that is another, then the mods it
// used that this profile lacks and those it has switched off, which need different fixes.
func saveGroups(in Input) []Group {
	fit := in.Fit
	var out []Group
	if fit.LastProfileID != "" && fit.LastProfileID != in.ProfileID {
		out = append(out, Group{
			Kind: "lastProfile", Count: 1, Names: []string{},
			SaveFolder: fit.Folder, SaveFarm: fit.Farm, SaveName: fit.Name,
			ProfileName: cmp.Or(in.ProfileName(fit.LastProfileID), fit.LastProfileID), SwitchProfileID: fit.LastProfileID,
		})
	}
	switchTo := ""
	if fit.LastProfileExists {
		switchTo = fit.LastProfileID
	}
	for _, kind := range []struct {
		name     string
		disabled bool
	}{{"saveMods", false}, {"saveModsOff", true}} {
		g := Group{Kind: kind.name, Names: []string{}, SwitchProfileID: switchTo}
		for _, l := range fit.Missing {
			if l.Disabled != kind.disabled {
				continue
			}
			g.Count++
			if len(g.Names) < NameCap {
				g.Names = append(g.Names, cmp.Or(l.Name, l.ID.Local()))
			}
		}
		if g.Count > 0 {
			out = append(out, g)
		}
	}
	return out
}

func changesGroup(diff profile.HistoryDiff) Group {
	g := Group{Kind: "changes", Count: len(diff.Items), Names: []string{}}
	for _, it := range diff.Items {
		if it.Detail != "" && len(g.Names) < NameCap {
			g.Names = append(g.Names, it.Detail)
		}
	}
	g.Changes = diff.Items[:min(len(diff.Items), NameCap)]
	return g
}
