package nexussvc

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/metadata"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/usererr"

	"github.com/Rethunk-Tech/go-semverish"
)

// pageFetches bounds the dataset pages read at once while telling candidate pages apart.
const pageFetches = 8

// FoundSources is the profile after FindSources, with how many entries got a Nexus page or file and how many
// source-less ones were left as they were.
type FoundSources struct {
	Profile profile.Profile `json:"profile"`
	Found   int             `json:"found"`
	Left    int             `json:"left"`
}

// FindSources gives each entry of the profile that has no source the page that publishes its mods on a site the game
// has a matcher for (game.FindSourceSites), so mods copied in from a folder or another manager get their page and
// update checks back.
func (s *Service) FindSources(ctx context.Context, gameID, profileID string) (FoundSources, error) {
	if s.Profiles == nil {
		return FoundSources{}, errors.New("profiles are unavailable")
	}
	p, err := s.Profiles.Get(gameID, profileID)
	if err != nil {
		return FoundSources{}, err
	}
	info, _ := components.Game(gameID)
	sites := game.FindSourceSites(info)
	out := FoundSources{Profile: p}
	if slices.Contains(sites, "nexus") {
		if out, err = s.findNexus(ctx, gameID, profileID, p); err != nil {
			return FoundSources{}, err
		}
	}
	more, err := s.findOnSites(ctx, gameID, profileID, out.Profile, sites)
	if err != nil {
		return FoundSources{}, err
	}
	out.Profile, out.Found, out.Left = more.Profile, out.Found+more.Found, more.Left+out.Left
	return out, nil
}

// findNexus gives each source-less entry the Nexus page that publishes its mods, and each entry on a Nexus page with
// no file recorded the file it was installed from, which a share needs. Where the mod dataset lists several pages for
// an entry (reuploads, translations), the page is taken only when the installed version and the author single one out.
func (s *Service) findNexus(ctx context.Context, gameID, profileID string, p profile.Profile) (FoundSources, error) {
	index, err := s.meta.Index(ctx)
	if err != nil {
		return FoundSources{}, err
	}
	candidates := map[string][]int{}
	var unsure []int
	left := 0
	for _, e := range p.Entries {
		if !e.Source.Unsourced() {
			continue
		}
		ids := candidatePages(e.Mods, index)
		if len(ids) == 0 {
			left++
			continue
		}
		candidates[e.Key] = ids
		if len(ids) > 1 {
			unsure = append(unsure, ids...)
		}
	}
	listed := s.datasetPages(ctx, unsure)
	pages := map[string]int{}
	var fileless []int
	for _, e := range p.Entries {
		ids := candidates[e.Key]
		if len(ids) > 1 {
			ids = narrowPages(e.Mods, ids, listed)
		}
		switch {
		case len(ids) == 1:
			pages[e.Key] = ids[0]
		case len(ids) > 1:
			left++
		case e.Source.Kind == profile.KindNexus && e.Source.ModID > 0 && e.Source.FileID == 0 && !e.IsOverlay():
			pages[e.Key] = e.Source.ModID
		}
		if id, ok := pages[e.Key]; ok {
			fileless = append(fileless, id)
		}
	}
	listed = s.datasetPages(ctx, fileless)
	found := map[string]profile.NexusFile{}
	for _, e := range p.Entries {
		id, ok := pages[e.Key]
		if !ok {
			continue
		}
		f := profile.NexusFile{ModID: id, FileID: installedFile(e.Mods, listed[id])}
		if e.Source.Unsourced() || f.FileID > 0 {
			found[e.Key] = f
		}
	}
	if len(found) > 0 {
		if p, err = s.Profiles.SetNexusFiles(gameID, profileID, found); err != nil {
			return FoundSources{}, err
		}
	}
	return FoundSources{Profile: p, Found: len(found), Left: left}, nil
}

// SetNexusPage records the Nexus page the player pasted the address of as the source of one entry, with the file
// the address names (file_id) or, failing that, the one the mod dataset matches to what is installed.
func (s *Service) SetNexusPage(ctx context.Context, gameID, profileID, key, link string) (profile.Profile, error) {
	if s.Profiles == nil {
		return profile.Profile{}, errors.New("profiles are unavailable")
	}
	domain, modID, ok := nexus.ParseModURL(link)
	if !ok {
		return profile.Profile{}, usererr.Wrap(usererr.Invalid, errors.New("that is not the address of a Nexus mod page"))
	}
	info, _ := components.Game(gameID)
	if !strings.EqualFold(info.NexusDomain(), domain) {
		return profile.Profile{}, usererr.Wrap(usererr.OtherGame, errors.New("that Nexus page is for another game"))
	}
	fileID := 0
	if u, err := url.Parse(strings.TrimSpace(link)); err == nil {
		fileID, _ = strconv.Atoi(u.Query().Get("file_id"))
	}
	if fileID <= 0 {
		fileID = 0
	}
	if fileID == 0 && info.HasMetadata(metadata.StardewDataset) {
		p, err := s.Profiles.Get(gameID, profileID)
		if err != nil {
			return profile.Profile{}, err
		}
		if i := slices.IndexFunc(p.Entries, func(e profile.Entry) bool { return e.Key == key }); i >= 0 {
			// A page the dataset does not list leaves the file unknown.
			if page, err := s.meta.Page(ctx, modID); err == nil {
				fileID = installedFile(p.Entries[i].Mods, page)
			}
		}
	}
	return s.Profiles.SetNexusFile(gameID, profileID, key, profile.NexusFile{ModID: modID, FileID: fileID})
}

