package installer

import (
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
)

var _ = Register(thunderstoreRules{})

// thunderstoreRules places a Thunderstore package's files by BepInEx's rules, below the profile's root.
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

// Detect takes a Thunderstore package, and for a BepInEx game also an archive from another site that holds no
// manifest but plainly is a BepInEx mod: a DLL or a BepInEx folder. r2modman installs those by the same rules.
func (thunderstoreRules) Detect(a Archive, g Game) bool {
	if !slices.Contains(g.Loaders, bepinex5.ID) {
		return false
	}
	if _, ok := manifestOf(a); ok {
		return true
	}
	all, err := files(a, ".")
	return err == nil && slices.ContainsFunc(all, func(f string) bool {
		first, _, _ := strings.Cut(f, "/")
		return !skip(f) && (strings.EqualFold(path.Ext(f), ".dll") || strings.EqualFold(first, "BepInEx"))
	})
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
	var l Layout
	for _, f := range all {
		dest := bepinex5.Route(f, pkg)
		if dest == "" {
			continue
		}
		l.Files = append(l.Files, File{Src: f, Target: TargetProfile, Rel: dest})
	}
	return l, validate(l, g)
}
