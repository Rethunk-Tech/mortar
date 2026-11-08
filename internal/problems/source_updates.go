package problems

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"

	"github.com/Rethunk-Tech/go-semverish"
)

// sourceUpdates offers the newer version each of the game's sources lists for the mods, one source after another, each
// search taking its turn under the source's rate limit. Thunderstore also matches a mod installed elsewhere that it
// lists under the same GitHub repo or the same name and author; any other source answers for the mods installed from
// it. An update from a source other than the installed one is marked Switch. have holds the updates already found, so
// a version they cover is not offered twice.
func (s *Service) sourceUpdates(ctx context.Context, gameID string, mods []framework.Mod, have []Update) []Update {
	out := s.nexusPageUpdates(ctx, gameID, mods, have)
	out = append(out, s.githubUpdates(ctx, mods, append(slices.Clone(have), out...))...)
	twins := s.thunderstoreTwins(gameID)
	g, _ := components.Game(gameID)
	for _, src := range g.Sources {
		found, err := s.searchUpdates(ctx, gameID, src, mods, append(slices.Clone(have), out...), twins)
		out = append(out, found...)
		if err != nil {
			return out
		}
	}
	return out
}

// githubUpdates offers each GitHub-installed mod its repository's newest stable release with an archive, one repository
// at a time under GitHub's turn; the client keeps each answer an hour. A failed lookup ends the turn, as a search does.
func (s *Service) githubUpdates(ctx context.Context, mods []framework.Mod, have []Update) []Update {
	if s.GitHub == nil {
		return nil
	}
	var out []Update
	for _, x := range mods {
		owner, repo, ok := strings.Cut(x.SourceRepo, "/")
		if x.SourceKind != profile.KindGitHub || !ok || x.IgnoreUpdates {
			continue
		}
		release := func() {}
		if s.Throttle != nil {
			r, err := s.Throttle(ctx, profile.KindGitHub)
			if err != nil {
				return out
			}
			release = r
		}
		all, err := s.GitHub.Releases(ctx, owner, repo)
		release()
		if err != nil {
			return out
		}
		all = source.ReleasesWithin(all, source.CutoffOf(ctx))
		r, _, err := github.Select(all, "", profile.NormalizeChannel(x.UpdateChannel) == profile.ChannelBeta)
		if err != nil {
			continue
		}
		version := strings.TrimLeft(r.Tag, "vV")
		installed := strings.TrimLeft(cmp.Or(x.SourceVersion, x.Version), "vV")
		if c, ok := semverish.Compare(version, installed); !ok || c <= 0 || coveredBy(append(slices.Clone(have), out...), x.Key, version) {
			continue
		}
		out = append(out, Update{
			Key: x.Key, ID: x.ModID(), Name: x.Name, Installed: installed, Version: version,
			URL: cmp.Or(r.HTMLURL, "https://github.com/"+x.SourceRepo+"/releases/tag/"+r.Tag), GitHubRepo: x.SourceRepo, Source: "GitHub",
		})
	}
	return out
}

// thunderstoreTwins maps the key of each package not installed from Thunderstore to the Thunderstore package
// ("Namespace-Name") that ships a plugin with the same BepInPlugin GUID, read from the game's installed packages in
// every profile, so no request is made. A name search is left to match the rest.
func (s *Service) thunderstoreTwins(gameID string) map[string]string {
	if s.profiles == nil {
		return nil
	}
	return s.profiles.PluginTwins(gameID, pluginsIn)
}

