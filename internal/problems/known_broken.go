package problems

import (
	"context"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// StatusGameUnknown is the Broken status of a known-broken entry for some game versions when the install's version is
// unknown: shown, not counted.
const StatusGameUnknown = "game-unknown"

// mortarSiteName is the Source of a row from Mortar's own known-broken list.
const mortarSiteName = "Mortar"

// knownBrokenMods lists the enabled mods and packages that the game's known-broken list (Rethunk-Tech/mortar-known-broken)
// flags at their installed version on gameVersion.
func knownBrokenMods(ctx context.Context, gameID, gameVersion string, pkgs []profile.PackageRef, enabled []framework.Mod) []Broken {
	list := components.KnownBrokenFor(ctx, gameID)
	if len(list.Entries) == 0 {
		return nil
	}
	return curatedBroken(list, gameVersion, thunderstoreKey(gameID), pkgs, enabled)
}

func curatedBroken(list components.BrokenList, gameVersion, tsKey string, pkgs []profile.PackageRef, enabled []framework.Mod) []Broken {
	var out []Broken
	seen := map[string]bool{}
	add := func(key string, m components.BrokenMod, name string) {
		k, ok := list.Find(m, gameVersion)
		if seen[key] || !ok {
			return
		}
		seen[key] = true
		if gameVersion == "" && k.GameScoped() {
			out = append(out, Broken{Key: key, ID: m.ID, Name: name, Status: StatusGameUnknown, Source: mortarSiteName, Summary: k.Reason})
			return
		}
		out = append(out, Broken{
			Key: key, ID: m.ID, Name: name, Status: k.Status(), Source: mortarSiteName,
			Summary: k.Reason, Replacement: curatedReplacement(k.Replacement, tsKey),
		})
	}
	for _, p := range pkgs {
		add(p.Key, components.BrokenMod{ID: p.ID, Version: p.Version}, p.Name)
	}
	for _, m := range enabled {
		add(m.Key, components.BrokenMod{ID: m.ModID(), NexusID: nexusIDOf(m), Version: m.Version}, m.Name)
	}
	return out
}

// curatedReplacement points at a Thunderstore package, which is where the BepInEx games' packages come from.
func curatedReplacement(pkg, tsKey string) *Ref {
	if pkg == "" {
		return nil
	}
	ref := &Ref{Site: "thunderstore", Package: pkg, PageName: pkg}
	if _, name, ok := strings.Cut(pkg, "-"); ok && name != "" {
		ref.PageName = strings.ReplaceAll(name, "_", " ")
	}
	if e, ok := source.Get("thunderstore"); ok && tsKey != "" {
		if d, ok := e.Source.(interface {
			ModPageURL(gameKey, id string) string
		}); ok {
			ref.URL = d.ModPageURL(tsKey, pkg)
		}
	}
	return ref
}

// withKnownBroken marks each update whose new version the game's known-broken list still flags, or that leaves a
// flagged installed version behind.
func withKnownBroken(list components.BrokenList, gameVersion string, r UpdatesResult) UpdatesResult {
	for i, u := range r.Updates {
		m := components.BrokenMod{ID: u.ID, NexusID: u.NexusID, GitHub: u.GitHubRepo, Version: u.Version}
		if e, ok := list.Find(m, gameVersion); ok {
			r.Updates[i].KnownBroken = &source.KnownBrokenNote{Kind: e.Status(), Reason: e.Reason, Replacement: e.Replacement}
			continue
		}
		m.Version = u.Installed
		_, r.Updates[i].FixesKnownBroken = list.Find(m, gameVersion)
	}
	return r
}
