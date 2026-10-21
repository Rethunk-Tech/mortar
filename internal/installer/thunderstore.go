package installer

import (
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/loader/melonloader"
	"github.com/Rethunk-Tech/mortar/internal/loader/thunderpack"
)

var _ = Register(thunderstoreRules{})

// thunderstoreRules places a Thunderstore package's files by the rules of the game's loader (BepInEx, MelonLoader or
// one of the thunderpack loaders), below the profile's root.
type thunderstoreRules struct{}

func (thunderstoreRules) ID() string { return "thunderstore-rules" }

func manifestOf(a Archive) (name string, ok bool) {
	b, err := fs.ReadFile(a.FS, "manifest.json")
	if err != nil {
		return "", false
	}
	m, err := bepinex5.ParseManifest(b)
	if err != nil || m.Name == "" || m.Version == "" {
		return "", false
	}
	return m.Name, true
}

// Detect takes a Thunderstore package, and for a BepInEx or MelonLoader game also an archive from another site that
// holds no manifest but plainly is a mod of the loader: a DLL or the loader's folder. r2modman installs those by the
// same rules. A thunderpack game takes only a Thunderstore package.
func (thunderstoreRules) Detect(a Archive, g Game) bool {
	loaderFolder := ""
	if _, ok := thunderpack.Of(g.Loaders); ok {
		_, isPackage := manifestOf(a)
		return isPackage
	}
	switch {
	case slices.Contains(g.Loaders, bepinex5.ID):
		loaderFolder = "BepInEx"
	case slices.Contains(g.Loaders, melonloader.ID):
		loaderFolder = "MelonLoader"
	default:
		return false
	}
	if _, ok := manifestOf(a); ok {
		return true
	}
	all, err := files(a, ".")
	return err == nil && slices.ContainsFunc(all, func(f string) bool {
		first, _, _ := strings.Cut(f, "/")
		return !skip(f) && (strings.EqualFold(path.Ext(f), ".dll") || strings.EqualFold(first, loaderFolder))
	})
}

// router is where the game's loader puts each file of package pkg, "" for a file it drops.
func router(g Game, all []string, pkg string) func(f string) string {
	if id, ok := thunderpack.Of(g.Loaders); ok {
		dests := thunderpack.Layout(id, all, pkg, g.Routes)
		return func(f string) string { return dests[f] }
	}
	if slices.Contains(g.Loaders, melonloader.ID) {
		return func(f string) string { return melonloader.Route(f, pkg, g.Routes) }
	}
	return func(f string) string { return bepinex5.Route(f, pkg, g.Routes) }
}

func (thunderstoreRules) Layout(a Archive, g Game, _ Choices) (Layout, error) {
	pkg := a.Key
	if pkg == "" {
		pkg, _ = manifestOf(a)
	}
	all, err := files(a, ".")
	if err != nil {
		return Layout{}, err
	}
	// Flattening can send two files to one place; as in r2modman, a folder's own files are placed before its
	// subfolders' and the last one placed wins. Windows and Wine ignore case, so neither may two names differing only in case.
	slices.SortStableFunc(all, filesFirst)
	var l Layout
	at := map[string]int{}
	route := router(g, all, pkg)
	for _, f := range all {
		dest := route(f)
		if dest == "" || skip(f) {
			continue
		}
		file := File{Src: f, Target: TargetProfile, Rel: dest}
		if i, ok := at[strings.ToLower(dest)]; ok {
			l.Files[i] = file
			continue
		}
		at[strings.ToLower(dest)] = len(l.Files)
		l.Files = append(l.Files, file)
	}
	return l, validate(l, g)
}

// filesFirst orders slash paths as a walk that lists a folder's files before its subfolders.
func filesFirst(a, b string) int {
	as, bs := strings.Split(a, "/"), strings.Split(b, "/")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] == bs[i] {
			continue
		}
		if aFile, bFile := i == len(as)-1, i == len(bs)-1; aFile != bFile {
			if aFile {
				return -1
			}
			return 1
		}
		return strings.Compare(as[i], bs[i])
	}
	return len(as) - len(bs)
}
