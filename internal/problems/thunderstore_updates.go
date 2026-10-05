package problems

import (
	"context"
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const thunderstoreSite = "Thunderstore"

func fold(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// thunderstoreUpdates offers the newer Thunderstore version of each mod that is on Thunderstore: a package installed
// from it, or a mod installed elsewhere that Thunderstore lists under the same GitHub repo or the same name and
// author. An update from a source other than the installed one is marked Switch. have holds the updates already
// found, so a version they cover is not offered twice.
func (s *Service) thunderstoreUpdates(ctx context.Context, gameID string, mods []Installed, have []Update) []Update {
	key := ""
	for _, g := range game.Catalog() {
		if g.ID != gameID {
			continue
		}
		if src, ok := g.Source("thunderstore"); ok {
			key = src.Key
		}
	}
	entry, ok := source.Get("thunderstore")
	searcher, canSearch := entry.Source.(source.Searcher)
	if key == "" || !ok || !canSearch {
		return nil
	}
	var out []Update
	for _, x := range mods {
		if (profile.Source{Kind: x.SourceKind}).Bundled() || x.IgnoreUpdates {
			continue
		}
		text, installed := x.Name, x.Version
		if x.SourceKind == profile.KindThunderstore {
			_, text, _ = strings.Cut(x.SourceName, "-")
			installed = x.SourceVersion
		}
		if text == "" {
			continue
		}
		page, err := searcher.Search(ctx, source.Query{Game: gameID, Key: key, Text: text, Page: source.FirstPage})
		if err != nil {
			return out
		}
		for _, it := range page.Items {
			if !sameMod(x, it) {
				continue
			}
			if c, ok := meta.CompareVersions(it.Version, installed); !ok || c <= 0 {
				break
			}
			if coveredBy(have, x.Key, it.Version) {
				break
			}
			out = append(out, Update{
				Key: x.Key, ID: x.ModID(), Name: x.Name, Installed: installed, Version: it.Version, URL: it.URL,
				Source: thunderstoreSite, Package: it.ID, Switch: x.SourceKind != profile.KindThunderstore,
			})
			break
		}
	}
	return out
}

func sameMod(x Installed, it source.Item) bool {
	switch {
	case x.SourceKind == profile.KindThunderstore:
		return strings.EqualFold(it.ID, x.SourceName)
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
