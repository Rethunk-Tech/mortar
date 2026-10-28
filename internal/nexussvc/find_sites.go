package nexussvc

import (
	"context"
	"maps"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/vsmoddb"
)

// modDBResolver finds a mod on the Vintage Story Mod DB by the mod id its modinfo.json names.
type modDBResolver interface {
	Resolve(ctx context.Context, id, version string) (vsmoddb.Resolved, error)
}

// findOnSites gives the source-less entries of p their Thunderstore package (a BepInEx plugin GUID the game's other
// installed Thunderstore packages also ship) and Mod DB page (a modinfo.json mod id), where the game has those matchers.
func (s *Service) findOnSites(ctx context.Context, gameID, profileID string, p profile.Profile, sites []string) (FoundSources, error) {
	found := map[string]profile.Source{}
	if slices.Contains(sites, "thunderstore") {
		maps.Copy(found, thunderstoreSources(p.Entries, s.Profiles.PluginTwins(gameID, func(dir string) []dotnet.Plugin { return dotnet.ScanDir(dir).Plugins })))
	}
	if slices.Contains(sites, "vsmoddb") {
		maps.Copy(found, modDBSources(ctx, p.Entries, vsmoddb.Driver{}))
	}
	left := 0
	for _, e := range p.Entries {
		if _, ok := found[e.Key]; !ok && e.Source.Unsourced() && len(sites) > 0 && !slices.Equal(sites, []string{"nexus"}) {
			left++
		}
	}
	if len(found) == 0 {
		return FoundSources{Profile: p, Left: left}, nil
	}
	p, err := s.Profiles.SetSources(gameID, profileID, found)
	return FoundSources{Profile: p, Found: len(found), Left: left}, err
}

// thunderstoreSources is the Thunderstore source of each source-less package entry whose key twins lists.
func thunderstoreSources(entries []profile.Entry, twins map[string]string) map[string]profile.Source {
	out := map[string]profile.Source{}
	for _, e := range entries {
		if name, ok := twins[e.Key]; ok && e.Package && e.Source.Unsourced() {
			out[e.Key] = profile.Source{Kind: profile.KindThunderstore, Name: name}
		}
	}
	return out
}

// modDBSources is the Mod DB source of each source-less entry whose mods name one mod id on the site. An entry whose
// mods resolve to several pages, or whose id the site does not know, stays as it is.
func modDBSources(ctx context.Context, entries []profile.Entry, r modDBResolver) map[string]profile.Source {
	out := map[string]profile.Source{}
	for _, e := range entries {
		if !e.Source.Unsourced() {
			continue
		}
		var pick vsmoddb.Resolved
		ok := true
		for _, m := range e.Mods {
			if m.ModID == "" {
				continue
			}
			got, err := r.Resolve(ctx, m.ModID, m.Version)
			if err != nil {
				got, err = r.Resolve(ctx, m.ModID, "")
				got.ReleaseID = 0
			}
			if err != nil || pick.ID != "" && pick.ID != got.ID {
				ok = false
				break
			}
			pick = got
		}
		if ok && pick.ID != "" {
			out[e.Key] = profile.Source{Kind: profile.KindVSModDB, Name: pick.ID, FileID: pick.ReleaseID, ModName: pick.Name, Picture: pick.Picture}
		}
	}
	return out
}
