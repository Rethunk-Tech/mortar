package profile

import (
	"slices"
)

// CLICompare is the user-mod comparison exposed by the command line.
type CLICompare struct {
	OnlyA            []DiffSide `json:"onlyA"`
	OnlyB            []DiffSide `json:"onlyB"`
	DifferentVersion []DiffPair `json:"differentVersion"`
	DifferentEnabled []DiffPair `json:"differentEnabled"`
	// DifferentSource lists mods both profiles hold, from different sources (a Nexus and a Thunderstore copy).
	DifferentSource []DiffPair `json:"differentSource"`
	Identical       []DiffPair `json:"identical"`
}

// CompareProfilesCLI compares user mods by case-insensitive mod id.
func CompareProfilesCLI(a, b Profile) CLICompare {
	left, right := indexUserMods(a), indexUserMods(b)
	out := CLICompare{
		OnlyA:            []DiffSide{},
		OnlyB:            []DiffSide{},
		DifferentVersion: []DiffPair{},
		DifferentEnabled: []DiffPair{},
		DifferentSource:  []DiffPair{},
		Identical:        []DiffPair{},
	}
	for key, side := range left {
		other, ok := right[key]
		if !ok {
			out.OnlyA = append(out.OnlyA, side)
			continue
		}
		name := side.Name
		if name == "" {
			name = other.Name
		}
		pair := DiffPair{ID: side.ID, Name: name, A: side, B: other}
		versionDiff := side.Version != other.Version
		enabledDiff := side.Enabled != other.Enabled
		if versionDiff {
			out.DifferentVersion = append(out.DifferentVersion, pair)
		}
		if enabledDiff {
			out.DifferentEnabled = append(out.DifferentEnabled, pair)
		}
		sourceDiff := side.Source.Kind != other.Source.Kind
		if sourceDiff {
			out.DifferentSource = append(out.DifferentSource, pair)
		}
		if !versionDiff && !enabledDiff && !sourceDiff {
			out.Identical = append(out.Identical, pair)
		}
	}
	for key, side := range right {
		if _, ok := left[key]; !ok {
			out.OnlyB = append(out.OnlyB, side)
		}
	}
	sortSides(out.OnlyA)
	sortSides(out.OnlyB)
	sortPairs := func(pairs []DiffPair) {
		slices.SortFunc(pairs, func(a, b DiffPair) int {
			return compareNameThenID(a.Name, a.ID, b.Name, b.ID)
		})
	}
	sortPairs(out.DifferentVersion)
	sortPairs(out.DifferentEnabled)
	sortPairs(out.DifferentSource)
	sortPairs(out.Identical)
	return out
}
