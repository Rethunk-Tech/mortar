package problems

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
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
	var out []Update
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
	if src.Key == "" || !ok || !canSearch {
		return nil, nil
	}
	var out []Update
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
			u := Update{
				Key: x.Key, ID: x.ModID(), Name: x.Name, Installed: installed, Version: it.Version, URL: it.URL,
				Source: entry.Source.Name(), Switch: x.SourceKind != src.ID,
			}
			switch src.ID {
			case profile.KindThunderstore:
				u.Package = it.ID
			case profile.KindNexus:
				u.NexusID, _ = strconv.Atoi(it.ID)
			case profile.KindGitHub:
				u.GitHubRepo = it.Repo
			}
			out = append(out, u)
			break
		}
	}
	return out, nil
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