// searchUpdates is one source's share of sourceUpdates. twins maps a package's key to its Thunderstore twin
// (thunderstoreTwins). The error is the first failed search, which ends the source's
// turn: a source that is down or out of quota is not asked again for every mod.
func (s *Service) searchUpdates(ctx context.Context, gameID string, src components.GameSource, mods []framework.Mod, have []Update, twins map[string]string) ([]Update, error) {
	entry, ok := source.Get(src.ID)
	if checker, canCheck := entry.Source.(source.UpdateChecker); ok && canCheck {
		return s.checkedUpdates(ctx, src, entry.Source, checker, mods, have)
	}
	searcher, canSearch := entry.Source.(source.Searcher)
	// GitHub and Nexus mods get their updates from SMAPI's API through their update keys (checkUpdates). A search per
	// installed mod would repeat that answer at one request each: hundreds per check against Nexus's hourly quota, and
	// a GitHub repository search lists no versions anyway.
	if src.Key == "" || !ok || !canSearch || src.ID == profile.KindGitHub || src.ID == profile.KindNexus {
		return nil, nil
	}
	ss := &sourceSearch{s: s, gameID: gameID, src: src, named: entry.Source, searcher: searcher, twins: twins}
	// A source that holds its whole listing answers for every package known by id in one pass. A search per mod would
	// read that listing once each, and its first page misses a package whose name many others share.
	if lister, lists := entry.Source.(source.ItemLister); lists {
		var ids []string
		for _, x := range mods {
			if id := ss.listedID(x); id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			var err error
			if ss.listed, err = s.throttledItems(ctx, lister, src, ids); err != nil {
				return nil, err
			}
		}
	}
	var out []Update
	var installedRefs []source.VersionRef
	for _, x := range mods {
		if (profile.Source{Kind: x.SourceKind}).Bundled() || x.IgnoreUpdates || !asksSource(src.ID, x) {
			continue
		}
		installed, text, twin := ss.query(x)
		if text == "" {
			continue
		}
		items, err := ss.items(ctx, x, text)
		if err != nil {
			return out, err
		}
		if u, ref, ok := ss.newer(ctx, x, items, twin, installed, append(slices.Clone(have), out...)); ok {
			installedRefs = append(installedRefs, ref)
			out = append(out, u)
		}
	}
	s.markDependencyChanges(ctx, src, entry.Source, out, installedRefs)
	return out, nil
}

// sourceSearch is one source's lookups for searchUpdates.
type sourceSearch struct {
	s        *Service
	gameID   string
	src      components.GameSource
	named    source.Source
	searcher source.Searcher
	twins    map[string]string
	listed   map[string]source.Item
}

func (ss *sourceSearch) listedID(x framework.Mod) string {
	if x.SourceKind == ss.src.ID {
		return x.SourceName
	}
	if ss.src.ID == profile.KindThunderstore {
		return ss.twins[x.Key]
	}
	return ""
}

// query is the installed version, the text to search for and the Thunderstore twin of a mod.
func (ss *sourceSearch) query(x framework.Mod) (installed, text, twin string) {
	text, installed = x.Name, x.Version
	if x.SourceKind == ss.src.ID {
		installed = cmp.Or(x.SourceVersion, x.Version)
	}
	if x.SourceKind == profile.KindThunderstore && ss.src.ID == profile.KindThunderstore {
		_, text, _ = strings.Cut(x.SourceName, "-")
	}
	if ss.src.ID == profile.KindThunderstore && x.SourceKind != profile.KindThunderstore {
		twin = ss.twins[x.Key]
	}
	if twin != "" {
		_, text, _ = strings.Cut(twin, "-")
	}
	return installed, text, twin
}

