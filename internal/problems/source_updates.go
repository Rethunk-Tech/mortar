package problems

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func fold(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// sourceUpdates offers the newer version each of the game's sources lists for the mods, one source after another, each
// search taking its turn under the source's rate limit. Thunderstore also matches a mod installed elsewhere that it
// lists under the same GitHub repo or the same name and author; any other source answers for the mods installed from
// it. An update from a source other than the installed one is marked Switch. have holds the updates already found, so
// a version they cover is not offered twice.
func (s *Service) sourceUpdates(ctx context.Context, gameID string, mods []framework.Mod, have []Update) []Update {
	out := s.nexusPageUpdates(ctx, gameID, mods, have)
	for _, g := range game.Catalog() {
		if g.ID != gameID {
			continue
		}
		for _, src := range g.Sources {
			found, err := s.searchUpdates(ctx, gameID, src, mods, append(slices.Clone(have), out...))
			out = append(out, found...)
			if err != nil {
				return out
			}
		}
	}
	return out
}

// searchUpdates is one source's share of sourceUpdates. The error is the first failed search, which ends the source's
// turn: a source that is down or out of quota is not asked again for every mod.
func (s *Service) searchUpdates(ctx context.Context, gameID string, src components.GameSource, mods []framework.Mod, have []Update) ([]Update, error) {
	entry, ok := source.Get(src.ID)
	searcher, canSearch := entry.Source.(source.Searcher)
	// GitHub and Nexus mods get their updates from SMAPI's API through their update keys (checkUpdates). A search per
	// installed mod would repeat that answer at one request each: hundreds per check against Nexus's hourly quota, and
	// a GitHub repository search lists no versions anyway.
	if src.Key == "" || !ok || !canSearch || src.ID == profile.KindGitHub || src.ID == profile.KindNexus {
		return nil, nil
	}
	var out []Update
	var installedRefs []source.VersionRef
	for _, x := range mods {
		if (profile.Source{Kind: x.SourceKind}).Bundled() || x.IgnoreUpdates || !asksSource(src.ID, x) {
			continue
		}
		text, installed := x.Name, x.Version
		if x.SourceKind == src.ID {
			installed = cmp.Or(x.SourceVersion, x.Version)
		}
		if x.SourceKind == profile.KindThunderstore && src.ID == profile.KindThunderstore {
			_, text, _ = strings.Cut(x.SourceName, "-")
		}
		if text == "" {
			continue
		}
		page, err := s.throttledSearch(ctx, searcher, gameID, src, text)
		if err != nil {
			return out, err
		}
		for _, it := range page.Items {
			if !sameMod(x, it) {
				continue
			}
			if c, ok := meta.CompareVersions(it.Version, installed); !ok || c <= 0 {
				break
			}
			if coveredBy(append(slices.Clone(have), out...), x.Key, it.Version) {
				break
			}
			ref := source.VersionRef{}
			if x.SourceKind == src.ID {
				ref = source.VersionRef{ID: x.SourceName, Version: installed}
			}
			installedRefs = append(installedRefs, ref)
			u := Update{
				Key: x.Key, ID: x.ModID(), Name: x.Name, Installed: installed, Version: it.Version, URL: it.URL,
				Source: entry.Source.Name(), Switch: x.SourceKind != src.ID,
			}
			switch src.ID {
			case profile.KindThunderstore:
				u.Package = it.ID
			case profile.KindNexus:
				u.NexusID, _ = strconv.Atoi(it.ID)
			}
			out = append(out, u)
			break
		}
	}
	s.markDependencyChanges(ctx, src, entry.Source, out, installedRefs)
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
		page, _, ok := store.NexusFile(x.Key)
		return ok && it.ID == strconv.Itoa(page)
	case x.SourceRepo != "" && it.Repo != "":
		return strings.EqualFold(it.Repo, x.SourceRepo)
	}
	return fold(it.Name) != "" && fold(it.Name) == fold(x.Name) && fold(it.Author) != "" && fold(it.Author) == fold(x.Author)
}

func coveredBy(have []Update, key, version string) bool {
	for _, u := range have {
		if u.Key != key {
			continue
		}
		if c, ok := meta.CompareVersions(u.Version, version); ok && c >= 0 {
			return true
		}
	}
	return false
}

// NexusPagesOf looks up the page of many Nexus mods of the game in one batched call.
type NexusPagesOf func(ctx context.Context, gameID string, modIDs []int) (map[int]nexus.Page, error)

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
	scheme := versionScheme(gameID)
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
				URL: "https://www.nexusmods.com/" + title.Domain + "/mods/" + strconv.Itoa(id), NexusID: id, Source: "Nexus",
			})
		}
	}
	return out
}