// installedFile is the page's file whose mods are the listed ones at their installed versions, or 0 when no file
// or several fit equally. A file that holds a listed mod at another version is not it; of the rest, the one holding
// the most listed mods wins, then the one holding the fewest others.
func installedFile(mods []profile.Component, p meta.Page) int {
	best, bestHeld, bestOthers, tied := 0, 0, 0, false
	for _, f := range p.Downloads {
		held, differs := 0, false
		for _, published := range f.Mods {
			i := slices.IndexFunc(mods, func(m profile.Component) bool {
				return mod.Equal(m.ID, mod.SMAPI(published.UniqueID))
			})
			if i < 0 {
				continue
			}
			if !sameVersion(mods[i].Version, published.Version) {
				differs = true
				break
			}
			held++
		}
		if differs || held == 0 {
			continue
		}
		others := len(f.Mods) - held
		switch {
		case held > bestHeld || (held == bestHeld && others < bestOthers):
			best, bestHeld, bestOthers, tied = int(f.ID), held, others, false
		case held == bestHeld && others == bestOthers:
			tied = true
		}
	}
	if tied {
		return 0
	}
	return best
}

// sameVersion treats a manifest's "1.2" and the dataset's "1.2.0" as one version.
func sameVersion(a, b string) bool {
	if a == b {
		return true
	}
	c, ok := semverish.Compare(a, b)
	return ok && c == 0
}

// datasetPages reads the dataset's record of each page; one that cannot be read is left out.
func (s *Service) datasetPages(ctx context.Context, ids []int) map[int]meta.Page {
	slices.Sort(ids)
	ids = slices.Compact(ids)
	out := make(map[int]meta.Page, len(ids))
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, pageFetches)
	for _, id := range ids {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			page, err := s.meta.Page(ctx, id)
			if err != nil {
				return
			}
			mu.Lock()
			out[id] = page
			mu.Unlock()
		})
	}
	wg.Wait()
	return out
}

// candidatePages are the Nexus pages that publish every listed mod the index knows, lowest id first.
func candidatePages(mods []profile.Component, index map[string][]meta.Ref) []int {
	var pages []int
	known := false
	for _, m := range mods {
		if m.ID.Format() != mod.FormatSMAPI {
			continue
		}
		var own []int
		for _, r := range index[strings.ToLower(m.ID.Local())] {
			if strings.EqualFold(r.Site, "nexus") {
				own = append(own, r.ID)
			}
		}
		if len(own) == 0 {
			continue
		}
		if !known {
			pages, known = own, true
			continue
		}
		pages = slices.DeleteFunc(pages, func(id int) bool { return !slices.Contains(own, id) })
	}
	slices.Sort(pages)
	return pages
}

// narrowPages keeps the candidates that publish one of the mods at its installed version, then those by an author
// of one; a test no candidate passes is skipped. It returns ids unchanged when a candidate's page was not read,
// since the page left out could be the right one.
func narrowPages(mods []profile.Component, ids []int, listed map[int]meta.Page) []int {
	for _, id := range ids {
		if _, ok := listed[id]; !ok {
			return ids
		}
	}
	for _, passes := range []func(meta.Page) bool{
		func(p meta.Page) bool { return publishesInstalled(p, mods) },
		func(p meta.Page) bool { return writtenBy(p, mods) },
	} {
		var kept []int
		for _, id := range ids {
			if passes(listed[id]) {
				kept = append(kept, id)
			}
		}
		if len(kept) > 0 {
			ids = kept
		}
	}
	return ids
}

func publishesInstalled(p meta.Page, mods []profile.Component) bool {
	for _, f := range p.Downloads {
		for _, published := range f.Mods {
			if slices.ContainsFunc(mods, func(m profile.Component) bool {
				return mod.Equal(m.ID, mod.SMAPI(published.UniqueID)) && sameVersion(m.Version, published.Version)
			}) {
				return true
			}
		}
	}
	return false
}

// writtenBy matches names loosely, since a manifest's author is free text ("A, B (original author)") and a page's
// is an account name.
func writtenBy(p meta.Page, mods []profile.Component) bool {
	page := mod.LooseName(p.Author)
	if page == "" {
		return false
	}
	return slices.ContainsFunc(mods, func(m profile.Component) bool {
		author := mod.LooseName(m.Author)
		return author != "" && (strings.Contains(author, page) || strings.Contains(page, author))
	})
}
