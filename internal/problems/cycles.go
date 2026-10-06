package problems

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// dependencyCycles finds enabled SMAPI mods that wait for each other. SMAPI's ModResolver orders a mod after every
// installed dependency, required or optional, and after its ContentPackFor; when that walk comes back to a mod it is
// still ordering, it fails that mod with "circular reference". A mod that requires a failed one fails too, so a loop
// of only required links loads none of its mods, while an optional link lets the mods before it load. One row per
// loop names its shortest path, starting from the first mod by name.
func dependencyCycles(enabled []framework.Mod) []Broken {
	index := map[string]int{}
	var nodes []framework.Mod
	for _, m := range enabled {
		id := m.ModID()
		if id.Format() != mod.FormatSMAPI || m.UniqueID == "" {
			continue
		}
		if _, dup := index[id.Fold()]; !dup {
			index[id.Fold()] = len(nodes)
			nodes = append(nodes, m)
		}
	}
	g := make(graph, len(nodes))
	for i, m := range nodes {
		for _, d := range m.Dependencies {
			if j, ok := index[d.ModID().Fold()]; ok {
				g.link(i, j, d.Required)
			}
		}
		if cp := m.ContentPackForID(); cp != "" {
			if j, ok := index[cp.Fold()]; ok {
				g.link(i, j, true)
			}
		}
	}
	var out []Broken
	for _, l := range g.loops(func(i int) string { return nodes[i].Name }) {
		loop := make([]Dependent, len(l.path))
		for k, i := range l.path {
			loop[k] = Dependent{Key: nodes[i].Key, ID: nodes[i].ModID(), Name: nodes[i].Name}
		}
		first := nodes[l.path[0]]
		out = append(out, Broken{
			Key: first.Key, ID: first.ModID(), Name: first.Name, Status: "cycle",
			Cycle: loop, CycleBlocksAll: l.blocksAll || len(loop) == 1, Summary: cycleSentence(loop, l.blocksAll),
		})
	}
	slices.SortFunc(out, func(a, b Broken) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return out
}

// pluginCycles finds enabled BepInEx plugins that depend on each other. BepInEx 5's chainloader sorts every plugin it
// will load after each [BepInDependency], hard or soft, matching GUIDs without regard to case; a loop makes that sort
// throw "Cyclic Dependency" and the chainloader stops with "Error occurred starting the game", so no plugin loads at
// all. A plugin it drops before sorting (an older copy of a GUID, or one incompatible with a loaded plugin) adds no
// links. declared is what each enabled package's DLLs declare, by package key. One row per loop, at its first package.
func pluginCycles(pkgs []profile.PackageRef, declared map[string]dotnet.Declared) []Broken {
	type plugin struct {
		pkg           profile.PackageRef
		name, version string
	}
	index := map[string]int{}
	var nodes []plugin
	var relations [][]dotnet.Relation
	for _, p := range pkgs {
		if !p.Enabled {
			continue
		}
		d := declared[p.Key]
		for _, pl := range d.Plugins {
			g := strings.ToLower(pl.GUID)
			if g == "" {
				continue
			}
			var own []dotnet.Relation
			for _, r := range d.Relations {
				if strings.EqualFold(r.Plugin, pl.GUID) {
					own = append(own, r)
				}
			}
			node := plugin{p, cmp.Or(pl.Name, p.Name), pl.Version}
			if i, dup := index[g]; dup {
				if c, ok := meta.CompareVersions(pl.Version, nodes[i].version); ok && c > 0 {
					nodes[i], relations[i] = node, own
				}
				continue
			}
			index[g] = len(nodes)
			nodes = append(nodes, node)
			relations = append(relations, own)
		}
	}
	g := make(graph, len(nodes))
	for i, rs := range relations {
		if slices.ContainsFunc(rs, func(r dotnet.Relation) bool {
			_, present := index[strings.ToLower(r.GUID)]
			return r.Kind == dotnet.Incompatible && present
		}) {
			continue
		}
		for _, r := range rs {
			if j, ok := index[strings.ToLower(r.GUID)]; ok && r.Kind != dotnet.Incompatible {
				g.link(i, j, true)
			}
		}
	}
	var out []Broken
	for _, l := range g.loops(func(i int) string { return nodes[i].name }) {
		loop := make([]Dependent, len(l.path))
		for k, i := range l.path {
			loop[k] = Dependent{Key: nodes[i].pkg.Key, ID: nodes[i].pkg.ID, Name: nodes[i].name}
		}
		first := nodes[l.path[0]].pkg
		out = append(out, Broken{
			Key: first.Key, ID: first.ID, Name: first.Name, Status: "cycle",
			Cycle: loop, CycleBlocksAll: true, CycleStopsLoader: true, Summary: pluginCycleSentence(loop),
		})
	}
	return out
}

// edge is a link from a mod to one it waits for; required is false for an optional dependency.
type edge struct {
	to       int
	required bool
}

// graph is each node's links, by node index.
type graph [][]edge

// link adds i → j, keeping one link per pair that is required when either way of naming it is.
func (g graph) link(i, j int, required bool) {
	if k := slices.IndexFunc(g[i], func(e edge) bool { return e.to == j }); k >= 0 {
		g[i][k].required = g[i][k].required || required
		return
	}
	g[i] = append(g[i], edge{to: j, required: required})
}

func (g graph) successors(i int) []int {
	out := make([]int, len(g[i]))
	for k, e := range g[i] {
		out[k] = e.to
	}
	return out
}

// loop is a shortest path from a node back to itself, without the closing node, and whether its every link is required.
type loop struct {
	path      []int
	blocksAll bool
}

// loops is one loop per group of nodes that wait for each other, starting from the group's first node by name.
func (g graph) loops(name func(int) string) []loop {
	var out []loop
	for _, scc := range stronglyConnected(len(g), g.successors) {
		start := slices.MinFunc(scc, func(a, b int) int {
			return strings.Compare(strings.ToLower(name(a)), strings.ToLower(name(b)))
		})
		if len(scc) == 1 && !slices.Contains(g.successors(start), start) {
			continue
		}
		path := shortestLoop(start, scc, g.successors)
		blocksAll := true
		for k, i := range path {
			next := path[(k+1)%len(path)]
			blocksAll = blocksAll && g[i][slices.IndexFunc(g[i], func(e edge) bool { return e.to == next })].required
		}
		out = append(out, loop{path, blocksAll})
	}
	return out
}

func pluginCycleSentence(loop []Dependent) string {
	const outcome = "so BepInEx stops before loading any plugin"
	switch len(loop) {
	case 1:
		return fmt.Sprintf("%s lists itself as a dependency, %s", loop[0].Name, outcome)
	case 2:
		return fmt.Sprintf("%s and %s each depend on the other, %s", loop[0].Name, loop[1].Name, outcome)
	}
	names := make([]string, 0, len(loop)+1)
	for _, d := range loop {
		names = append(names, d.Name)
	}
	return fmt.Sprintf("%s depend on each other in a loop, %s", strings.Join(append(names, loop[0].Name), " → "), outcome)
}

func cycleSentence(loop []Dependent, blocksAll bool) string {
	if len(loop) == 1 {
		return fmt.Sprintf("%s lists itself as a dependency, so SMAPI skips it", loop[0].Name)
	}
	outcome := "SMAPI skips at least one of them"
	if blocksAll {
		outcome = "SMAPI loads none of them"
	}
	if len(loop) == 2 {
		if blocksAll {
			outcome = "SMAPI loads neither"
		}
		return fmt.Sprintf("%s and %s each wait for the other, so %s", loop[0].Name, loop[1].Name, outcome)
	}
	names := make([]string, 0, len(loop)+1)
	for _, d := range loop {
		names = append(names, d.Name)
	}
	return fmt.Sprintf("%s wait for each other in a loop, so %s", strings.Join(append(names, loop[0].Name), " → "), outcome)
}

// shortestLoop is the shortest path from start back to itself inside scc, without the closing start.
func shortestLoop(start int, scc []int, successors func(int) []int) []int {
	in := map[int]bool{}
	for _, i := range scc {
		in[i] = true
	}
	prev := map[int]int{}
	queue := []int{start}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		for _, j := range successors(i) {
			if !in[j] {
				continue
			}
			if j == start {
				path := []int{i}
				for path[0] != start {
					path = append([]int{prev[path[0]]}, path...)
				}
				return path
			}
			if _, seen := prev[j]; !seen {
				prev[j] = i
				queue = append(queue, j)
			}
		}
	}
	return []int{start}
}

// stronglyConnected is Tarjan's algorithm over nodes 0..n-1.
func stronglyConnected(n int, successors func(int) []int) [][]int {
	index, low := make([]int, n), make([]int, n)
	onStack := make([]bool, n)
	for i := range index {
		index[i] = -1
	}
	var stack []int
	var out [][]int
	next := 0
	var visit func(int)
	visit = func(v int) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range successors(v) {
			if index[w] < 0 {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] != index[v] {
			return
		}
		var scc []int
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		out = append(out, scc)
	}
	for v := range n {
		if index[v] < 0 {
			visit(v)
		}
	}
	return out
}
