package browse

import (
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// Holdings is what a profile already has, as Browse cards and the "In this profile" filter see it: an entry's own
// source, an installed mod's manifest update keys, and the loader companion Mortar bundles.
type Holdings struct {
	nexus  map[int]bool
	github map[string]bool
	// named holds the entries of sources identified by Name (Thunderstore, Modrinth, itch.io, ...), by kind.
	named   map[string]map[string]bool
	bridges map[string]bool
	// ids maps a source ref to the package identity the profile's entry or installed manifest gives it.
	ids map[string]string
}

// Hold reads prof's entries and the manifests of its installed mods (nil when unknown) for the catalog game info.
func Hold(info components.GameInfo, prof profile.Profile, installed []profile.Installed) Holdings {
	h := Holdings{nexus: map[int]bool{}, github: map[string]bool{}, named: map[string]map[string]bool{}, bridges: map[string]bool{}, ids: map[string]string{}}
	bundled := false
	for _, e := range prof.Entries {
		// An entry of several mods names no single identity.
		if len(e.Mods) == 1 {
			ref := e.Source.Name
			switch e.Source.Kind {
			case profile.KindNexus:
				ref = strconv.Itoa(e.Source.ModID)
			case profile.KindGitHub:
				ref = e.Source.Repo
			}
			h.identify(e.Source.Kind, ref, e.Mods[0].ID)
		}
		switch e.Source.Kind {
		case profile.KindNexus:
			h.nexus[e.Source.ModID] = true
		case profile.KindGitHub:
			h.github[strings.ToLower(e.Source.Repo)] = true
		case profile.SourceMortar:
			bundled = true
		case profile.KindLocal:
		default:
			h.hold(e.Source.Kind, e.Source.Name)
		}
	}
	for _, m := range installed {
		for _, k := range m.UpdateKeys {
			if id, ok := manifest.NexusUpdateKey(k); ok {
				h.nexus[id] = true
				h.identify(profile.KindNexus, strconv.Itoa(id), mod.SMAPI(m.UniqueID))
			}
			if repo, ok := manifest.GitHubUpdateKey(k); ok {
				h.github[strings.ToLower(repo)] = true
				h.identify(profile.KindGitHub, repo, mod.SMAPI(m.UniqueID))
			}
		}
	}
	if bundled {
		h.addBridges(info)
	}
	return h
}

// identify records that the source's ref is the package id; a ref two packages claim names none.
func (h Holdings) identify(kind, ref string, id mod.ID) {
	if kind == "" || ref == "" || id == "" {
		return
	}
	key := kind + "|" + strings.ToLower(ref)
	if prev, ok := h.ids[key]; ok && prev != id.Fold() {
		id = ""
	}
	h.ids[key] = id.Fold()
}

// Identity is the package identity the profile gives a hit's source and id, "" when it holds no such mod.
func (h Holdings) Identity(source, id string) string {
	return h.ids[source+"|"+strings.ToLower(id)]
}

func (h Holdings) hold(kind, name string) {
	if h.named[kind] == nil {
		h.named[kind] = map[string]bool{}
	}
	h.named[kind][strings.ToLower(name)] = true
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
	case "github":
		return h.github[strings.ToLower(id)] || h.bridges[strings.ToLower(id)]
	}
	return h.named[source][strings.ToLower(id)]
}

// Bundled reports whether the mod is a loader companion Mortar installed itself.
func (h Holdings) Bundled(source, id string) bool {
	return source == "github" && h.bridges[strings.ToLower(id)]
}
