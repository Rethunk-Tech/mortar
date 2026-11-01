package devsvc

import (
	"slices"

	"github.com/Rethunk-Tech/go-safefs"
)

// BuildOrder groups the projects in dirs into waves for building them all: each wave holds the projects whose
// dependencies among dirs (the packages another of them publishes) are built in an earlier wave, so a wave's projects
// can build at once, each compiling against what the waves before it wrote. Projects in a dependency cycle go in the
// last wave together, as they would build one at a time anyway.
func (s *Service) BuildOrder(dirs []string) [][]string {
	publishes := map[string]string{}
	for _, d := range dirs {
		if pkg := packageOf(d); pkg != "" {
			publishes[pkg] = d
		}
	}
	needs := map[string][]string{}
	for _, d := range dirs {
		csproj, ok := csprojOf(d)
		if !ok {
			continue
		}
		raw, err := safefs.ReadFile(csproj)
		if err != nil {
			continue
		}
		for pkg := range projectDependencies(string(raw)) {
			if dep, ok := publishes[pkg]; ok && dep != d {
				needs[d] = append(needs[d], dep)
			}
		}
	}
	waves := [][]string{}
	built := map[string]bool{}
	left := slices.Clone(dirs)
	for len(left) > 0 {
		var wave, rest []string
		for _, d := range left {
			if !slices.ContainsFunc(needs[d], func(dep string) bool { return !built[dep] }) {
				wave = append(wave, d)
			} else {
				rest = append(rest, d)
			}
		}
		if len(wave) == 0 {
			return append(waves, rest)
		}
		for _, d := range wave {
			built[d] = true
		}
		waves = append(waves, wave)
		left = rest
	}
	return waves
}