func (ss *sourceSearch) items(ctx context.Context, x framework.Mod, text string) ([]source.Item, error) {
	if id := ss.listedID(x); ss.listed != nil && id != "" {
		if it, ok := ss.listed[strings.ToLower(id)]; ok {
			return []source.Item{it}, nil
		}
		return nil, nil
	}
	page, err := ss.s.throttledSearch(ctx, ss.searcher, ss.gameID, ss.src, text)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// newer is the update from the first item that is the mod, when that item is newer than the installed version and no
// update already found covers it.
func (ss *sourceSearch) newer(ctx context.Context, x framework.Mod, items []source.Item, twin, installed string, known []Update) (Update, source.VersionRef, bool) {
	for _, it := range items {
		if !sameMod(x, it) && (twin == "" || !strings.EqualFold(it.ID, twin)) {
			continue
		}
		version := it.Version
		if d, ok := ss.named.(thunderstore.Driver); ok && !source.CutoffOf(ctx).IsZero() {
			ns, name, _ := strings.Cut(it.ID, "-")
			if r, err := d.Resolve(ctx, ss.src.Key, ns, name, "", ""); err == nil {
				version = r.Version
			}
		}
		if c, ok := semverish.Compare(version, installed); !ok || c <= 0 || coveredBy(known, x.Key, version) {
			break
		}
		ref := source.VersionRef{}
		if x.SourceKind == ss.src.ID {
			ref = source.VersionRef{ID: x.SourceName, Version: installed}
		}
		u := Update{
			Key: x.Key, ID: x.ModID(), Name: x.Name, Installed: installed, Version: version, URL: it.URL,
			Source: ss.named.Name(), Switch: x.SourceKind != ss.src.ID,
		}
		switch ss.src.ID {
		case profile.KindThunderstore:
			u.Package = it.ID
		case profile.KindNexus:
			u.NexusID, _ = strconv.Atoi(it.ID)
		case profile.KindVSModDB, profile.KindParadox:
			u.Package, u.PackageSource = it.ID, ss.src.ID
		}
		return u, ref, true
	}
	return Update{}, source.VersionRef{}, false
}

// checkedUpdates is searchUpdates for a source that checks every installed file in one request: the mods installed from
// it with a known file digest.
func (s *Service) checkedUpdates(ctx context.Context, src components.GameSource, named source.Source, checker source.UpdateChecker, mods []framework.Mod, have []Update) ([]Update, error) {
	var asked []framework.Mod
	var files []source.InstalledFile
	for _, x := range mods {
		if x.SourceKind != src.ID || x.SourceDigest == "" || x.IgnoreUpdates {
			continue
		}
		asked = append(asked, x)
		files = append(files, source.InstalledFile{ID: x.SourceName, Version: cmp.Or(x.SourceVersion, x.Version), Digest: x.SourceDigest, Channel: x.UpdateChannel})
	}
	if len(files) == 0 {
		return nil, nil
	}
	if s.Throttle != nil {
		release, err := s.Throttle(ctx, src.ID)
		if err != nil {
			return nil, err
		}
		defer release()
	}
	latest, err := checker.Latest(ctx, src, "", files)
	if err != nil {
		return nil, err
	}
	var out []Update
	var installedRefs []source.VersionRef
	for i, x := range asked {
		l, ok := latest[x.SourceDigest]
		if !ok || l.Version == files[i].Version || coveredBy(append(slices.Clone(have), out...), x.Key, l.Version) {
			continue
		}
		installedRefs = append(installedRefs, source.VersionRef{ID: l.ProjectID, Version: files[i].Version})
		out = append(out, Update{
			Key: x.Key, ID: x.ModID(), Name: x.Name, Installed: files[i].Version, Version: l.Version, URL: l.URL,
			Source: named.Name(), Package: l.ProjectID, PackageSource: src.ID, PackageVersion: l.VersionID,
		})
	}
	s.markDependencyChanges(ctx, src, named, out, installedRefs)
	return out, nil
}

// markDependencyChanges sets each update's added and removed dependencies from the source's listing, in one lookup for
// all of them. A source without dependency data, or an update from another source than the installed one, gets none.
func (s *Service) markDependencyChanges(ctx context.Context, src components.GameSource, entry source.Source, updates []Update, installed []source.VersionRef) {
	lister, ok := entry.(source.DependencyLister)
	if !ok || len(updates) == 0 {
		return
	}
	refs := make([]source.VersionRef, 0, 2*len(updates))
	for i, u := range updates {
		if installed[i] != (source.VersionRef{}) {
			refs = append(refs, installed[i], source.VersionRef{ID: u.Package, Version: u.Version})
		}
	}
	listed, err := lister.Dependencies(ctx, src.Key, "", refs)
	if err != nil {
		return
	}
	for i := range updates {
		u := &updates[i]
		before, hadBefore := listed[installed[i]]
		after, hadAfter := listed[source.VersionRef{ID: u.Package, Version: u.Version}]
		if installed[i] != (source.VersionRef{}) && hadBefore && hadAfter {
			u.AddedDeps, u.RemovedDeps = depDiff(before, after)
		}
	}
}

// depDiff is the packages in after but not before, and in before but not after, each sorted and compared by id alone.
func depDiff(before, after []string) (added, removed []string) {
	has := func(list []string, id string) bool {
		return slices.ContainsFunc(list, func(o string) bool { return strings.EqualFold(o, id) })
	}
	for _, id := range after {
		if !has(before, id) {
			added = append(added, id)
		}
	}
	for _, id := range before {
		if !has(after, id) {
			removed = append(removed, id)
		}
	}
	slices.Sort(added)
	slices.Sort(removed)
	return added, removed
}

// throttledItems waits for the source's slot before reading its listing.
func (s *Service) throttledItems(ctx context.Context, lister source.ItemLister, src components.GameSource, ids []string) (map[string]source.Item, error) {
	if s.Throttle != nil {
		release, err := s.Throttle(ctx, src.ID)
		if err != nil {
			return nil, err
		}
		defer release()
	}
	return lister.Items(ctx, src.Key, "", ids)
}

// throttledSearch waits for the source's slot before asking it.
func (s *Service) throttledSearch(ctx context.Context, searcher source.Searcher, gameID string, src components.GameSource, text string) (source.Page, error) {
	if s.Throttle != nil {
		release, err := s.Throttle(ctx, src.ID)
		if err != nil {
			return source.Page{}, err
		}
		defer release()
	}
	return searcher.Search(ctx, source.Query{Game: gameID, Key: src.Key, Text: text, Page: source.FirstPage})
}

// asksSource reports whether the mod is looked up on the source: Thunderstore knows mods from anywhere by repo or
// name, every other source only the mods installed from it.
func asksSource(id string, x framework.Mod) bool {
	return id == profile.KindThunderstore || x.SourceKind == id
}

func sameMod(x framework.Mod, it source.Item) bool {
	switch {
	case x.SourceKind == profile.KindThunderstore:
		return strings.EqualFold(it.ID, x.SourceName)
	case x.SourceKind == profile.KindNexus:
		page, _, ok := nexusFile(x.Key)
		return ok && it.ID == strconv.Itoa(page)
	case x.SourceRepo != "" && it.Repo != "":
		return strings.EqualFold(it.Repo, x.SourceRepo)
	}
	return mod.LooseName(it.Name) != "" && mod.LooseName(it.Name) == mod.LooseName(x.Name) && mod.LooseName(it.Author) != "" && mod.LooseName(it.Author) == mod.LooseName(x.Author)
}

func coveredBy(have []Update, key, version string) bool {
	for _, u := range have {
		if itemOf(u.Key) != itemOf(key) {
			continue
		}
		if c, ok := semverish.Compare(u.Version, version); ok && c >= 0 {
			return true
		}
	}
	return false
}

// NexusPagesOf looks up the page of many Nexus mods of the game in one batched call.
type NexusPagesOf func(ctx context.Context, gameID string, modIDs []int) (map[int]nexusmods.Page, error)

// Requirements reads the requirements listed on the game's pages from the same batched page data; nil when p is.
func (p NexusPagesOf) Requirements(gameID string) RequirementsOf {
	if p == nil {
		return nil
	}
	return func(ctx context.Context, modIDs []int) (map[int][]nexusmods.Requirement, error) {
		pages, err := p(ctx, gameID, modIDs)
		out := make(map[int][]nexusmods.Requirement, len(pages))
		for id, page := range pages {
			out[id] = page.Requirements
		}
		return out, err
	}
}

// nexusPageUpdates offers the version a Nexus mod's page lists for the mods installed from Nexus that carry no update
// key, so SMAPI's API cannot answer for them (every mod of a game with another loader): one batched lookup for all of
// them, at a handful of requests per hundred mods. A mod is offered only when the loader's version scheme orders both
// versions and the page's is strictly newer; a pair it cannot order is not guessed at.
func (s *Service) nexusPageUpdates(ctx context.Context, gameID string, mods []framework.Mod, have []Update) []Update {
	title, err := game.NexusTitle(gameID)
	if s.NexusPages == nil || err != nil {
		return nil
	}
	var ids []int
	byID := map[int][]framework.Mod{}
	for _, x := range mods {
		if x.SourceKind != profile.KindNexus || x.SourceModID <= 0 || len(x.UpdateKeys) > 0 || x.IgnoreUpdates || (profile.Source{Kind: x.SourceKind}).Bundled() {
			continue
		}
		if len(byID[x.SourceModID]) == 0 {
			ids = append(ids, x.SourceModID)
		}
		byID[x.SourceModID] = append(byID[x.SourceModID], x)
	}
	if len(ids) == 0 {
		return nil
	}
	scheme := game.VersionScheme(gameID)
	pages, _ := s.NexusPages(ctx, gameID, ids)
	var out []Update
	for _, id := range ids {
		page, ok := pages[id]
		if !ok || !page.Available || page.Version == "" {
			continue
		}
		for _, x := range byID[id] {
			installed := cmp.Or(x.SourceVersion, x.Version)
			if c, ok := deps.Compare(scheme, page.Version, installed); !ok || c <= 0 || coveredBy(append(slices.Clone(have), out...), x.Key, page.Version) {
				continue
			}
			out = append(out, Update{
				Key: x.Key, ID: x.ModID(), Name: x.Name, Installed: installed, Version: page.Version,
				URL: nexusmods.ModURL(title.Domain, id), NexusID: id, Source: "Nexus",
			})
		}
	}
	return out
}
