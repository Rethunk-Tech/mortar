package problems

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

// updatesTTL bounds how long a profile's updates are served without asking again; meta.Client caches for the
// same hour, so a shorter one would only repeat its answers.
const updatesTTL = time.Hour

var semverPrerelease = regexp.MustCompile(`^[vV]?(\d+)\.(\d+)(?:\.(\d+))?(?:\.(\d+))?-([0-9A-Za-z].*)$`)

func hasPrerelease(version string) bool {
	return semverPrerelease.MatchString(strings.TrimSpace(version))
}

// Update is a newer version SMAPI's API suggests for an installed mod. URL is the page to get it from, and
// NexusID that page's Nexus mod ID, 0 when the suggested version is not on Nexus (a CurseForge beta of a mod
// that is also on Nexus must not be fetched from Nexus). GitHubRepo is "owner/repo" when the mod's
// update key and the suggested update both name a GitHub repository, so the release can be installed directly.
type Update struct {
	Key        string `json:"key"`
	UniqueID   string `json:"uniqueId"`
	Name       string `json:"name"`
	Installed  string `json:"installed"`
	Version    string `json:"version"`
	URL        string `json:"url"`
	NexusID    int    `json:"nexusId"`
	GitHubRepo string `json:"githubRepo"`
	Unofficial bool   `json:"unofficial"`
}

// UpdatesResult lists a profile's updates. Unknown is set when SMAPI's API could not be reached for some mod,
// so the list may be short.
type UpdatesResult struct {
	Updates []Update `json:"updates"`
	Unknown bool     `json:"unknown"`
}

// CheckUpdates asks SMAPI's API about every user mod (the bundled ones update with SMAPI). It never returns
// an error: a failed lookup leaves Unknown set.
func CheckUpdates(ctx context.Context, m Meta, env Environment, mods []Installed, enabledOnly bool) UpdatesResult {
	r := UpdatesResult{Updates: []Update{}}
	req := meta.UpdateRequest{APIVersion: env.APIVersion, GameVersion: env.GameVersion, Platform: env.Platform}
	var asked []Installed
	for _, x := range mods {
		if x.SourceKind == profile.SourceSMAPI || x.SourceKind == profile.SourceMortar {
			continue
		}
		if enabledOnly && !x.Enabled {
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
				NexusID: nexusUpdate(x.UpdateKeys, res.Suggested.URL), GitHubRepo: githubUpdate(x.UpdateKeys, res.Suggested.URL),
			})
			if res.Unofficial != nil {
				r.Updates = append(r.Updates, Update{
					Key: x.Key, UniqueID: x.UniqueID, Name: x.Name,
					Installed: x.Version, Version: res.Unofficial.Version, URL: res.Unofficial.URL,
					NexusID: nexusUpdate(x.UpdateKeys, res.Unofficial.URL), GitHubRepo: githubUpdate(x.UpdateKeys, res.Unofficial.URL),
					Unofficial: true,
				})
			}
		case res.Unofficial != nil:
			x := asked[i]
			r.Updates = append(r.Updates, Update{
				Key: x.Key, UniqueID: x.UniqueID, Name: x.Name,
				Installed: x.Version, Version: res.Unofficial.Version, URL: res.Unofficial.URL,
				NexusID: nexusUpdate(x.UpdateKeys, res.Unofficial.URL), GitHubRepo: githubUpdate(x.UpdateKeys, res.Unofficial.URL),
				Unofficial: true,
			})
		}
	}
	return r
}

// HideHeld drops updates the profile has pinned or skipped for that exact newer version, and prerelease
// versions when includePrerelease is false unless the installed version is itself a prerelease.
func HideHeld(r UpdatesResult, mods []Installed, includePrerelease bool) UpdatesResult {
	byKey := make(map[string]Installed, len(mods))
	for _, m := range mods {
		byKey[m.Key] = m
	}
	kept := make([]Update, 0, len(r.Updates))
	for _, u := range r.Updates {
		m, ok := byKey[u.Key]
		if !ok {
			if !includePrerelease && hasPrerelease(u.Version) && !hasPrerelease(u.Installed) {
				continue
			}
			kept = append(kept, u)
			continue
		}
		hold := profile.Entry{Pinned: m.Pinned, SkipVersion: m.SkipVersion, IgnoreUpdates: m.IgnoreUpdates}
		if !hold.OffersUpdate(u.Version) {
			continue
		}
		if !includePrerelease && hasPrerelease(u.Version) && !hasPrerelease(u.Installed) {
			continue
		}
		kept = append(kept, u)
	}
	r.Updates = kept
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
		if repo, ok := githubKey(k); ok {
			return "https://github.com/" + repo
		}
	}
	return ""
}

// nexusID is the Nexus mod of the first update key that names one, or 0.
func nexusID(keys []string) int {
	for _, k := range keys {
		if n, ok := nexusKey(k); ok {
			return n
		}
	}
	return 0
}

// githubKey is the "owner/repo" of a "GitHub:owner/repo" update key.
func githubKey(key string) (string, bool) {
	site, rest, ok := strings.Cut(key, ":")
	rest = strings.TrimSpace(rest)
	return rest, ok && strings.EqualFold(strings.TrimSpace(site), "github") && strings.Count(rest, "/") == 1
}

// githubUpdate is the repository of the first GitHub update key, provided the suggested update lives on GitHub too.
func nexusUpdate(keys []string, url string) int {
	u := strings.ToLower(url)
	if u != "" && !strings.Contains(u, "nexusmods.com/") && !strings.HasPrefix(u, "https://github.com/") {
		return 0
	}
	return nexusID(keys)
}

func githubUpdate(keys []string, url string) string {
	if !strings.HasPrefix(strings.ToLower(url), "https://github.com/") {
		return ""
	}
	for _, k := range keys {
		if repo, ok := githubKey(k); ok {
			return repo
		}
	}
	return ""
}
