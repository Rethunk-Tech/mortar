package problems

import (
	"cmp"
	"context"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// updatesTTL bounds how long a profile's updates are served without asking again; meta.Client caches for the
// same hour, so a shorter one would only repeat its answers.
const updatesTTL = time.Hour

var semverPrerelease = regexp.MustCompile(`^[vV]?(\d+)\.(\d+)(?:\.(\d+))?(?:\.(\d+))?-([0-9A-Za-z].*)$`)

// hasPrerelease reports a semver prerelease suffix. Authors also tag releases "-stable" or "-release", which
// semver would read as a prerelease and so let betas through to someone who turned them off.
func hasPrerelease(version string) bool {
	m := semverPrerelease.FindStringSubmatch(strings.TrimSpace(version))
	if m == nil {
		return false
	}
	tag, _, _ := strings.Cut(strings.ToLower(m[5]), ".")
	switch tag {
	case "stable", "release", "final", "ga", "rtm":
		return false
	}
	return true
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
	// GitHubFallback is the mod's own GitHub repo when the update points elsewhere (usually Nexus); the queue uses
	// it only when Nexus would need a click and the same version is released there.
	GitHubFallback string `json:"githubFallback,omitempty"`
	Unofficial     bool   `json:"unofficial"`
	Source         string `json:"source"`
}

// UpdatesResult lists a profile's updates. Unknown is set when SMAPI's API could not be reached for some mod,
// so the list may be short.
type UpdatesResult struct {
	Updates []Update `json:"updates"`
	// Held lists versions SMAPI's API suggests that Mortar does not offer, with the reason, so the Console can say
	// why SMAPI's "You can update" list differs from Mortar's.
	Held    []Held `json:"held"`
	Unknown bool   `json:"unknown"`
}

// Held reasons.
const (
	// HeldCurrent: the download installed is already that version; only a manifest inside it was not bumped.
	HeldCurrent = "current"
	// HeldSkipped, HeldPinned, HeldIgnored and HeldSource are the profile's own choices for the mod.
	HeldSkipped = "skipped"
	HeldPinned  = "pinned"
	HeldIgnored = "ignored"
	HeldSource  = "source"
	// HeldPrerelease: a prerelease while prereleases are off; HeldUnofficial: an unofficial SMAPI build while those are off.
	HeldPrerelease = "prerelease"
	HeldUnofficial = "unofficial"
)

// Held is one suggested version Mortar holds back. Have is the version of the download installed.
type Held struct {
	Key      string `json:"key"`
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Have     string `json:"have"`
	Reason   string `json:"reason"`
}

// NexusFilesOf lists many Nexus mods' current files in one call (nexus.Client.FilesOf); nil when signed out.
type NexusFilesOf func(ctx context.Context, t nexus.Title, modIDs []int) (map[int][]nexus.BatchFile, error)

// checkUpdates asks SMAPI's API about every user mod (the bundled ones update with SMAPI). It never returns an
// error: a failed lookup leaves Unknown set.
func checkUpdates(ctx context.Context, m Meta, env Environment, mods []Installed, enabledOnly, fresh bool, filesOf NexusFilesOf) UpdatesResult {
	r := UpdatesResult{Updates: []Update{}, Held: []Held{}}
	req := meta.UpdateRequest{APIVersion: env.APIVersion, GameVersion: env.GameVersion, Platform: env.Platform, Fresh: fresh}
	var asked []Installed
	for _, x := range mods {
		if (profile.Source{Kind: x.SourceKind}).Bundled() {
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
	results := m.CheckUpdates(ctx, req)
	live := liveNexusFiles(ctx, filesOf, env.Nexus, asked, results)
	current := func(x Installed, url, version string) bool {
		if files, ok := live[nexusUpdate(x.UpdateKeys, url)]; ok {
			if is, known := liveFileIsCurrent(files, x, version); known {
				return is
			}
		}
		return nexusFileIsCurrent(ctx, m, x, url, version)
	}
	for i, res := range results {
		if res.Suggested != nil && (downloaded(asked[i], res.Suggested.Version) ||
			current(asked[i], res.Suggested.URL, res.Suggested.Version)) {
			x := asked[i]
			r.Held = append(r.Held, Held{
				Key: x.Key, UniqueID: x.UniqueID, Name: x.Name, Version: res.Suggested.Version,
				Have: cmp.Or(x.SourceVersion, x.Version), Reason: HeldCurrent,
			})
			res.Suggested = nil
		}
		if res.Unofficial != nil && (downloaded(asked[i], res.Unofficial.Version) ||
			current(asked[i], res.Unofficial.URL, res.Unofficial.Version)) {
			res.Unofficial = nil
		}
		if !res.Known {
			r.Unknown = true
			continue
		}
		x := asked[i]
		if res.Suggested != nil {
			r.Updates = append(r.Updates, Update{
				Key: x.Key, UniqueID: x.UniqueID, Name: x.Name,
				Installed: x.Version, Version: res.Suggested.Version, URL: res.Suggested.URL,
				NexusID: nexusUpdate(x.UpdateKeys, res.Suggested.URL), GitHubRepo: githubUpdate(x.UpdateKeys, res.Suggested.URL),
				GitHubFallback: cmp.Or(githubFallback(x.UpdateKeys, res.Suggested.URL), metadataFallback(res.GitHubRepo, res.Suggested.URL)),
				Source:         updateSource(*res.Suggested, nexusUpdate(x.UpdateKeys, res.Suggested.URL), githubUpdate(x.UpdateKeys, res.Suggested.URL)),
			})
		}
		if res.Unofficial != nil {
			r.Updates = append(r.Updates, Update{
				Key: x.Key, UniqueID: x.UniqueID, Name: x.Name,
				Installed: x.Version, Version: res.Unofficial.Version, URL: res.Unofficial.URL,
				NexusID: nexusUpdate(x.UpdateKeys, res.Unofficial.URL), GitHubRepo: githubUpdate(x.UpdateKeys, res.Unofficial.URL),
				Source:     updateSource(*res.Unofficial, nexusUpdate(x.UpdateKeys, res.Unofficial.URL), githubUpdate(x.UpdateKeys, res.Unofficial.URL)),
				Unofficial: true,
			})
		}
	}
	applyChannelFileOffers(ctx, m, env.Nexus.Domain, asked, &r)
	return r
}

// HideHeld drops updates the profile has pinned or skipped for that exact newer version, and prerelease
// versions when includePrerelease is false unless the installed version is itself a prerelease.
func HideHeld(r UpdatesResult, mods []Installed, includePrerelease bool, smapiBuilds string) UpdatesResult {
	byKey := make(map[string]Installed, len(mods))
	for _, m := range mods {
		byKey[m.Key] = m
	}
	kept := make([]Update, 0, len(r.Updates))
	held := slices.Clone(r.Held)
	hold := func(u Update, reason string) {
		held = append(held, Held{Key: u.Key, UniqueID: u.UniqueID, Name: u.Name, Version: u.Version, Have: u.Installed, Reason: reason})
	}
	for _, u := range r.Updates {
		if smapiBuilds == "never" && u.Unofficial {
			hold(u, HeldUnofficial)
			continue
		}
		m, ok := byKey[u.Key]
		if !ok {
			if !keepPrerelease(Installed{}, includePrerelease, u.Version, u.Installed) {
				hold(u, HeldPrerelease)
				continue
			}
			kept = append(kept, u)
			continue
		}
		entry := profile.Entry{Pinned: m.Pinned, SkipVersion: m.SkipVersion, IgnoreUpdates: m.IgnoreUpdates}
		switch {
		case slices.Contains(m.SkipSources, u.Source):
			hold(u, HeldSource)
		case !entry.OffersUpdate(u.Version):
			hold(u, heldChoice(entry))
		case !keepPrerelease(m, includePrerelease, u.Version, u.Installed):
			hold(u, HeldPrerelease)
		default:
			kept = append(kept, u)
		}
	}
	r.Updates, r.Held = kept, held
	return r
}

// heldChoice names the profile choice that made OffersUpdate refuse a version.
func heldChoice(e profile.Entry) string {
	switch {
	case e.IgnoreUpdates:
		return HeldIgnored
	case e.Pinned:
		return HeldPinned
	default:
		return HeldSkipped
	}
}

func updateSource(u meta.Update, nexusID int, githubRepo string) string {
	if site, _, ok := strings.Cut(u.Source, ":"); ok && strings.TrimSpace(site) != "" {
		return strings.TrimSpace(site)
	}
	if githubRepo != "" {
		return "GitHub"
	}
	if nexusID > 0 {
		return "Nexus"
	}
	if strings.HasPrefix(strings.ToLower(u.URL), "https://github.com/") {
		return "GitHub"
	}
	if strings.Contains(strings.ToLower(u.URL), "nexusmods.com/") {
		return "Nexus"
	}
	if parsed, err := url.Parse(u.URL); err == nil {
		host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
		if name, ok := source.NameOfHost(host); ok {
			return name
		}
		switch host {
		case "curseforge.com", "minecraft.curseforge.com":
			return "CurseForge"
		case "chucklefish.com":
			return "Chucklefish"
		}
		if host != "" {
			return host
		}
	}
	return ""
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
func Relate(mods []Installed, domain, key, uniqueID string) (r Relations, ok bool) {
	i := slices.IndexFunc(mods, func(x Installed) bool { return x.Key == key && profile.SameID(x.UniqueID, uniqueID) })
	if i < 0 {
		return Relations{}, false
	}
	self := mods[i]
	r = Relations{PageURL: pageURL(domain, self.UpdateKeys), Needs: []Need{}, NeededBy: []Dependent{}}
	for _, dep := range self.Dependencies {
		n := Need{UniqueID: dep.UniqueID, Name: dep.UniqueID, MinimumVersion: dep.MinimumVersion, Required: dep.Required, State: "ok"}
		if j := slices.IndexFunc(mods, func(x Installed) bool { return profile.SameID(x.UniqueID, dep.UniqueID) }); j >= 0 {
			n.Name = mods[j].Name
		}
		if reason, have := depState(mods, dep); reason != "" {
			n.State, n.InstalledVersion = reason, have
		}
		r.Needs = append(r.Needs, n)
	}
	for _, x := range mods {
		if x.Key == self.Key && profile.SameID(x.UniqueID, self.UniqueID) {
			continue
		}
		if slices.ContainsFunc(x.Dependencies, func(d manifest.Dependency) bool { return profile.SameID(d.UniqueID, self.UniqueID) }) {
			r.NeededBy = append(r.NeededBy, Dependent{Key: x.Key, UniqueID: x.UniqueID, Name: x.Name})
		}
	}
	return r, true
}

// Pages maps "key/uniqueId" to the page of each mod whose update keys name one.
func Pages(mods []Installed, domain string) map[string]string {
	out := map[string]string{}
	for _, m := range mods {
		if u := pageURL(domain, m.UpdateKeys); u != "" {
			out[m.Key+"/"+m.UniqueID] = u
		}
	}
	return out
}

// pageURL is the page of the first update key that names one: a Nexus mod or a GitHub repository.
func pageURL(domain string, keys []string) string {
	for _, k := range keys {
		if n, ok := manifest.NexusUpdateKey(k); ok {
			return siteURL(domain, meta.Ref{Site: "Nexus", ID: n})
		}
		if repo, ok := manifest.GitHubUpdateKey(k); ok {
			return "https://github.com/" + repo
		}
	}
	return ""
}

// nexusID is the Nexus mod of the first update key that names one, or 0.
func nexusID(keys []string) int {
	for _, k := range keys {
		if n, ok := manifest.NexusUpdateKey(k); ok {
			return n
		}
	}
	return 0
}

// nexusUpdate is the Nexus mod of the update keys, unless the suggested update lives somewhere other than Nexus or GitHub.
func nexusUpdate(keys []string, url string) int {
	u := strings.ToLower(url)
	if u != "" && !strings.Contains(u, "nexusmods.com/") && !strings.HasPrefix(u, "https://github.com/") {
		return 0
	}
	return nexusID(keys)
}

// githubUpdate is the repository of the first GitHub update key, provided the suggested update lives on GitHub too.
func githubUpdate(keys []string, url string) string {
	if !strings.HasPrefix(strings.ToLower(url), "https://github.com/") {
		return ""
	}
	for _, k := range keys {
		if repo, ok := manifest.GitHubUpdateKey(k); ok {
			return repo
		}
	}
	return ""
}

// metadataFallback is the GitHub repo SMAPI's metadata lists for a mod whose manifest has no GitHub key, unless the
// suggested download is already on GitHub. A free Nexus account must click for every file, so GitHub goes first.
func metadataFallback(repo, url string) string {
	if strings.HasPrefix(strings.ToLower(url), "https://github.com/") || strings.Count(repo, "/") != 1 {
		return ""
	}
	return repo
}

// githubFallback is the repo of the mod's GitHub update key when the suggested download is not on GitHub.
func githubFallback(keys []string, url string) string {
	if githubUpdate(keys, url) != "" {
		return ""
	}
	for _, k := range keys {
		if repo, ok := manifest.GitHubUpdateKey(k); ok {
			return repo
		}
	}
	return ""
}

// downloaded reports whether the entry's download is already at version or newer, so the update is installed
// even though this manifest was left at an older version inside it.
func downloaded(x Installed, version string) bool {
	if x.SourceVersion == "" {
		return false
	}
	c, ok := meta.CompareVersions(x.SourceVersion, version)
	return ok && c >= 0
}

var nexusFileVersionSuffix = regexp.MustCompile(`(?i)(?:[\s._-]+v?\d+(?:[._-]\d+)+|[\s._-]+v?\d+)$`)

func nexusFileStem(name string) string {
	name = strings.TrimSpace(name)
	lower := strings.ToLower(name)
	for _, ext := range []string{".zip", ".rar", ".7z"} {
		if strings.HasSuffix(lower, ext) {
			name = strings.TrimSpace(name[:len(name)-len(ext)])
			break
		}
	}
	for {
		stem := strings.TrimSpace(nexusFileVersionSuffix.ReplaceAllString(name, ""))
		if stem == name {
			break
		}
		name = stem
	}
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

// sameNexusFileGroup prefers the manifests inside each file: two files carrying the same mod are versions of one
// download. Archive names are the fallback; the dataset sometimes stores them as hashed paths.
func sameNexusFileGroup(a, b meta.File) bool {
	if len(a.Mods) > 0 && len(b.Mods) > 0 {
		return slices.ContainsFunc(a.Mods, func(m meta.Mod) bool { return containsPreviewMod(b, m.UniqueID) })
	}
	aStem, bStem := nexusFileStem(a.FileName), nexusFileStem(b.FileName)
	return aStem == "" || bStem == "" || aStem == bStem
}

func newerPreviewFile(candidate, current meta.File) bool {
	candidateOld := strings.EqualFold(candidate.Type, "OLD_VERSION")
	currentOld := strings.EqualFold(current.Type, "OLD_VERSION")
	if candidateOld != currentOld {
		return currentOld
	}
	return candidate.ID > current.ID
}

func containsPreviewMod(file meta.File, uniqueID string) bool {
	return slices.ContainsFunc(file.Mods, func(m meta.Mod) bool { return profile.SameID(m.UniqueID, uniqueID) })
}

// nexusFileIsCurrent uses the cached SMAPI file preview to keep a stale manifest from making a file update itself.
// The preview also prevents a raw game-content file from becoming an update target for a SMAPI mod.
func nexusFileIsCurrent(ctx context.Context, m Meta, x Installed, url, suggested string) bool {
	modID := nexusUpdate(x.UpdateKeys, url)
	if modID == 0 {
		return false
	}
	keyModID, fileID, ok := store.NexusFile(x.Key)
	if !ok || keyModID != modID {
		return false
	}
	page, err := m.Page(ctx, modID)
	if err != nil {
		return false
	}
	var installed meta.File
	for _, file := range page.Downloads {
		if file.ID == int64(fileID) {
			installed = file
			break
		}
	}
	if installed.ID == 0 {
		return false
	}
	latest := installed
	for _, file := range page.Downloads {
		if sameNexusFileGroup(file, installed) && newerPreviewFile(file, latest) {
			latest = file
		}
	}
	if latest.ID == installed.ID {
		// The dataset trails Nexus: when SMAPI already knows a newer version than the newest file the page lists,
		// the page is stale and the update is real.
		if c, ok := meta.CompareVersions(latest.Version, suggested); ok && c < 0 {
			return false
		}
		return !strings.EqualFold(installed.Type, "OLD_VERSION")
	}
	return !containsPreviewMod(latest, x.UniqueID)
}

// liveNexusFiles asks Nexus once for the current files of every flagged Nexus mod, so a stale dataset page cannot
// hide a new file or invent one. It returns nothing when signed out or when the call fails.
func liveNexusFiles(ctx context.Context, filesOf NexusFilesOf, t nexus.Title, asked []Installed, results []meta.UpdateResult) map[int][]nexus.BatchFile {
	if filesOf == nil {
		return nil
	}
	var ids []int
	for i, res := range results {
		for _, u := range []*meta.Update{res.Suggested, res.Unofficial} {
			if u == nil || i >= len(asked) {
				continue
			}
			if id := nexusUpdate(asked[i].UpdateKeys, u.URL); id != 0 && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	files, err := filesOf(ctx, t, ids)
	if err != nil {
		return nil
	}
	return files
}

// liveFileIsCurrent says whether the installed Nexus file is still the newest in its group (same display name) and
// already at the suggested version. known is false when the installed file is not in the list.
func liveFileIsCurrent(files []nexus.BatchFile, x Installed, suggested string) (current, known bool) {
	_, fileID, ok := store.NexusFile(x.Key)
	if !ok {
		return false, false
	}
	var installed nexus.BatchFile
	for _, f := range files {
		if f.FileID == fileID {
			installed = f
		}
	}
	if installed.FileID == 0 {
		return false, false
	}
	for _, f := range files {
		if f.FileID > installed.FileID && !strings.EqualFold(f.Category, "OLD_VERSION") &&
			strings.EqualFold(strings.TrimSpace(f.Name), strings.TrimSpace(installed.Name)) {
			return false, true
		}
	}
	if c, ok := meta.CompareVersions(installed.Version, suggested); ok && c < 0 {
		return false, true
	}
	return true, true
}
