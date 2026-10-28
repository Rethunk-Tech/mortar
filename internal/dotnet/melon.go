package dotnet

import (
	"cmp"
	"encoding/binary"
	"math"
	"slices"
	"strings"
)

// Melon is what a MelonLoader mod or plugin assembly says about itself: its [assembly: MelonInfo], [MelonPriority],
// [MelonAdditionalDependencies] and [MelonOptionalDependencies] attributes (MelonLoader/Attributes), and the assemblies
// it references, which MelonLoader treats as dependencies too.
type Melon struct {
	// Assembly is the assembly's simple name, which other melons name as a dependency.
	Assembly              string
	Name, Version, Author string
	// Priority is MelonPriority; lower loads first.
	Priority                   int
	Refs, Additional, Optional []string
	HasInfo                    bool
	// Cycle is a melon MelonLoader leaves out because its dependencies depend on it.
	Cycle bool
}

// ScanMelon reads the MelonLoader attributes of the assembly at path; HasInfo is false for a library that is no melon.
func ScanMelon(path string) (Melon, error) {
	return withModule(path, scanMelon)
}

func scanMelon(m *module) (out Melon, err error) {
	if m.rows[tAssembly] > 0 {
		out.Assembly = m.str(m.cell(tAssembly, 1, 7))
	}
	for row := 1; row <= m.rows[tAssemblyRef]; row++ {
		if name := m.str(m.cell(tAssemblyRef, row, 6)); !slices.Contains(out.Refs, name) {
			out.Refs = append(out.Refs, name)
		}
	}
	for a := range m.attributes("MelonLoader") {
		if a.owner == tAssembly {
			out.readAttribute(a.name, m.value(a))
		}
	}
	return out, nil
}

func (mel *Melon) readAttribute(name string, blob []byte) {
	if len(blob) < 2 || blob[0] != 1 || blob[1] != 0 {
		return
	}
	blob = blob[2:]
	switch name {
	case "MelonInfo":
		// MelonInfo(Type, name, version, author, downloadLink): the type is serialized as its name string.
		var args [4]string
		_, rest, ok := serString(blob)
		for i := range args {
			if !ok {
				return
			}
			args[i], rest, ok = serString(rest)
		}
		mel.Name, mel.Version, mel.Author, mel.HasInfo = args[0], args[1], args[2], true
	case "MelonPriority":
		if len(blob) >= 4 {
			// The blob is a signed 32-bit integer.
			u := binary.LittleEndian.Uint32(blob)
			mel.Priority = int(u&math.MaxInt32) - int(u>>31)<<31
		}
	case "MelonAdditionalDependencies":
		mel.Additional = append(mel.Additional, stringArray(blob)...)
	case "MelonOptionalDependencies":
		mel.Optional = append(mel.Optional, stringArray(blob)...)
	}
}

// stringArray reads a params string[] argument: a 4-byte count (all ones for null), then each string.
func stringArray(b []byte) []string {
	if len(b) < 4 {
		return nil
	}
	n := binary.LittleEndian.Uint32(b)
	if n == 0xFFFFFFFF || n > maxRelations {
		return nil
	}
	b = b[4:]
	var out []string
	for range n {
		s, rest, ok := serString(b)
		if !ok {
			break
		}
		out, b = append(out, s), rest
	}
	return out
}

// MelonOrder is the order MelonLoader registers melons of one kind (plugins, or mods) in (MelonBase.SortMelons): a
// topological sort in which a melon follows the melons whose assemblies it references or names in
// MelonAdditionalDependencies, the loadable melon with the smallest name going first, then a stable sort by priority.
// A melon in a dependency cycle is not loaded; the ones MelonLoader reports are returned last with Cycle set.
//
// ponytail: names compare lower-cased ordinal where MelonLoader's SortedList uses the current culture.
func MelonOrder(melons []Melon) []Melon {
	index := map[string]int{}
	for i, m := range melons {
		index[m.Assembly] = i
	}
	deps := make([][]int, len(melons))
	dependents := make([][]int, len(melons))
	for i, m := range melons {
		for _, name := range slices.Concat(m.Refs, m.Additional) {
			if j, ok := index[name]; ok && j != i && !slices.Contains(deps[i], j) {
				deps[i] = append(deps[i], j)
				dependents[j] = append(dependents[j], i)
			}
		}
	}
	waiting := make([]int, len(melons))
	var ready []int
	for i := range melons {
		if waiting[i] = len(deps[i]); waiting[i] == 0 {
			ready = append(ready, i)
		}
	}
	var order []Melon
	done := make([]bool, len(melons))
	for len(ready) > 0 {
		slices.SortStableFunc(ready, func(a, b int) int {
			return cmp.Or(strings.Compare(strings.ToLower(melons[a].Name), strings.ToLower(melons[b].Name)), a-b)
		})
		next := ready[0]
		ready = ready[1:]
		order, done[next] = append(order, melons[next]), true
		for _, d := range dependents[next] {
			if waiting[d]--; waiting[d] == 0 {
				ready = append(ready, d)
			}
		}
	}
	for i, m := range melons {
		if !done[i] {
			m.Cycle = true
			order = append(order, m)
		}
	}
	slices.SortStableFunc(order, func(a, b Melon) int {
		if a.Cycle != b.Cycle {
			if a.Cycle {
				return 1
			}
			return -1
		}
		return cmp.Compare(a.Priority, b.Priority)
	})
	return order
}
