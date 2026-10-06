package problems

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
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
	type edge struct {
		to       int
		required bool
	}
	edges := make([][]edge, len(nodes))
	for i, m := range nodes {
		add := func(id mod.ID, required bool) {
			j, ok := index[id.Fold()]
			if !ok {
				return
			}
			if k := slices.IndexFunc(edges[i], func(e edge) bool { return e.to == j }); k >= 0 {
				edges[i][k].required = edges[i][k].required || required
				return
			}
			edges[i] = append(edges[i], edge{to: j, required: required})
		}
		for _, d := range m.Dependencies {
			add(d.ModID(), d.Required)
		}
		if cp := m.ContentPackForID(); cp != "" {
			add(cp, true)
		}
	}
	successors := func(i int) []int {
		out := make([]int, len(edges[i]))
		for k, e := range edges[i] {
			out[k] = e.to
		}
		return out
	}
	var out []Broken
	for _, scc := range stronglyConnected(len(nodes), successors) {
		start := slices.MinFunc(scc, func(a, b int) int {
			return strings.Compare(strings.ToLower(nodes[a].Name), strings.ToLower(nodes[b].Name))
		})
		if len(scc) == 1 && !slices.Contains(successors(start), start) {
			continue
		}
		path := shortestLoop(start, scc, successors)
		blocksAll := true
		loop := make([]Dependent, len(path))
		for k, i := range path {
			next := path[(k+1)%len(path)]
			e := edges[i][slices.IndexFunc(edges[i], func(e edge) bool { return e.to == next })]
			blocksAll = blocksAll && e.required
			loop[k] = Dependent{Key: nodes[i].Key, ID: nodes[i].ModID(), Name: nodes[i].Name}
		}
		first := nodes[start]
		out = append(out, Broken{
			Key: first.Key, ID: first.ModID(), Name: first.Name, Status: "cycle",
			Cycle: loop, CycleBlocksAll: blocksAll || len(loop) == 1, Summary: cycleSentence(loop, blocksAll),
		})
	}
	slices.SortFunc(out, func(a, b Broken) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return out
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
