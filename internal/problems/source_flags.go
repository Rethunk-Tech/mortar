package problems

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const nexusSiteName = "Nexus Mods"

// sourceFlagged turns the sites' own flags on installed mods into Broken rows that name the site: Nexus's page status
// from the pages Mortar already holds, and the archived, abandoned or deleted flags Modrinth, CurseForge and GitHub
// publish, asked for in one cached batch per site.
func (s *Service) sourceFlagged(ctx context.Context, domain string, enabled []framework.Mod) []Broken {
	cl := &meta.Client{CacheDir: filepath.Join(s.home, "cache")}
	out := nexusFlagged(cl, domain, enabled)
	for _, kind := range []string{profile.KindModrinth, profile.KindCurseForge, profile.KindGitHub} {
		entry, ok := source.Get(kind)
		checker, can := entry.Source.(source.StandingChecker)
		if ok && can && source.Unavailable(entry.Source) == "" {
			out = append(out, s.siteFlagged(ctx, cl, kind, entry.Source.Name(), checker, enabled)...)
		}
	}
	return out
}

// nexusFlagged reads the status of each Nexus mod's page from the cache, never fetching: a page that is not published
// has no download to offer.
func nexusFlagged(cl *meta.Client, domain string, enabled []framework.Mod) []Broken {
	var out []Broken
	for _, im := range enabled {
		pageID, _, ok := nexusFile(im.Key)
		if !ok {
			continue
		}
		d, ok := nexussvc.PeekDetails(cl, domain, pageID)
		if !ok {
			continue
		}
		if state := nexusState(d.Page.Status, d.Page.Available); state != "" {
			out = append(out, flagRow(im, nexusSiteName, source.Standing{State: state}))
		}
	}
	return out
}

// nexusState words a Nexus page status other than published; a page with no status counts as published unless it
// says it is unavailable.
func nexusState(status string, available bool) string {
	switch status {
	case "published":
		return ""
	case "removed", "wastebinned":
		return "removed"
	case "hidden":
		return "hidden"
	case "under_moderation":
		return "moderated"
	case "not_published":
		return "unpublished"
	case "":
		if !available {
			return "unpublished"
		}
	}
	return ""
}

func (s *Service) siteFlagged(ctx context.Context, cl *meta.Client, kind, site string, checker source.StandingChecker, enabled []framework.Mod) []Broken {
	byID := map[string][]framework.Mod{}
	for _, im := range enabled {
		if im.SourceKind != kind {
			continue
		}
		id := im.SourceName
		if kind == profile.KindGitHub {
			id = im.SourceRepo
		}
		if id != "" {
			byID[id] = append(byID[id], im)
		}
	}
	if len(byID) == 0 {
		return nil
	}
	ids := slices.Sorted(maps.Keys(byID))
	sum := sha256.Sum256([]byte(strings.Join(ids, "\n")))
	flags, err := meta.Cached(cl, meta.StandingPrefix+kind+"-"+hex.EncodeToString(sum[:8])+".json", standingTTL, func() (map[string]source.Standing, error) {
		if s.Throttle != nil {
			release, err := s.Throttle(ctx, kind)
			if err != nil {
				return nil, err
			}
			defer release()
		}
		return checker.Standings(ctx, "", ids)
	})
	if err != nil {
		return nil
	}
	var out []Broken
	for _, id := range ids {
		if st, hit := flags[id]; hit {
			for _, im := range byID[id] {
				out = append(out, flagRow(im, site, st))
			}
		}
	}
	return out
}

func flagRow(im framework.Mod, site string, st source.Standing) Broken {
	return Broken{
		Key: im.Key, ID: im.ModID(), Name: im.Name, Status: st.State, Source: site,
		Summary: standingSummary(site, st),
	}
}

// standingSummary is the sentence a flag reads as, naming the site that raised it.
func standingSummary(site string, st source.Standing) string {
	if st.Reason != "" {
		return fmt.Sprintf("%s: %s", site, st.Reason)
	}
	switch st.State {
	case "archived":
		return site + " marks this project archived, so it is no longer updated."
	case "abandoned":
		return site + " marks this mod abandoned."
	case "inactive":
		return site + " marks this mod inactive."
	case "removed":
		return site + " has removed this mod."
	case "hidden":
		return site + " has hidden this mod's page."
	case "moderated":
		return site + " has this mod under moderation."
	case "unpublished":
		return site + " no longer publishes this mod."
	default:
		return site + " says this mod is no longer available."
	}
}
