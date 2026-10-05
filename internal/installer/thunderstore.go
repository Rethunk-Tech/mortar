package installer

import (
	"encoding/json"
	"io/fs"
	"slices"

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
	var m struct {
		Name    string `json:"name"`
		Version string `json:"version_number"`
	}
	if json.Unmarshal(b, &m) != nil || m.Name == "" || m.Version == "" {
		return "", false
	}
	return m.Name, true
}

func (thunderstoreRules) Detect(a Archive, g Game) bool {
	_, ok := manifestOf(a)
	return ok && slices.Contains(g.Loaders, bepinex5.ID)
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
		if dest := bepinex5.Route(f, pkg); dest != "" {
			l.Files = append(l.Files, File{Src: f, Target: TargetProfile, Rel: dest})
		}
	}
	return l, validate(l, g)
}
