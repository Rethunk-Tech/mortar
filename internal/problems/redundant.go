package problems

import (
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

var markdownLinkText = regexp.MustCompile(`\[([^\]]+)\]\(`)

// superseded finds enabled mods whose SMAPI compatibility summary sends the player to another mod that is enabled
// too: by Nexus page, mod id or the linked name. Their Broken and Compat rows are dropped, since the advice is
// already taken.
func superseded(r Result, domain string, mods []Installed) Result {
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
	add := func(key string, uniqueID mod.ID, name, summary string) {
		if gone[key] || listed[key] {
			return
		}
		by := namedIn(domain, summary, enabled, key)
		if len(by) == 0 {
			return
		}
		gone[key] = true
		r.Redundant = append(r.Redundant, Redundant{Kind: "superseded", Key: key, ID: uniqueID, Name: name, By: by, Detail: summary})
	}
	for _, b := range r.Broken {
		add(b.Key, b.ID, b.Name, b.Summary)
	}
	for _, c := range r.Compat {
		if c.Status != "ok" && c.Status != "optional" {
			add(c.Key, c.ID, c.Name, c.Summary)
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
func namedIn(domain, summary string, enabled []Installed, self string) []ModRef {
	if strings.TrimSpace(summary) == "" {
		return nil
	}
	pages := map[int]bool{}
	for _, id := range modPageIDs(nexusModURL, summary, domain) {
		pages[id] = true
	}
	ids := map[string]bool{}
	for _, id := range uniqueID.FindAllString(summary, -1) {
		ids[mod.SMAPI(id).Fold()] = true
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
		if pages[nexusIDOf(m)] || ids[m.ModID().Fold()] || names[strings.ToLower(strings.TrimSpace(m.Name))] {
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
