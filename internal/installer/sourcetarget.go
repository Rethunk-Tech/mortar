package installer

import (
	"path"
	"slices"
)

var _ = Register(sourceTarget{})

// DriverSourceTarget is the id of the driver that lays an archive out whole in the target that lists its source.
const DriverSourceTarget = "source-target"

// sourceTarget lays out an archive from a site whose mods the game loads from a folder of its own (Paradox Mods
// archives for Cities: Skylines II), in the target that names the site, below a folder named for the mod. The game's
// loader has no say, so this runs before any driver that reads the archive's contents.
type sourceTarget struct{}

func (sourceTarget) ID() string { return DriverSourceTarget }

// TargetFor is the shared target of g that takes archives from source.
func (g Game) TargetFor(source string) (Target, bool) {
	if source == "" {
		return Target{}, false
	}
	i := slices.IndexFunc(g.Targets, func(t Target) bool { return t.Shared && slices.Contains(t.Sources, source) })
	if i < 0 {
		return Target{}, false
	}
	return g.Targets[i], true
}

func (sourceTarget) Detect(a Archive, g Game) bool {
	_, ok := g.TargetFor(a.Source)
	return ok && a.Package != ""
}

func (sourceTarget) Layout(a Archive, g Game, _ Choices) (Layout, error) {
	t, _ := g.TargetFor(a.Source)
	all, err := files(a, ".")
	if err != nil {
		return Layout{}, err
	}
	folder := a.Source + "-" + a.Package
	var l Layout
	for _, f := range all {
		if !skip(f) {
			l.Files = append(l.Files, File{Src: f, Target: t.ID, Rel: path.Join(folder, f)})
		}
	}
	return l, validate(l, g)
}
