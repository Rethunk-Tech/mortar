package problems

import (
	"regexp"
	"strconv"
	"strings"
)

// Redundant is an enabled mod that adds nothing beside the others: "superseded" when its replacement is enabled too,
// "shadowed" when later packs overwrite every edit it makes, "sameJob" when another enabled C# mod changes the same game
// members (Covered when a larger mod changes everything this one does).
type Redundant struct {
	Kind     string   `json:"kind"`
	Key      string   `json:"key"`
	UniqueID string   `json:"uniqueId"`
	Name     string   `json:"name"`
	By       []ModRef `json:"by"`
	Detail   string   `json:"detail,omitempty"`
	Covered  bool     `json:"covered,omitempty"`
}

// ModRef names one enabled mod.
type ModRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

var markdownLinkText = regexp.MustCompile(`\[([^\]]+)\]\(`)

// superseded finds enabled mods whose SMAPI compatibility summary sends the player to another mod that is enabled
// too: by Nexus page, UniqueID or the linked name. Their Broken and Compat rows are dropped, since the advice is
// already taken.
func superseded(r Result, mods []Installed) Result {
	enabled := make([]Installed, 0, len(mods))
	for _, m := range mods {
		if m.Enabled {
			enabled = append(enabled, m)
		}
	}
	gone := map[string]bool{}
	listed := map[string]bool{}
	for _, x := range r.Redundant {
		listed[x.Key] = true
	}
	add := func(key, uniqueID, name, summary string) {
		if gone[key] || listed[key] {
			return
		}
		by := namedIn(summary, enabled, key)
		if len(by) == 0 {
			return
		}
		gone[key] = true
		r.Redundant = append(r.Redundant, Redundant{Kind: "superseded", Key: key, UniqueID: uniqueID, Name: name, By: by, Detail: summary})
	}
	for _, b := range r.Broken {
		add(b.Key, b.UniqueID, b.Name, b.Summary)
	}
	for _, c := range r.Compat {
		if c.Status != "ok" && c.Status != "optional" {
			add(c.Key, c.UniqueID, c.Name, c.Summary)
		}
	}
	if len(gone) == 0 {
		return r
	}
	broken := r.Broken[:0:0]
	for _, b := range r.Broken {
		if !gone[b.Key] {
			broken = append(broken, b)
		}
	}
	compat := r.Compat[:0:0]
	for _, c := range r.Compat {
		if !gone[c.Key] {
			compat = append(compat, c)
		}
	}
	r.Broken, r.Compat = broken, compat
	return r
}

// namedIn lists the enabled mods, other than self, that summary names.
func namedIn(summary string, enabled []Installed, self string) []ModRef {
	if strings.TrimSpace(summary) == "" {
		return nil
	}
	pages := map[int]bool{}
	for _, parts := range nexusModURL.FindAllStringSubmatch(summary, -1) {
		if id, err := strconv.Atoi(parts[1]); err == nil {
			pages[id] = true
		}
	}
	ids := map[string]bool{}
	for _, id := range uniqueID.FindAllString(summary, -1) {
		ids[strings.ToLower(id)] = true
	}
	names := map[string]bool{}
	for _, parts := range markdownLinkText.FindAllStringSubmatch(summary, -1) {
		names[strings.ToLower(strings.TrimSpace(parts[1]))] = true
	}
	var out []ModRef
	for _, m := range enabled {
		if m.Key == self {
			continue
		}
		if pages[nexusIDOf(m)] || ids[strings.ToLower(m.UniqueID)] || names[strings.ToLower(strings.TrimSpace(m.Name))] {
			out = append(out, ModRef{Key: m.Key, Name: m.Name})
		}
	}
	return out
}

// redundantCount counts each group of mods doing the same job once, as the Problems tab shows it, and every other
// Redundant row on its own.
func redundantCount(rows []Redundant) int {
	parent := map[string]string{}
	var find func(string) string
	find = func(k string) string {
		p, ok := parent[k]
		if !ok || p == k {
			parent[k] = k
			return k
		}
		root := find(p)
		parent[k] = root
		return root
	}
	n := 0
	for _, row := range rows {
		if row.Kind != "sameJob" || row.Covered {
			n++
			continue
		}
		for _, by := range row.By {
			parent[find(by.Key)] = find(row.Key)
		}
	}
	groups := map[string]bool{}
	for k := range parent {
		groups[find(k)] = true
	}
	return n + len(groups)
}
