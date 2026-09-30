package problems

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/meta"
)

// updatesTTL bounds how long a profile's updates are served without asking again; meta.Client caches for the
// same hour, so a shorter one would only repeat its answers.
const updatesTTL = time.Hour

// sourceSMAPI and sourceMortar mark the bundled mods (SMAPI's own and Mortar's bridge), which update with
// their owner and never show in the mod list.
const (
	sourceSMAPI  = "smapi"
	sourceMortar = "mortar"
)

// Update is a newer version SMAPI's API suggests for an installed mod. URL is the page to get it from.
type Update struct {
	Key       string `json:"key"`
	UniqueID  string `json:"uniqueId"`
	Name      string `json:"name"`
	Installed string `json:"installed"`
	Version   string `json:"version"`
	URL       string `json:"url"`
}

// UpdatesResult lists a profile's updates. Unknown is set when SMAPI's API could not be reached for some mod,
// so the list may be short.
type UpdatesResult struct {
	Updates []Update `json:"updates"`
	Unknown bool     `json:"unknown"`
}

// CheckUpdates asks SMAPI's API about every user mod (the bundled ones update with SMAPI). It never returns
// an error: a failed lookup leaves Unknown set.
func CheckUpdates(ctx context.Context, m Meta, env Environment, mods []Installed) UpdatesResult {
	r := UpdatesResult{Updates: []Update{}}
	req := meta.UpdateRequest{APIVersion: env.APIVersion, GameVersion: env.GameVersion, Platform: env.Platform}
	var asked []Installed
	for _, x := range mods {
		if x.SourceKind == sourceSMAPI || x.SourceKind == sourceMortar {
			continue
		}
		asked = append(asked, x)
		req.Mods = append(req.Mods, meta.InstalledMod{ID: x.UniqueID, UpdateKeys: x.UpdateKeys, Version: x.Version})
	}
	if len(asked) == 0 {
		return r
	}
	for i, res := range m.CheckUpdates(ctx, req) {
		switch {
		case !res.Known:
			r.Unknown = true
		case res.Suggested != nil:
			x := asked[i]
			r.Updates = append(r.Updates, Update{
				Key: x.Key, UniqueID: x.UniqueID, Name: x.Name,
				Installed: x.Version, Version: res.Suggested.Version, URL: res.Suggested.URL,
			})
		}
	}
	return r
}

// Need is one dependency of a mod and whether the profile meets it. State is "ok", "absent", "disabled" or
// "outdated"; Name is the installed mod's name, or the UniqueID when there is none.
type Need struct {
	UniqueID         string `json:"uniqueId"`
	Name             string `json:"name"`
	MinimumVersion   string `json:"minimumVersion"`
	Required         bool   `json:"required"`
	State            string `json:"state"`
	InstalledVersion string `json:"installedVersion"`
}

// Dependent is a mod of the profile that lists another as a dependency.
type Dependent struct {
	Key      string `json:"key"`
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
}

// Relations is how one mod stands with the others in its profile. PageURL is empty when its update keys name
// no page Mortar knows.
type Relations struct {
	PageURL  string      `json:"pageUrl"`
	Needs    []Need      `json:"needs"`
	NeededBy []Dependent `json:"neededBy"`
}

// Relate reports what the mod key/uniqueID needs and which mods need it. ok is false when the profile lacks it.
func Relate(mods []Installed, key, uniqueID string) (r Relations, ok bool) {
	i := slices.IndexFunc(mods, func(x Installed) bool { return x.Key == key && sameID(x.UniqueID, uniqueID) })
	if i < 0 {
		return Relations{}, false
	}
	self := mods[i]
	r = Relations{PageURL: pageURL(self.UpdateKeys), Needs: []Need{}, NeededBy: []Dependent{}}
	for _, dep := range self.Dependencies {
		n := Need{UniqueID: dep.UniqueID, Name: dep.UniqueID, MinimumVersion: dep.MinimumVersion, Required: dep.Required, State: "ok"}
		if j := slices.IndexFunc(mods, func(x Installed) bool { return sameID(x.UniqueID, dep.UniqueID) }); j >= 0 {
			n.Name = mods[j].Name
		}
		if reason, have := depState(mods, dep); reason != "" {
			n.State, n.InstalledVersion = reason, have
		}
		r.Needs = append(r.Needs, n)
	}
	for _, x := range mods {
		if x.Key == self.Key && sameID(x.UniqueID, self.UniqueID) {
			continue
		}
		if slices.ContainsFunc(x.Dependencies, func(d manifest.Dependency) bool { return sameID(d.UniqueID, self.UniqueID) }) {
			r.NeededBy = append(r.NeededBy, Dependent{Key: x.Key, UniqueID: x.UniqueID, Name: x.Name})
		}
	}
	return r, true
}

// Pages maps "key/uniqueId" to the page of each mod whose update keys name one.
func Pages(mods []Installed) map[string]string {
	out := map[string]string{}
	for _, m := range mods {
		if u := pageURL(m.UpdateKeys); u != "" {
			out[m.Key+"/"+m.UniqueID] = u
		}
	}
	return out
}

// pageURL is the page of the first update key that names one: a Nexus mod or a GitHub repository.
func pageURL(keys []string) string {
	for _, k := range keys {
		if n, ok := nexusKey(k); ok {
			return siteURL(meta.Ref{Site: "Nexus", ID: n})
		}
		site, rest, ok := strings.Cut(k, ":")
		if ok && strings.EqualFold(strings.TrimSpace(site), "github") && strings.Count(rest, "/") == 1 {
			return "https://github.com/" + strings.TrimSpace(rest)
		}
	}
	return ""
}
