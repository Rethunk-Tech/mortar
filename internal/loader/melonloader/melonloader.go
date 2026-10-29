// Package melonloader is the MelonLoader driver. MelonLoader keeps its files, mods and settings in a base folder the
// game is told to use (--melonloader.basedir), so a profile is that folder; only the version.dll proxy sits in the
// game's install, placed for a launch.
package melonloader

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/components"
)

// ID is the catalog's id for this loader.
const ID = "melonloader"

const (
	proxyDLL   = "version.dll"
	loaderDir  = "MelonLoader"
	markerFile = ".mortar-melonloader.json"
	// generatedIl2Cpp and generatedMono are where MelonLoader writes the game's assemblies on its first start.
	generatedIl2Cpp = "Il2CppAssemblies"
	generatedMono   = "Managed"
)

// Route is where a file of Thunderstore package pkg (Namespace-Name) goes in the profile root, from its path relPath
// inside the package, or "" when the file is package metadata or unsafe. It follows r2modman's MelonLoader rules.
// A game whose catalog entry lists routes keeps the classic layout (components.PlaceByRules). A game with no routes
// runs MelonLoader 0.7's recursive layout: a top-level UserData folder goes to UserData/<package>, anything else to
// Mods/<package> with its structure.
func Route(relPath, pkg string, routes []components.LoaderRoute) string {
	p, ok := components.PackageFile(relPath, pkg)
	if !ok {
		return ""
	}
	if len(routes) > 0 {
		return components.PlaceByRules(p, pkg, routes)
	}
	if segs := strings.Split(p, "/"); len(segs) > 1 && strings.EqualFold(segs[0], "UserData") {
		return path.Join("UserData", pkg, path.Join(segs[1:]...))
	}
	return path.Join("Mods", pkg, p)
}

// packRoot is the folder of the extracted release that holds the proxy and the loader folder: dir itself, or the one
// folder below it that does.
func packRoot(dir string) (string, error) {
	holds := func(d string) bool {
		return safefs.IsDir(filepath.Join(d, loaderDir)) && safefs.IsFile(filepath.Join(d, proxyDLL))
	}
	if holds(dir) {
		return dir, nil
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if e.IsDir() && holds(filepath.Join(dir, e.Name())) {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", errors.New("the zip has no " + loaderDir + " folder with " + proxyDLL + " beside it")
}

// InstallPack lays the MelonLoader release zip out into profileRoot, replacing the loader folder and the proxy; the
// profile's Mods, Plugins, UserData and UserLibs stay.
func InstallPack(zipPath, profileRoot string) error {
	if err := os.MkdirAll(profileRoot, 0o750); err != nil {
		return err
	}
	// Beside the profile, so the final moves stay on one file system.
	tmp, err := os.MkdirTemp(filepath.Dir(profileRoot), ".melonloader-pack-*")
	if err != nil {
		return err
	}
	defer func() { _ = safefs.RemoveAll(tmp) }()
	if err := archive.Extract(zipPath, tmp); err != nil {
		return err
	}
	src, err := packRoot(tmp)
	if err != nil {
		return err
	}
	for _, name := range []string{loaderDir, proxyDLL} {
		if err := safefs.RemoveAll(filepath.Join(profileRoot, name)); err != nil {
			return err
		}
		if err := safefs.Rename(filepath.Join(src, name), filepath.Join(profileRoot, name)); err != nil {
			return err
		}
	}
	return nil
}

// generated reports whether MelonLoader already wrote the game's assemblies into the profile; until it has, the next
// start must ask it to (--melonloader.agfregenerate), as r2modman does.
func generated(profileRoot string) bool {
	for _, d := range []string{generatedIl2Cpp, generatedMono} {
		if _, err := os.Stat(filepath.Join(profileRoot, loaderDir, d, "Assembly-CSharp.dll")); err == nil {
			return true
		}
	}
	return false
}
