// Package problems finds what is wrong with a profile's mods: dependencies that are missing, mods installed
// twice, and mods SMAPI's API marks broken for the game version. It also reports the updates SMAPI's API
// suggests and how one mod relates to the others. It only reports; nothing here blocks a launch.
package problems

import (
	"cmp"
	"context"
	"encoding/json"
	"log"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// Meta is the slice of meta.Client the checks use.
type Meta interface {
	Lookup(ctx context.Context, uniqueID string) ([]meta.Ref, error)
	Page(ctx context.Context, id int) (meta.Page, error)
	PageRequirements(ctx context.Context, domain string, pageID int) ([]meta.Requirement, error)
	Collection(ctx context.Context, domain, slug string, revision int) (meta.Collection, error)
	CheckUpdates(ctx context.Context, req meta.UpdateRequest) []meta.UpdateResult
}

// Environment says what the profile runs on, for the broken-mod check.
type Environment struct {
	// Nexus is how Nexus names the game the check runs for.
	Nexus       nexus.Title
	GameVersion string
	APIVersion  string
	Platform    string
	// VersionScheme is how the game's loader versions its mods (deps.SemverSMAPI and the like); dependency
	// constraints are read in it.
	VersionScheme string
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
	DependentID      mod.ID `json:"dependentId"`
	DependentName    string `json:"dependentName"`
	ID               mod.ID `json:"id"`
	MinimumVersion   string `json:"minimumVersion"`
	Reason           string `json:"reason"`
	InstalledVersion string `json:"installedVersion"`
	Listed           bool   `json:"listed"`
	Note             string `json:"note"`
	Optional         bool   `json:"optional"`
	Where            *Ref   `json:"where"`
}

// Copy is one of two or more enabled copies of a mod. Needed and TooOld name the enabled mods that depend on the
// mod id and whose MinimumVersion this copy meets or does not. Newest marks the highest version among the copies.
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

// Duplicate is a mod id with several enabled copies.
type Duplicate struct {
	ID            mod.ID      `json:"id"`
	Name          string      `json:"name"`
	Copies        []Copy      `json:"copies"`
	NexusFiles    []NexusFile `json:"nexusFiles,omitempty"`
	NexusOptional bool        `json:"nexusOptional,omitempty"`
}

type NexusFile struct {
	Key      string `json:"key"`
	FileName string `json:"fileName"`
	Version  string `json:"version"`
	Remove   bool   `json:"remove"`
}

// Broken is an enabled mod SMAPI's API marks broken, obsolete or abandoned for the game version.
type Broken struct {
	Key         string `json:"key"`
	ID          mod.ID `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	BrokeIn     string `json:"brokeIn"`
	Summary     string `json:"summary,omitempty"`
	Replacement *Ref   `json:"replacement,omitempty"`
}

// Damaged is a mod whose stored files no longer match what was stored: files went missing, changed or appeared.
// Files names the first few.
type Damaged struct {
	Key     string   `json:"key"`
	Name    string   `json:"name"`
	Missing int      `json:"missing"`
	Changed int      `json:"changed"`
	Extra   int      `json:"extra"`
	Files   []string `json:"files"`
}

type DismissedProblem struct {
	Token         string                   `json:"token"`
	AssetConflict *framework.AssetConflict `json:"assetConflict,omitempty"`
	Broken        *Broken                  `json:"broken,omitempty"`
	Missing       *Missing                 `json:"missing,omitempty"`
	Setting       *framework.SettingHint   `json:"setting,omitempty"`
}

// Result is everything found for one profile. Unknown is set when a lookup failed, so the lists may be short.
type Result struct {
	Missing        []Missing                 `json:"missing"`
	Duplicates     []Duplicate               `json:"duplicates"`
	Broken         []Broken                  `json:"broken"`
	AssetConflicts []framework.AssetConflict `json:"assetConflicts"`
	Settings       []framework.SettingHint   `json:"settings"`
	Cleanup        []framework.Cleanup       `json:"cleanup,omitempty"`
	Compat         []Compat                  `json:"compat,omitempty"`
	Redundant      []framework.Redundant     `json:"redundant,omitempty"`
	RunErrors      []RunError                `json:"runErrors"`
	Drift          []profile.Drift           `json:"drift,omitempty"`
	Damaged        []Damaged                 `json:"damaged,omitempty"`
	Dismissed      []DismissedProblem        `json:"dismissed"`
	Unknown        bool                      `json:"unknown"`
	Timings        []CheckTiming             `json:"timings,omitempty"`
}

// CheckTiming is one check family's duration and item count from the last Problems run for a profile.
type CheckTiming struct {
	Name  string `json:"name"`
	Ms    int64  `json:"ms"`
	Count int    `json:"count"`
}

// Count is the number of problems, one per missing dependency, duplicate, broken mod, asset conflict, setting and last-run error.
func (r Result) Count() int {
	conflicts := conflictRows(r.AssetConflicts)
	duplicates := 0
	for _, duplicate := range r.Duplicates {
		if !duplicate.NexusOptional {
			duplicates++
		}
	}
	return len(r.Missing) + duplicates + len(r.Broken) + conflicts + len(r.Settings) + len(r.RunErrors) + len(r.Drift) + len(r.Damaged)
}

// WarningCount is cosmetic asset conflicts plus compat, cleanup and redundancy hints.
func (r Result) WarningCount() int {
	n := len(r.Compat) + len(r.Cleanup) + redundantCount(r.Redundant)
	for _, c := range r.AssetConflicts {
		if c.Cosmetic {
			n++
		}
	}
	return n
}

// satisfied reports whether version meets dep's constraint under scheme.
func satisfied(scheme string, dep manifest.Dependency, version string) bool {
	return deps.Satisfies(scheme, version, dep.Dep().Constraint)
}

// meets is satisfied for a bare minimum in SMAPI's scheme, which is what the dataset's Nexus files are versioned in.
func meets(version, minimum string) bool {
	return deps.Satisfies(deps.SemverSMAPI, version, minimum)
}

// Check computes the problems of mods. Lookups that fail leave Unknown set and never return an error.
func Check(ctx context.Context, m Meta, env Environment, mods []framework.Mod) Result {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(x framework.Mod) bool { return !x.Enabled })
	found, timings := runFrameworks(framework.Input{Enabled: enabled, All: mods})
	r := Result{
		Missing:        []Missing{},
		Broken:         []Broken{},
		AssetConflicts: found.AssetConflicts,
		Redundant:      found.Redundant,
		Drift:          []profile.Drift{},
		Timings:        timings,
	}
	reqStart := time.Now()
	missing := missingDeps(env.VersionScheme, enabled, mods)
	listed, listedUnknown := listedRequirements(ctx, m, env.Nexus.Domain, enabled, mods)
	missing = append(missing, listed...)
	r.Timings = append(r.Timings, CheckTiming{Name: "requirements", Ms: time.Since(reqStart).Milliseconds(), Count: len(missing)})
	r.Unknown = fillWhere(ctx, m, env.Nexus.Domain, enabled, missing)
	r.Unknown = r.Unknown || listedUnknown
	r.Missing = missing
	otherStart := time.Now()
	r.Duplicates = duplicatesWithNexus(ctx, m, env.VersionScheme, enabled)
	r.Settings = found.Settings
	r.Cleanup = cleanupHints(mods, found.Cleanup)
	broken, unknown := brokenMods(ctx, m, env, enabled)
	r.Broken = broken
	r.Unknown = r.Unknown || unknown
	r.Timings = append(r.Timings, CheckTiming{Name: "others", Ms: time.Since(otherStart).Milliseconds(), Count: len(r.Duplicates) + len(r.Broken) + len(r.Cleanup)})
	return r
}

// runFrameworks runs the analyzers of the frameworks present in the profile and merges what they find, timing each.
func runFrameworks(in framework.Input) (framework.Findings, []CheckTiming) {
	var all framework.Findings
	var timings []CheckTiming
	for _, f := range framework.Present(in.Enabled) {
		start := time.Now()
		got := f.Analyze(in)
		timings = append(timings, CheckTiming{
			Name:  "framework:" + f.ID().Local(),
			Ms:    time.Since(start).Milliseconds(),
			Count: len(slices.DeleteFunc(slices.Clone(in.Enabled), func(m framework.Mod) bool { return !f.Matches(m) })),
		})
		all.AssetConflicts = append(all.AssetConflicts, got.AssetConflicts...)
		all.Settings = append(all.Settings, got.Settings...)
		all.Cleanup = append(all.Cleanup, got.Cleanup...)
		all.Redundant = append(all.Redundant, got.Redundant...)
	}
	return all, timings
}

func logCheckTimings(profileID string, timings []CheckTiming) {
	for _, t := range timings {
		log.Printf("problems: %s %s %v (%d packs)", profileID, t.Name, time.Duration(t.Ms)*time.Millisecond, t.Count)
	}
}

func duplicatesWithNexus(ctx context.Context, m Meta, scheme string, enabled []framework.Mod) []Duplicate {
	out := duplicates(scheme, enabled)
	for i := range out {
		var pageID int
		var files []NexusFile
		optional := false
		for _, copy := range out[i].Copies {
			page, _, ok := store.NexusFile(copy.Key)
			if !ok {
				continue
			}
			if pageID == 0 {
				pageID = page
			}
			if page != pageID {
				continue
			}
			if !slices.ContainsFunc(files, func(file NexusFile) bool { return file.Key == copy.Key }) {
				files = append(files, NexusFile{Key: copy.Key})
			}
		}
		if len(files) < 2 || pageID == 0 {
			continue
		}
		page, err := m.Page(ctx, pageID)
		if err != nil {
			continue
		}
		for j := range files {
			_, fileID, _ := store.NexusFile(files[j].Key)
			for _, file := range page.Downloads {
				if file.ID != int64(fileID) {
					continue
				}
				files[j].FileName, files[j].Version = file.FileName, file.Version
				kind := strings.ToLower(file.Type)
				optional = optional || kind == "optional" || kind == "miscellaneous"
				break
			}
		}
		files = slices.DeleteFunc(files, func(file NexusFile) bool { return file.FileName == "" })
		if len(files) < 2 {
			continue
		}
		keep := 0
		for j := range files {
			if newerNexusFile(files[j], files[keep]) {
				keep = j
			}
		}
		for j := range files {
			files[j].Remove = j != keep
		}
		out[i].NexusFiles, out[i].NexusOptional = files, optional
	}
	return out
}

func newerNexusFile(a, b NexusFile) bool {
	if c, ok := meta.CompareVersions(a.Version, b.Version); ok && c != 0 {
		return c > 0
	}
	_, aID, _ := store.NexusFile(a.Key)
	_, bID, _ := store.NexusFile(b.Key)
	return aID > bID
}

func missingDeps(scheme string, enabled, all []framework.Mod) []Missing {
	out := []Missing{}
	for _, d := range enabled {
		for _, dep := range d.Dependencies {
			if !dep.Required {
				continue
			}
			miss := Missing{DependentID: d.ModID(), DependentName: d.Name, ID: dep.ModID(), MinimumVersion: dep.MinimumVersion}
			reason, installedVersion := depState(scheme, all, dep)
			if reason == "" {
				continue
			}
			miss.Reason, miss.InstalledVersion = reason, installedVersion
			out = append(out, miss)
		}
	}
	return out
}

// fetchAll calls fetch for every id, a few at a time, because a cold profile needs one Nexus round trip per page.
// failed reports any error.
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

func listedRequirements(ctx context.Context, m Meta, domain string, enabled, all []framework.Mod) ([]Missing, bool) {
	out := []Missing{}
	var pageIDs []int
	for _, d := range enabled {
		if pageID, _, ok := store.NexusFile(d.Key); ok && !slices.Contains(pageIDs, pageID) {
			pageIDs = append(pageIDs, pageID)
		}
	}
	requirementsByPage, unknown := fetchAll(ctx, pageIDs, func(ctx context.Context, id int) ([]meta.Requirement, error) {
		return m.PageRequirements(ctx, domain, id)
	})
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
		pageID, _, ok := store.NexusFile(d.Key)
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
			uniqueID := mod.NewID("nexus", strconv.Itoa(req.ModID))
			if hasPage {
				if id := mainID(page); id != "" {
					uniqueID = id
				}
			}
			miss := Missing{
				DependentID:   d.ModID(),
				DependentName: d.Name,
				ID:            uniqueID,
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
				URL:      nexus.ModURL(domain, req.ModID),
			}
			if uniqueID == mod.NewID("nexus", strconv.Itoa(req.ModID)) {
				miss.Where = pageRef
			} else if located, ok := Locate(ctx, m, domain, uniqueID, "", d.UpdateKeys); located != nil {
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

func listedDepState(all []framework.Mod, pageID int, page meta.Page, pageKnown bool) (string, bool) {
	disabled := false
	for _, x := range all {
		entryPage, _, listed := store.NexusFile(x.Key)
		matches := listed && entryPage == pageID
		if !matches && pageKnown {
			for _, file := range page.Downloads {
				for _, im := range file.Mods {
					if mod.Equal(x.ModID(), im.ModID()) {
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

func manifestHasRequirement(all []framework.Mod, entryKey string, page meta.Page) bool {
	for _, d := range all {
		if !d.Enabled || !strings.EqualFold(d.Key, entryKey) {
			continue
		}
		for _, dep := range d.Dependencies {
			if !dep.Required {
				continue
			}
			for _, file := range page.Downloads {
				for _, im := range file.Mods {
					if mod.Equal(dep.ModID(), im.ModID()) {
						return true
					}
				}
			}
		}
	}
	return false
}

func mainID(page meta.Page) mod.ID {
	for _, file := range page.Downloads {
		if !strings.EqualFold(file.Type, "main") {
			continue
		}
		for _, im := range file.Mods {
			if im.ModID() != "" {
				return im.ModID()
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
func depState(scheme string, all []framework.Mod, dep manifest.Dependency) (reason, installedVersion string) {
	var installed []framework.Mod
	for _, x := range all {
		if mod.Equal(x.ModID(), dep.ModID()) {
			installed = append(installed, x)
		}
	}
	switch {
	case len(installed) == 0:
		return "absent", ""
	case slices.ContainsFunc(installed, func(x framework.Mod) bool { return x.Enabled && satisfied(scheme, dep, x.Version) }):
		return "", ""
	case slices.ContainsFunc(installed, func(x framework.Mod) bool { return x.Enabled }):
		return "outdated", highest(installed)
	case slices.ContainsFunc(installed, func(x framework.Mod) bool { return satisfied(scheme, dep, x.Version) }):
		return "disabled", ""
	}
	return "outdated", highest(installed)
}

func highest(mods []framework.Mod) string {
	best := mods[0].Version
	for _, x := range mods[1:] {
		if c, ok := meta.CompareVersions(x.Version, best); ok && c > 0 {
			best = x.Version
		}
	}
	return best
}

func duplicates(scheme string, enabled []framework.Mod) []Duplicate {
	out := []Duplicate{}
	seen := map[string]bool{}
	for _, first := range enabled {
		id := first.ModID().Fold()
		if seen[id] {
			continue
		}
		seen[id] = true
		var group []framework.Mod
		for _, x := range enabled {
			if mod.Equal(x.ModID(), first.ModID()) {
				group = append(group, x)
			}
		}
		if len(group) < 2 {
			continue
		}
		out = append(out, Duplicate{ID: first.ModID(), Name: first.Name, Copies: copies(scheme, group, enabled)})
	}
	return out
}

func copies(scheme string, group, enabled []framework.Mod) []Copy {
	top := highest(group)
	out := make([]Copy, len(group))
	for i, g := range group {
		c := Copy{Key: g.Key, Name: g.Name, Version: g.Version, Source: g.SourceKind, Nexus: g.SourceKind == profile.KindNexus, Needed: []string{}, TooOld: []string{}}
		if cmpv, ok := meta.CompareVersions(g.Version, top); ok && cmpv == 0 {
			c.Newest = true
		}
		for _, d := range enabled {
			for _, dep := range d.Dependencies {
				if !dep.Required || !mod.Equal(dep.ModID(), g.ModID()) {
					continue
				}
				if satisfied(scheme, dep, g.Version) {
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

func brokenMods(ctx context.Context, m Meta, env Environment, enabled []framework.Mod) (broken []Broken, unknown bool) {
	broken = []Broken{}
	if len(enabled) == 0 {
		return broken, false
	}
	req := meta.UpdateRequest{APIVersion: env.APIVersion, GameVersion: env.GameVersion, Platform: env.Platform}
	for _, x := range enabled {
		req.Mods = append(req.Mods, meta.InstalledMod{ID: x.ModID().Local(), UpdateKeys: x.UpdateKeys, Version: x.Version})
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
		b := Broken{Key: x.Key, ID: x.ModID(), Name: x.Name, Status: s, BrokeIn: res.BrokeIn, Summary: res.CompatibilitySummary}
		b.Replacement = replacementFromSummary(ctx, m, env.Nexus.Domain, x.UpdateKeys, res.CompatibilitySummary)
		broken = append(broken, b)
	}
	return broken, unknown
}

// fillWhere finds where each absent or outdated dependency can be had. It reports whether any lookup failed.
func fillWhere(ctx context.Context, m Meta, domain string, enabled []framework.Mod, missing []Missing) (unknown bool) {
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
		dependent := slices.IndexFunc(enabled, func(e framework.Mod) bool { return mod.Equal(e.ModID(), x.DependentID) })
		var keys []string
		if dependent >= 0 {
			keys = enabled[dependent].UpdateKeys
		}
		k := x.ID.Fold() + "|" + x.MinimumVersion + "|" + strings.Join(keys, ",")
		f, hit := cache[k]
		if !hit {
			f.ref, f.ok = Locate(ctx, m, domain, x.ID, x.MinimumVersion, keys)
			cache[k] = f
		}
		x.Where = f.ref
		unknown = unknown || !f.ok
	}
	return unknown
}

func siteURL(domain string, r meta.Ref) string {
	switch strings.ToLower(r.Site) {
	case "nexus":
		return nexus.ModURL(domain, r.ID)
	case "curseforge":
		return "https://www.curseforge.com/projects/" + strconv.Itoa(r.ID)
	}
	return pageLink(domain, r)
}

// pageLink asks the ref's registered source for the mod page under the game's key for that source.
func pageLink(domain string, r meta.Ref) string {
	entry, ok := source.Get(r.Site)
	linker, canLink := entry.Source.(source.PageLinker)
	if !ok || !canLink {
		return ""
	}
	for _, g := range game.Catalog() {
		if g.NexusDomain() == domain {
			if gs, listed := g.Source(entry.Source.ID()); listed {
				return linker.ModPageURL(gs.Key, strconv.Itoa(r.ID))
			}
		}
	}
	return ""
}

// Locate names the page and file to get uniqueID from. ok is false when the dataset could not be read, as
// opposed to reading it and finding the mod unlisted (nil, true).
func Locate(ctx context.Context, m Meta, domain string, uniqueID mod.ID, minimum string, dependentKeys []string) (*Ref, bool) {
	refs, err := m.Lookup(ctx, uniqueID.Local())
	if err != nil {
		return nil, false
	}
	var nexus, other []meta.Ref
	for _, r := range refs {
		if strings.EqualFold(r.Site, "nexus") {
			nexus = append(nexus, r)
		} else if siteURL(domain, r) != "" {
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
		return &Ref{Site: other[0].Site, PageID: other[0].ID, URL: siteURL(domain, other[0])}, true
	}
	for _, k := range dependentKeys {
		if n, ok := manifest.NexusUpdateKey(k); ok {
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
		cand, top := fileIn(page, domain, r, uniqueID, minimum)
		if cand == nil {
			continue
		}
		if c, isOrdered := meta.CompareVersions(top, bestTop); best == nil || (isOrdered && c > 0) {
			best, bestTop = cand, top
		}
	}
	if best == nil {
		return &Ref{Site: "Nexus", PageID: nexus[0].ID, URL: siteURL(domain, nexus[0])}, ok
	}
	return best, ok
}

// githubRepo is the repository SMAPI's update API records for uniqueID, or "".
func githubRepo(ctx context.Context, m Meta, uniqueID mod.ID) string {
	got := m.CheckUpdates(ctx, meta.UpdateRequest{Mods: []meta.InstalledMod{{ID: uniqueID.Local()}}})
	if len(got) != 1 || !got[0].Known {
		return ""
	}
	if repo, ok := manifest.GitHubUpdateKey("github:" + got[0].GitHubRepo); ok {
		return repo
	}
	return ""
}

// fileIn picks the newest MAIN file of the page that holds uniqueID at a version meeting minimum. With none,
// it still returns the page when some file holds the mod, so its link is worth showing; otherwise nil.
// top is the highest version of the mod on the page, which ranks pages against each other.
func fileIn(page meta.Page, domain string, r meta.Ref, uniqueID mod.ID, minimum string) (ref *Ref, top string) {
	pageRef := Ref{Site: "Nexus", PageID: r.ID, PageName: page.Name, URL: cmp.Or(page.PageURL, siteURL(domain, r))}
	var best *Ref
	holds := false
	for _, f := range page.Downloads {
		for _, im := range f.Mods {
			if !mod.Equal(im.ModID(), uniqueID) {
				continue
			}
			if c, ok := meta.CompareVersions(im.Version, top); !holds || (ok && c > 0) {
				top = im.Version
			}
			holds = true
			if !strings.EqualFold(f.Type, "main") || !meets(im.Version, minimum) {
				continue
			}
			cand := pageRef
			cand.FileID, cand.FileName, cand.Version = f.ID, f.FileName, im.Version
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

// conflictRows counts non-cosmetic conflicts as the Problems tab shows them: conflicts between the same mods with the
// same winner and fix are one row however many assets they span.
func conflictRows(conflicts []framework.AssetConflict) int {
	rows := map[string]bool{}
	for _, c := range conflicts {
		if c.Cosmetic {
			continue
		}
		ids := mod.Strings(c.PackIDs)
		slices.Sort(ids)
		fixes, _ := json.Marshal(c.Fixes)
		rows[c.Kind+"\x00"+strings.Join(ids, "\x00")+"\x00"+c.WinnerName+"\x00"+string(fixes)] = true
	}
	return len(rows)
}
