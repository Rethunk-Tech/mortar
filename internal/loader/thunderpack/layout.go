package thunderpack

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// Of is the id of the first of loaders that is a thunderpack loader.
func Of(loaders []string) (string, bool) {
	for _, s := range specs {
		if slices.Contains(loaders, s.LoaderID) {
			return s.LoaderID, true
		}
	}
	return "", false
}

// Layout is where loader id puts each file of Thunderstore package pkg in the profile, by the file's path in the
// package; a file it drops has no entry. routes are the catalog's, for a loader whose rules are data.
func Layout(id string, files []string, pkg string, routes []components.LoaderRoute) map[string]string {
	for _, s := range specs {
		if s.LoaderID != id {
			continue
		}
		if s.Route != nil {
			return s.Route(files, pkg)
		}
		out := map[string]string{}
		for _, f := range files {
			if p, ok := components.PackageFile(f, pkg); ok {
				if dest := components.PlaceByRules(p, pkg, routes); dest != "" {
					out[f] = dest
				}
			}
		}
		return out
	}
	return nil
}

// perFile is a Route that places every file by a function of its cleaned path.
func perFile(dest func(p, pkg string) string) func(files []string, pkg string) map[string]string {
	return func(files []string, pkg string) map[string]string {
		out := map[string]string{}
		for _, f := range files {
			if p, ok := components.PackageFile(f, pkg); ok {
				out[f] = dest(p, pkg)
			}
		}
		return out
	}
}

// firstSegment splits p at its first slash; rest is "" for a file at the top.
func firstSegment(p string) (first, rest string) {
	first, rest, _ = strings.Cut(p, "/")
	return first, rest
}

// inFolder reports whether p is below a folder named dir at the top, and what is below it.
func inFolder(p, dir string) (rest string, ok bool) {
	first, rest := firstSegment(p)
	return rest, rest != "" && strings.EqualFold(first, dir)
}

// Zipped reports whether the first thunderpack loader of loaders reads its mods only as .zip files in the profile's
// mods folder, so a mod from any site is placed as one zip.
func Zipped(loaders []string) bool {
	id, ok := Of(loaders)
	return ok && slices.ContainsFunc(specs, func(s Spec) bool { return s.LoaderID == id && s.ZippedMods })
}
