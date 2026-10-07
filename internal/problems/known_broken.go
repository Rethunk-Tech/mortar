package problems

import (
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// mortarSiteName is the Source of a row from the catalog's own known-broken list.
const mortarSiteName = "Mortar"

// knownBrokenMods lists the enabled mods and packages that the game's curated known-broken list in the signed
// catalog names, for a game whose sites publish no such flag.
func knownBrokenMods(gameID string, pkgs []profile.PackageRef, enabled []framework.Mod) []Broken {
	g, ok := components.Game(gameID)
	if !ok || len(g.KnownBroken) == 0 {
		return nil
	}
	return curatedBroken(g.KnownBroken, thunderstoreKey(gameID), pkgs, enabled)
}

func curatedBroken(list []components.KnownBroken, tsKey string, pkgs []profile.PackageRef, enabled []framework.Mod) []Broken {
	var out []Broken
	seen := map[string]bool{}
	add := func(key string, id mod.ID, name, version string, k components.KnownBroken) {
		named := strings.EqualFold(id.Local(), k.ID) || strings.EqualFold(name, k.ID)
		if seen[key] || !named || !inVersionRange(version, k.Versions) {
			return
		}
		seen[key] = true
		out = append(out, Broken{
			Key: key, ID: id, Name: name, Status: "broken", Source: mortarSiteName,
			Summary: k.Reason, Replacement: curatedReplacement(k.Replacement, tsKey),
		})
	}
	for _, k := range list {
		for _, p := range pkgs {
			add(p.Key, p.ID, p.Name, p.Version, k)
		}
		for _, m := range enabled {
			add(m.Key, m.ModID(), m.Name, m.Version, k)
		}
	}
	return out
}

// inVersionRange reports whether version meets every constraint of the range; an empty range holds for every version,
// and a version that cannot be ordered against a constraint is not flagged.
func inVersionRange(version, versions string) bool {
	for c := range strings.FieldsSeq(versions) {
		op := c[:1]
		if len(c) > 1 && (c[1] == '=') {
			op = c[:2]
		}
		got, ok := deps.Compare(deps.SemverStrict, version, strings.TrimPrefix(c, op))
		if !ok {
			return false
		}
		met := map[string]bool{"<": got < 0, "<=": got <= 0, ">": got > 0, ">=": got >= 0, "=": got == 0}[op]
		if !met {
			return false
		}
	}
	return true
}

// curatedReplacement points at a Thunderstore package, which is where the BepInEx games' packages come from.
func curatedReplacement(pkg, tsKey string) *Ref {
	if pkg == "" {
		return nil
	}
	ref := &Ref{Site: "Thunderstore", PageName: pkg}
	if e, ok := source.Get("thunderstore"); ok && tsKey != "" {
		if d, ok := e.Source.(interface {
			ModPageURL(gameKey, id string) string
		}); ok {
			ref.URL = d.ModPageURL(tsKey, pkg)
		}
	}
	return ref
}
