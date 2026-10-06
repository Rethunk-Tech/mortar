package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// bepinexMembers is what one plugin DLL changes in Lethal Company: the game members it assigns, and the game methods
// it takes over: a Harmony prefix that can skip the original or a HookGen hook, which match each other, or a transpiler
// that rewrites it. A prefix or postfix that only adds to a method says nothing about the job, since many mods hook the
// same busy methods. Game types sit in the global namespace and GameNetcodeStuff.
func bepinexMembers(dll string) []string {
	var out []string
	for _, ns := range []string{"", "GameNetcodeStuff"} {
		w, _ := dotnet.Writes(dll, ns, "Unity.Netcode")
		out = append(out, w...)
	}
	patches, _ := dotnet.Patches(dll)
	for _, p := range patches {
		switch p.Kind {
		case dotnet.PatchSkip, dotnet.PatchHook:
			out = append(out, "harmony:"+p.Target+" (skip or hook)")
		case dotnet.PatchTranspiler:
			out = append(out, "harmony:"+p.Target+" (transpiler)")
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// packageFootprints maps each enabled package mod, by folded mod id, to what its plugin DLLs change. A package's
// Thunderstore dependencies are not in its manifest, so the returned mods carry, as dependencies, the packages whose
// plugins theirs declare a BepInDependency on or ship the same GUID as (a duplicate a plugin clash already reports):
// sameJob leaves such pairs, and any package others build on, alone.
func packageFootprints(pkgs []profile.PackageRef, mods []framework.Mod) (map[string]map[string]bool, []framework.Mod) {
	byKey := map[string]int{}
	for i, m := range mods {
		if m.Enabled && m.EntryDll == "" && !(profile.Source{Kind: m.SourceKind}).Bundled() {
			byKey[m.Key] = i
		}
	}
	owners := map[string][]string{}
	declared := map[string]dotnet.Declared{}
	var dlls, keys []string
	for _, p := range pkgs {
		if _, ok := byKey[p.Key]; !ok {
			continue
		}
		d := dotnet.ScanDir(p.Dir)
		declared[p.Key] = d
		for _, pl := range d.Plugins {
			g := strings.ToLower(pl.GUID)
			owners[g] = append(owners[g], p.Key)
		}
		for _, dll := range dotnet.DLLs(p.Dir) {
			dlls, keys = append(dlls, dll), append(keys, p.Key)
		}
	}
	out := map[string]map[string]bool{}
	for i, members := range assemblyMembers(dlls, bepinexMembers) {
		id := mods[byKey[keys[i]]].ModID().Fold()
		for _, member := range members {
			if out[id] == nil {
				out[id] = map[string]bool{}
			}
			out[id][member] = true
		}
	}
	mods = slices.Clone(mods)
	for key, d := range declared {
		m := &mods[byKey[key]]
		m.Dependencies = slices.Clone(m.Dependencies)
		related := map[string]bool{}
		for _, r := range d.Relations {
			if r.Kind != dotnet.Incompatible {
				related[strings.ToLower(r.GUID)] = true
			}
		}
		for _, pl := range d.Plugins {
			related[strings.ToLower(pl.GUID)] = true
		}
		for g := range related {
			for _, other := range owners[g] {
				if other == key {
					continue
				}
				oid := mods[byKey[other]].ModID()
				m.Dependencies = append(m.Dependencies, manifest.Dependency{UniqueID: oid.Local(), Format: oid.Format(), Required: true})
			}
		}
	}
	return out, mods
}
