// Package problems finds what is wrong with a profile's mods: dependencies that are missing, mods installed
// twice, and mods SMAPI's API marks broken for the game version. It also reports the updates SMAPI's API
// suggests and how one mod relates to the others. It only reports; nothing here blocks a launch.
package problems

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

// Meta is the slice of meta.Client the checks use.
type Meta interface {
	Lookup(ctx context.Context, uniqueID string) ([]meta.Ref, error)
	Page(ctx context.Context, id int) (meta.Page, error)
	PageRequirements(ctx context.Context, pageID int) ([]meta.Requirement, error)
	CheckUpdates(ctx context.Context, req meta.UpdateRequest) []meta.UpdateResult
}

// Installed is one mod of the profile.
type Installed struct {
	Key string
	// SourceKind is the entry's source: "local" for an archive, "smapi" for the loader's own mods, "mortar" for the console bridge, "nexus" for a download.
	SourceKind string
	// SourceVersion is the version of the download the entry came from (a Nexus file's version). Authors
	// often leave some manifests of a multi-part download unbumped, so it can be newer than Version.
	SourceVersion string
	Enabled       bool
	Pinned        bool
	SkipVersion   string
	IgnoreUpdates bool
	// Folder is the mod's directory in the profile, used to read Content Patcher content.json.
	Folder string
	manifest.Manifest
}

// Environment says what the profile runs on, for the broken-mod check.
type Environment struct {
	GameVersion string
	APIVersion  string
	Platform    string
}

// Ref names a mod on the page that hosts it. FileID is 0 unless a Nexus file satisfying the requirement was found.
// GitHub is "owner/repo" when Site is "GitHub".
type Ref struct {
	Site     string `json:"site"`
	GitHub   string `json:"github"`
	PageID   int    `json:"pageId"`
	PageName string `json:"pageName"`
	URL      string `json:"url"`
	FileID   int64  `json:"fileId"`
	FileName string `json:"fileName"`
	Version  string `json:"version"`
}

// Missing is a required dependency the profile does not satisfy. Reason is "absent", "disabled" or "outdated".
// Where is nil when the dataset does not know the mod or could not be reached.
type Missing struct {
	DependentID      string `json:"dependentId"`
	DependentName    string `json:"dependentName"`
	UniqueID         string `json:"uniqueId"`
	MinimumVersion   string `json:"minimumVersion"`
	Reason           string `json:"reason"`
	InstalledVersion string `json:"installedVersion"`
	Listed           bool   `json:"listed"`
	Note             string `json:"note"`
	Optional         bool   `json:"optional"`
	Where            *Ref   `json:"where"`
}

// Copy is one of two or more enabled copies of a mod. Needed and TooOld name the enabled mods that depend on the
// UniqueID and whose MinimumVersion this copy meets or does not. Newest marks the highest version among the copies.
type Copy struct {
	Key     string   `json:"key"`
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Source  string   `json:"source"`
	Nexus   bool     `json:"nexus"`
	Newest  bool     `json:"newest"`
	Needed  []string `json:"needed"`
	TooOld  []string `json:"tooOld"`
}

// Duplicate is a UniqueID with several enabled copies.
type Duplicate struct {
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
	Copies   []Copy `json:"copies"`
}

