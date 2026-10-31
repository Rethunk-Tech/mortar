// Package bepinex5 holds the mechanics of the BepInEx 5 mod loader that do not depend on any game or profile model: laying out the
// BepInExPack, the Doorstop proxy files and launch flags, the Wine DLL override a Proton prefix needs, and where a Thunderstore
// package's files go.
package bepinex5

import (
	"path"
	"strings"

	tstore "github.com/Rethunk-Tech/go-thunderstore"

	"github.com/Rethunk-Tech/go-bepinex"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// defaultDoorstop is the Doorstop major of a pack that names none.
const defaultDoorstop = 3

// targetPath is the profile's entry assembly as Doorstop is given it.
func targetPath(profileRoot string, proton bool) string {
	return bepinex.HostPath(profileRoot, bepinex.Entry(profileRoot), proton)
}

// Route is where a file of Thunderstore package pkg (Namespace-Name) goes in the profile root, from its path relPath
// inside the package zip, or "" when the path is unsafe. It follows r2modman's BepInEx rules
// (r2modmanPlus src/installers/InstallRulePluginInstaller.ts, buildInstallForRuleSubtype), which mod authors build
// and test against: the first folder on the path named plugins, patchers, monomod, core or config, at any depth, is
// that rule's folder and keeps the structure below it (plugins, patchers, monomod and core in a folder named for the
// package, config flat). A file under no such folder is placed by its name alone, a .mm.dll in monomod and anything
// else in plugins (the package's own manifest.json and icon.png too, which in-game mod menus read), so the folders
// above it are flattened away: packages such as MirageCore ship FSharp.Core/FSharp.Core.dll
// and load it from beside their plugin. The game's routes (components.LoaderRoute) add folders of the same kind, each
// matched by its last segment at any depth, and extensions that send a file there wherever it sits.
func Route(relPath, pkg string, routes []components.LoaderRoute) string {
	if pkg == "" || strings.ContainsAny(pkg, `/\:`) || pkg == "." || pkg == ".." {
		return ""
	}
	p := path.Clean(strings.ReplaceAll(relPath, `\`, "/"))
	if p == "." || strings.HasPrefix(p, "/") || p == ".." || strings.HasPrefix(p, "../") || strings.Contains(p, ":") {
		return ""
	}
	segs := strings.Split(p, "/")
	for i, dir := range segs[:len(segs)-1] {
		rest := path.Join(segs[i+1:]...)
		if r, ok := components.RouteByFolder(routes, dir); ok {
			return r.Place(pkg, rest, p)
		}
		switch strings.ToLower(dir) {
		case "config":
			return path.Join("BepInEx", "config", rest)
		case "plugins", "patchers", "monomod", "core":
			return path.Join("BepInEx", strings.ToLower(dir), pkg, rest)
		}
	}
	name := segs[len(segs)-1]
	if r, ok := components.RouteByExtension(routes, name); ok {
		return r.Place(pkg, name, p)
	}
	if strings.HasSuffix(strings.ToLower(name), ".mm.dll") {
		return path.Join("BepInEx", "monomod", pkg, name)
	}
	return path.Join("BepInEx", "plugins", pkg, name)
}

// Needs is the packages the manifest depends on, each at least the version it names. The BepInEx pack is left out:
// the loader install supplies it, so no profile entry ever stands for it.
func Needs(m tstore.Manifest) []mod.Dependency {
	var out []mod.Dependency
	for _, s := range m.Dependencies {
		d, err := deps.Thunderstore(s)
		if err != nil || components.IsLoaderPackage(mod.ID(d.Target.Package).Local()) {
			continue
		}
		out = append(out, mod.NewDependency(mod.ID(d.Target.Package), d.Constraint, true))
	}
	return out
}
