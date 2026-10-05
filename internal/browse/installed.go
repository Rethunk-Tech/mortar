package browse

import (
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// Holdings is what a profile already has, as Browse cards and the "In this profile" filter see it: an entry's own
// source, an installed mod's manifest update keys, and the loader companion Mortar bundles.
type Holdings struct {
	nexus   map[int]bool
	github  map[string]bool
	pkg     map[string]bool
	bridges map[string]bool
}

// Hold reads prof's entries and the manifests of its installed mods (nil when unknown) for the catalog game info.
func Hold(info components.GameInfo, prof profile.Profile, installed []profile.Installed) Holdings {
	h := Holdings{nexus: map[int]bool{}, github: map[string]bool{}, pkg: map[string]bool{}, bridges: map[string]bool{}}
	bundled := false
	for _, e := range prof.Entries {
		switch e.Source.Kind {
		case profile.KindNexus:
			h.nexus[e.Source.ModID] = true
		case profile.KindThunderstore:
			h.pkg[strings.ToLower(e.Source.Name)] = true
		case profile.KindGitHub:
			h.github[strings.ToLower(e.Source.Repo)] = true
		case profile.SourceMortar:
			bundled = true
		}
	}
	for _, m := range installed {
		for _, k := range m.UpdateKeys {
			if id, ok := manifest.NexusUpdateKey(k); ok {
				h.nexus[id] = true
			}
			if repo, ok := manifest.GitHubUpdateKey(k); ok {
				h.github[strings.ToLower(repo)] = true
			}
		}
	}
	if bundled {
		h.addBridges(info)
	}
	return h
}

// addBridges notes the repositories of the game's loader companions, which Mortar installs itself.
func (h Holdings) addBridges(info components.GameInfo) {
	m, err := components.BundledManifest()
	if err != nil {
		return
	}
	for _, l := range info.Loaders {
		for _, c := range m.Components {
			if l.Companion != "" && c.Game == info.ID && c.Name == l.Companion && c.Source.Owner != "" {
				h.bridges[strings.ToLower(c.Source.Owner+"/"+c.Source.Repo)] = true
			}
		}
	}
}

// Has reports whether the profile has the mod, by entry source or update key.
func (h Holdings) Has(source, id string) bool {
	switch source {
	case "nexus":
		n, err := strconv.Atoi(id)
		return err == nil && h.nexus[n]
	case "thunderstore":
		return h.pkg[strings.ToLower(id)]
	case "github":
		return h.github[strings.ToLower(id)] || h.bridges[strings.ToLower(id)]
	}
	return false
}

// Bundled reports whether the mod is a loader companion Mortar installed itself.
func (h Holdings) Bundled(source, id string) bool {
	return source == "github" && h.bridges[strings.ToLower(id)]
}