// Broken is an enabled mod SMAPI's API marks broken, obsolete or abandoned for the game version.
type Broken struct {
	Key         string `json:"key"`
	UniqueID    string `json:"uniqueId"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	BrokeIn     string `json:"brokeIn"`
	Summary     string `json:"summary,omitempty"`
	Replacement *Ref   `json:"replacement,omitempty"`
}

// SettingHint is a Content Patcher compatibility setting that is not enabled for the profile.
type SettingHint struct {
	Key         string   `json:"key"`
	UniqueID    string   `json:"uniqueId"`
	Name        string   `json:"name"`
	Field       string   `json:"field"`
	Current     string   `json:"current"`
	Suggested   []string `json:"suggested"`
	For         []string `json:"for"`
	ForNames    []string `json:"forNames"`
	Description string   `json:"description"`
	// Variant marks a picker (a palette, recolour or similar choice) whose values the pack maps to mods.
	// CurrentFor names the mod the current value is for when that mod is not enabled.
	Variant    bool   `json:"variant"`
	CurrentFor string `json:"currentFor"`
}

// Result is everything found for one profile. Unknown is set when a lookup failed, so the lists may be short.
type Result struct {
	Missing        []Missing       `json:"missing"`
	Duplicates     []Duplicate     `json:"duplicates"`
	Broken         []Broken        `json:"broken"`
	AssetConflicts []AssetConflict `json:"assetConflicts"`
	Settings       []SettingHint   `json:"settings"`
	RunErrors      []RunError      `json:"runErrors"`
	Drift          []profile.Drift `json:"drift,omitempty"`
	Unknown        bool            `json:"unknown"`
}

// Count is the number of problems, one per missing dependency, duplicate, broken mod, asset conflict, setting and last-run error.
func (r Result) Count() int {
	conflicts := 0
	for _, c := range r.AssetConflicts {
		if !c.Cosmetic {
			conflicts++
		}
	}
	return len(r.Missing) + len(r.Duplicates) + len(r.Broken) + conflicts + len(r.Settings) + len(r.RunErrors) + len(r.Drift)
}

func sameID(a, b string) bool { return strings.EqualFold(a, b) }

// meets reports whether version satisfies minimum. What cannot be compared is taken as satisfied: SMAPI itself
// only complains about versions it can order.
func meets(version, minimum string) bool {
	if minimum == "" {
		return true
	}
	c, ok := meta.CompareVersions(version, minimum)
	return !ok || c >= 0
}

// Check computes the problems of mods. Lookups that fail leave Unknown set and never return an error.
func Check(ctx context.Context, m Meta, env Environment, mods []Installed) Result {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(x Installed) bool { return !x.Enabled })
	r := Result{
		Missing:        []Missing{},
		Duplicates:     duplicates(enabled),
		Broken:         []Broken{},
		AssetConflicts: assetConflicts(enabled),
		Settings:       compatibilitySettings(enabled),
		Drift:          []profile.Drift{},
	}
	missing := missingDeps(enabled, mods)
	listed, listedUnknown := listedRequirements(ctx, m, enabled, mods)
	missing = append(missing, listed...)
	r.Unknown = fillWhere(ctx, m, enabled, missing)
	r.Unknown = r.Unknown || listedUnknown
	r.Missing = missing
	broken, unknown := brokenMods(ctx, m, env, enabled)
	r.Broken = broken
	r.Unknown = r.Unknown || unknown
	return r
}

func missingDeps(enabled, all []Installed) []Missing {
	out := []Missing{}
	for _, d := range enabled {
		for _, dep := range d.Dependencies {
			if !dep.Required {
				continue
			}
			miss := Missing{DependentID: d.UniqueID, DependentName: d.Name, UniqueID: dep.UniqueID, MinimumVersion: dep.MinimumVersion}
			reason, installedVersion := depState(all, dep)
			if reason == "" {
				continue
			}
			miss.Reason, miss.InstalledVersion = reason, installedVersion
			out = append(out, miss)
		}
	}
	return out
}

// fetchAll calls fetch for every id, a few at a time: a cold profile needs one Nexus round trip per page, and
// doing them one by one kept the first check after a start busy for many seconds. failed reports any error.
func fetchAll[T any](ctx context.Context, ids []int, fetch func(context.Context, int) (T, error)) (got map[int]T, failed bool) {
	got = make(map[int]T, len(ids))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			v, err := fetch(ctx, id)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed = true
				return
			}
			got[id] = v
		}()
	}
	wg.Wait()
	return got, failed
}

func listedRequirements(ctx context.Context, m Meta, enabled, all []Installed) ([]Missing, bool) {
	out := []Missing{}
	var pageIDs []int
	for _, d := range enabled {
		if pageID, ok := nexusEntryPage(d.Key); ok && !slices.Contains(pageIDs, pageID) {
			pageIDs = append(pageIDs, pageID)
		}
	}
	requirementsByPage, unknown := fetchAll(ctx, pageIDs, m.PageRequirements)
	var reqIDs []int
	for _, reqs := range requirementsByPage {
		for _, req := range reqs {
			if !slices.Contains(reqIDs, req.ModID) {
				reqIDs = append(reqIDs, req.ModID)
			}
		}
	}
	pages, pagesUnknown := fetchAll(ctx, reqIDs, m.Page)
	unknown = unknown || pagesUnknown
	seenEntries := map[string]bool{}

	for _, d := range enabled {
		pageID, ok := nexusEntryPage(d.Key)
		if !ok {
			continue
		}
		entryKey := strings.ToLower(d.Key)
		if seenEntries[entryKey] {
			continue
		}
		seenEntries[entryKey] = true
		seenRequirements := map[int]bool{}
		for _, req := range requirementsByPage[pageID] {
			if seenRequirements[req.ModID] {
				continue
			}
			seenRequirements[req.ModID] = true
			page, hasPage := pages[req.ModID]
			if hasPage && manifestHasRequirement(all, d.Key, page) {
				continue
			}
			reason, satisfied := listedDepState(all, req.ModID, page, hasPage)
			if satisfied && reason == "" {
				continue
			}
			uniqueID := "nexus:" + strconv.Itoa(req.ModID)
			if hasPage {
				if id := mainUniqueID(page); id != "" {
					uniqueID = id
				}
			}
			miss := Missing{
				DependentID:   d.UniqueID,
				DependentName: d.Name,
				UniqueID:      uniqueID,
				Reason:        reason,
				Listed:        true,
				Note:          req.Notes,
				Optional:      optionalRequirement(req.Notes),
			}
			if miss.Reason == "" {
				miss.Reason = "absent"
			}
			pageName := page.Name
			if pageName == "" {
				pageName = req.Name
			}
			pageRef := &Ref{
				Site:     "Nexus",
				PageID:   req.ModID,
				PageName: pageName,
				URL:      "https://www.nexusmods.com/stardewvalley/mods/" + strconv.Itoa(req.ModID),
			}
			if uniqueID == "nexus:"+strconv.Itoa(req.ModID) {
				miss.Where = pageRef
			} else if located, ok := Locate(ctx, m, uniqueID, "", d.UpdateKeys); located != nil {
				miss.Where = located
				unknown = unknown || !ok
			} else {
				miss.Where = pageRef
				unknown = unknown || !ok
			}
			out = append(out, miss)
		}
	}
	return out, unknown
}

func nexusEntryPage(key string) (int, bool) {
	if !strings.HasPrefix(strings.ToLower(key), "nexus-") {
		return 0, false
	}
	rest := key[len("nexus-"):]
	page, _, ok := strings.Cut(rest, "-")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(page)
	return n, err == nil
}

func listedDepState(all []Installed, pageID int, page meta.Page, pageKnown bool) (string, bool) {
	prefix := "nexus-" + strconv.Itoa(pageID) + "-"
	disabled := false
	for _, x := range all {
		matches := strings.HasPrefix(strings.ToLower(x.Key), prefix)
		if !matches && pageKnown {
			for _, file := range page.Downloads {
				for _, mod := range file.Mods {
					if sameID(x.UniqueID, mod.UniqueID) {
						matches = true
						break
					}
				}
				if matches {
					break
				}
			}
		}
		if !matches {
			continue
		}
		if x.Enabled {
			return "", true
		}
		disabled = true
	}
	if disabled {
		return "disabled", false
	}
	return "absent", false
}

func manifestHasRequirement(all []Installed, entryKey string, page meta.Page) bool {
	for _, d := range all {
		if !d.Enabled || !sameID(d.Key, entryKey) {
			continue
		}
		for _, dep := range d.Dependencies {
			if !dep.Required {
				continue
			}
			for _, file := range page.Downloads {
				for _, mod := range file.Mods {
					if sameID(dep.UniqueID, mod.UniqueID) {
						return true
					}
				}
			}
		}
	}
	return false
}

func mainUniqueID(page meta.Page) string {
	for _, file := range page.Downloads {
		if !strings.EqualFold(file.Type, "main") {
			continue
		}
		for _, mod := range file.Mods {
			if mod.UniqueID != "" {
				return mod.UniqueID
			}
		}
	}
	return ""
}

func optionalRequirement(note string) bool {
	note = strings.ToLower(strings.TrimSpace(note))
	return strings.Contains(note, "optional") ||
		strings.Contains(note, "recommended") ||
		strings.Contains(note, "not strictly") ||
		(strings.HasPrefix(note, "for") && (len(note) == 3 || !isWordByte(note[3])))
}

func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}

// depState says why the mods in all do not satisfy dep: "absent", "disabled" or "outdated", with the highest
// installed version for the last. An empty reason means the dependency is met.
func depState(all []Installed, dep manifest.Dependency) (reason, installedVersion string) {
	var installed []Installed
	for _, x := range all {
		if sameID(x.UniqueID, dep.UniqueID) {
			installed = append(installed, x)
		}
	}
	switch {
	case len(installed) == 0:
		return "absent", ""
	case slices.ContainsFunc(installed, func(x Installed) bool { return x.Enabled && meets(x.Version, dep.MinimumVersion) }):
		return "", ""
	case slices.ContainsFunc(installed, func(x Installed) bool { return x.Enabled }):
		return "outdated", highest(installed)
	case slices.ContainsFunc(installed, func(x Installed) bool { return meets(x.Version, dep.MinimumVersion) }):
		return "disabled", ""
	}
	return "outdated", highest(installed)
}

func highest(mods []Installed) string {
	best := mods[0].Version
	for _, x := range mods[1:] {
		if c, ok := meta.CompareVersions(x.Version, best); ok && c > 0 {
			best = x.Version
		}
	}
	return best
}

func duplicates(enabled []Installed) []Duplicate {
	out := []Duplicate{}
	seen := map[string]bool{}
	for _, first := range enabled {
		id := strings.ToLower(first.UniqueID)
		if seen[id] {
			continue
		}
		seen[id] = true
		var group []Installed
		for _, x := range enabled {
			if sameID(x.UniqueID, first.UniqueID) {
				group = append(group, x)
			}
		}
		if len(group) < 2 {
			continue
		}
		out = append(out, Duplicate{UniqueID: first.UniqueID, Name: first.Name, Copies: copies(group, enabled)})
	}
	return out
}

func copies(group, enabled []Installed) []Copy {
	top := highest(group)
	out := make([]Copy, len(group))
	for i, g := range group {
		c := Copy{Key: g.Key, Name: g.Name, Version: g.Version, Source: g.SourceKind, Nexus: g.SourceKind == profile.KindNexus, Needed: []string{}, TooOld: []string{}}
		if cmpv, ok := meta.CompareVersions(g.Version, top); ok && cmpv == 0 {
			c.Newest = true
		}
		for _, d := range enabled {
			for _, dep := range d.Dependencies {
				if !dep.Required || !sameID(dep.UniqueID, g.UniqueID) {
					continue
				}
				if meets(g.Version, dep.MinimumVersion) {
					c.Needed = append(c.Needed, d.Name)
				} else {
					c.TooOld = append(c.TooOld, d.Name)
				}
			}
		}
		out[i] = c
	}
	return out
}

func brokenMods(ctx context.Context, m Meta, env Environment, enabled []Installed) (broken []Broken, unknown bool) {
	broken = []Broken{}
	if len(enabled) == 0 {
		return broken, false
	}
	req := meta.UpdateRequest{APIVersion: env.APIVersion, GameVersion: env.GameVersion, Platform: env.Platform}
	for _, x := range enabled {
		req.Mods = append(req.Mods, meta.InstalledMod{ID: x.UniqueID, UpdateKeys: x.UpdateKeys, Version: x.Version})
	}
	for i, res := range m.CheckUpdates(ctx, req) {
		if !res.Known {
			unknown = true
			continue
		}
		s := strings.ToLower(res.Compatibility)
		if s != "broken" && s != "obsolete" && s != "abandoned" {
			continue
		}
		x := enabled[i]
		b := Broken{Key: x.Key, UniqueID: x.UniqueID, Name: x.Name, Status: s, BrokeIn: res.BrokeIn, Summary: res.CompatibilitySummary}
		b.Replacement = replacementFromSummary(ctx, m, x.UpdateKeys, res.CompatibilitySummary)
		broken = append(broken, b)
	}
	return broken, unknown
}

// fillWhere finds where each absent or outdated dependency can be had. It reports whether any lookup failed.
func fillWhere(ctx context.Context, m Meta, enabled []Installed, missing []Missing) (unknown bool) {
	type found struct {
		ref *Ref
		ok  bool
	}
	cache := map[string]found{}
	for i := range missing {
		x := &missing[i]
		if x.Reason == "disabled" {
			continue
		}
		if x.Where != nil {
			continue
		}
		dependent := slices.IndexFunc(enabled, func(e Installed) bool { return sameID(e.UniqueID, x.DependentID) })
		var keys []string
		if dependent >= 0 {
			keys = enabled[dependent].UpdateKeys
		}
		k := strings.ToLower(x.UniqueID) + "|" + x.MinimumVersion + "|" + strings.Join(keys, ",")
		f, hit := cache[k]
		if !hit {
			f.ref, f.ok = Locate(ctx, m, x.UniqueID, x.MinimumVersion, keys)
			cache[k] = f
		}
		x.Where = f.ref
		unknown = unknown || !f.ok
	}
	return unknown
}

// nexusKey returns the page number of a "Nexus:1234" or "Nexus:1234@subkey" update key.
func nexusKey(key string) (int, bool) {
	site, rest, ok := strings.Cut(key, ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(site), "nexus") {
		return 0, false
	}
	rest, _, _ = strings.Cut(rest, "@")
	n, err := strconv.Atoi(strings.TrimSpace(rest))
	return n, err == nil
}

func siteURL(r meta.Ref) string {
	switch strings.ToLower(r.Site) {
	case "nexus":
		return "https://www.nexusmods.com/stardewvalley/mods/" + strconv.Itoa(r.ID)
	case "curseforge":
		return "https://www.curseforge.com/projects/" + strconv.Itoa(r.ID)
	case "moddrop":
		return "https://www.moddrop.com/stardew-valley/mods/" + strconv.Itoa(r.ID)
	}
	return ""
}

// Locate names the page and file to get uniqueID from. ok is false when the dataset could not be read, as
// opposed to reading it and finding the mod unlisted (nil, true).
func Locate(ctx context.Context, m Meta, uniqueID, minimum string, dependentKeys []string) (*Ref, bool) {
	refs, err := m.Lookup(ctx, uniqueID)
	if err != nil {
		return nil, false
	}
	var nexus, other []meta.Ref
	for _, r := range refs {
		if strings.EqualFold(r.Site, "nexus") {
			nexus = append(nexus, r)
		} else if siteURL(r) != "" {
			other = append(other, r)
		}
	}
	if len(nexus) == 0 {
		if repo := githubRepo(ctx, m, uniqueID); repo != "" {
			return &Ref{Site: "GitHub", GitHub: repo, URL: "https://github.com/" + repo}, true
		}
		if len(other) == 0 {
			return nil, true
		}
		return &Ref{Site: other[0].Site, PageID: other[0].ID, URL: siteURL(other[0])}, true
	}
	for _, k := range dependentKeys {
		if n, ok := nexusKey(k); ok {
			if i := slices.IndexFunc(nexus, func(r meta.Ref) bool { return r.ID == n }); i >= 0 {
				nexus = []meta.Ref{nexus[i]}
				break
			}
		}
	}
	var best *Ref
	var bestTop string
	ok := true
	for _, r := range nexus {
		page, err := m.Page(ctx, r.ID)
		if err != nil {
			ok = false
			continue
		}
		cand, top := fileIn(page, r, uniqueID, minimum)
		if cand == nil {
			continue
		}
		if c, isOrdered := meta.CompareVersions(top, bestTop); best == nil || (isOrdered && c > 0) {
			best, bestTop = cand, top
		}
	}
	if best == nil {
		return &Ref{Site: "Nexus", PageID: nexus[0].ID, URL: siteURL(nexus[0])}, ok
	}
	return best, ok
}

// githubRepo is the repository SMAPI's update API records for uniqueID, or "".
func githubRepo(ctx context.Context, m Meta, uniqueID string) string {
	got := m.CheckUpdates(ctx, meta.UpdateRequest{Mods: []meta.InstalledMod{{ID: uniqueID}}})
	if len(got) != 1 || !got[0].Known {
		return ""
	}
	if repo, ok := githubKey("github:" + got[0].GitHubRepo); ok {
		return repo
	}
	return ""
}

// fileIn picks the newest MAIN file of the page that holds uniqueID at a version meeting minimum. With none,
// it still returns the page when some file holds the mod, so its link is worth showing; otherwise nil.
// top is the highest version of the mod on the page, which ranks pages against each other.
func fileIn(page meta.Page, r meta.Ref, uniqueID, minimum string) (ref *Ref, top string) {
	pageRef := Ref{Site: "Nexus", PageID: r.ID, PageName: page.Name, URL: cmp.Or(page.PageURL, siteURL(r))}
	var best *Ref
	holds := false
	for _, f := range page.Downloads {
		for _, mod := range f.Mods {
			if !sameID(mod.UniqueID, uniqueID) {
				continue
			}
			if c, ok := meta.CompareVersions(mod.Version, top); !holds || (ok && c > 0) {
				top = mod.Version
			}
			holds = true
			if !strings.EqualFold(f.Type, "main") || !meets(mod.Version, minimum) {
				continue
			}
			cand := pageRef
			cand.FileID, cand.FileName, cand.Version = f.ID, f.FileName, mod.Version
			if best == nil || newer(cand, *best) {
				best = &cand
			}
		}
	}
	switch {
	case best != nil:
		return best, top
	case holds:
		return &pageRef, top
	}
	return nil, ""
}

// newer orders two files of one mod by version, then by file ID.
func newer(a, b Ref) bool {
	if c, ok := meta.CompareVersions(a.Version, b.Version); ok && c != 0 {
		return c > 0
	}
	return a.FileID > b.FileID
}
