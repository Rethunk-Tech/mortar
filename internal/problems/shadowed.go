package problems

import (
	"encoding/json"
	"image"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// shapedFields are the change fields whose whole effect editShapes and dataShapes describe.
var shapedFields = map[string]bool{
	"action": true, "target": true, "fromfile": true, "priority": true, "patchmode": true, "when": true,
	"fromarea": true, "toarea": true, "maptiles": true, "mapproperties": true, "fields": true,
	"targetfield": true, "entries": true, "logname": true, "update": true,
}

// changeDoesMore reports a change with effects its shapes leave out, which no later edit overwrites: any field
// beyond shapedFields (AddWarps, TextOperations, MoveEntries...), appended list entries, tile properties, and a
// source map, whose map properties are merged in too.
func changeDoesMore(raw json.RawMessage, ch cpChange) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return true
	}
	for k := range fields {
		if !shapedFields[strings.ToLower(k)] {
			return true
		}
	}
	if strings.EqualFold(strings.TrimSpace(ch.Action), kindEditMap) && strings.TrimSpace(ch.FromFile) != "" {
		return true
	}
	var entries map[string]json.RawMessage
	if json.Unmarshal(ch.Entries, &entries) == nil {
		if _, ok := entries["#-1"]; ok {
			return true
		}
	}
	for _, tile := range ch.MapTiles {
		var t struct {
			SetProperties json.RawMessage `json:"SetProperties"`
		}
		if json.Unmarshal(tile, &t) != nil || len(t.SetProperties) > 0 {
			return true
		}
	}
	return false
}

// shadowedPacks finds the enabled Content Patcher packs whose every active change is overwritten: each edit's
// pixels, tiles, properties and data keys are all rewritten by unconditional edits of other packs that apply
// after it, and each Load loses to another pack's unconditional Load of higher priority. A pack with any
// change this cannot judge (a tokenized target, a change with no shape, one that does more than its shape)
// is never shadowed.
func shadowedPacks(mods []Installed, at map[string]map[string][]packHit) []Redundant {
	var out []Redundant
	for _, mod := range mods {
		pack := readContentPack(mod)
		if len(pack.patches) == 0 || pack.skips > 0 {
			continue
		}
		by, ok := packShadowedBy(mod, pack, at)
		if !ok {
			continue
		}
		refs := make([]ModRef, 0, len(by))
		for _, key := range slices.Sorted(maps.Keys(by)) {
			refs = append(refs, ModRef{Key: key, Name: by[key]})
		}
		slices.SortStableFunc(refs, func(a, b ModRef) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
		out = append(out, Redundant{Kind: "shadowed", Key: mod.Key, UniqueID: mod.UniqueID, Name: mod.Name, By: refs})
	}
	slices.SortFunc(out, func(a, b Redundant) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return out
}

// packShadowedBy returns the packs, by key, that overwrite all of mod's active changes, or false when any of
// them still shows or the pack has none.
func packShadowedBy(mod Installed, pack cachedPack, at map[string]map[string][]packHit) (map[string]string, bool) {
	if slices.ContainsFunc(pack.patches, func(p cpPatch) bool { return p.kind == "other" && p.tokenName == "" }) {
		return nil, false
	}
	by := map[string]string{}
	changes := 0
	seen := map[string]bool{}
	for _, p := range pack.patches {
		if (p.kind != "load" && p.kind != "edit") || seen[p.kind+"\x00"+p.target] {
			continue
		}
		seen[p.kind+"\x00"+p.target] = true
		hits := at[p.kind][p.target]
		i := slices.IndexFunc(hits, func(h packHit) bool { return sameID(h.id, mod.UniqueID) })
		if i < 0 {
			continue
		}
		self := hits[i]
		if self.key != mod.Key {
			return nil, false
		}
		for _, l := range self.loads {
			changes++
			w, ok := loadWinner(l, self, hits)
			if !ok {
				return nil, false
			}
			by[hits[w].key] = hits[w].name
		}
		for _, e := range self.edits {
			changes++
			if e.extra {
				return nil, false
			}
			used, ok := editCoveredBy(e, self, hits)
			if !ok {
				return nil, false
			}
			for _, w := range used {
				by[hits[w].key] = hits[w].name
			}
		}
	}
	return by, changes > 0
}

// loadWinner is the hit whose Load replaces l: the one load of the target with the highest priority, held by
// another pack, applying unconditionally and ranked above l. Ties leave Content Patcher's choice unclear.
func loadWinner(l cpPatch, self packHit, hits []packHit) (int, bool) {
	winner, top, count := -1, 0, 0
	var best cpPatch
	for i, h := range hits {
		for _, c := range h.loads {
			rank := contentPatcherPriority("load", c.priority)
			switch {
			case count == 0 || rank > top:
				winner, top, count, best = i, rank, 1, c
			case rank == top:
				count++
			}
		}
	}
	if count != 1 || best.when.conditional || hits[winner].key == self.key || sameID(hits[winner].id, self.id) {
		return -1, false
	}
	return winner, top > contentPatcherPriority("load", l.priority)
}

type tileAt struct {
	layer string
	x, y  int
}

type ownedRect struct {
	owner int
	r     image.Rectangle
}

// coverIndex is what the edits applying after one edit write, by owning hit: data keys and map properties,
// single map tiles by layer, and areas by layer ("*" for a Replace map patch, which rewrites every layer).
type coverIndex struct {
	keys  map[string]int
	tiles map[tileAt]int
	rects map[string][]ownedRect
}

// editCoveredBy returns the hits whose edits overwrite all of e, or false when part of it still shows.
func editCoveredBy(e cpPatch, self packHit, hits []packHit) ([]int, bool) {
	idx := coverIndex{keys: map[string]int{}, tiles: map[tileAt]int{}, rects: map[string][]ownedRect{}}
	selfRank := contentPatcherPriority("edit", e.priority)
	for i, h := range hits {
		if h.key == self.key || sameID(h.id, self.id) {
			continue
		}
		later := h.loadAfter[strings.ToLower(self.id)] || h.dependencies[strings.ToLower(self.id)]
		for _, c := range h.edits {
			rank := contentPatcherPriority("edit", c.priority)
			if rank < selfRank || (rank == selfRank && !later) {
				continue
			}
			idx.add(i, c)
		}
	}
	used := map[int]bool{}
	for _, s := range e.shapes {
		if !idx.covers(s, e.image, used) {
			return nil, false
		}
	}
	return slices.Sorted(maps.Keys(used)), len(e.shapes) > 0
}

// add indexes what c rewrites in full. A conditional or tokenized edit may not apply, an overlay only writes
// its opaque pixels, and an area on unknown layers or of unknown size covers nothing for certain.
func (idx coverIndex) add(owner int, c cpPatch) {
	if c.when.conditional || hasToken(c.fromFile) {
		return
	}
	overlay := strings.EqualFold(strings.TrimSpace(c.patchMode), "overlay")
	if c.image && overlay {
		return
	}
	for _, s := range c.shapes {
		switch {
		case s.kind == 'p':
			idx.keys[s.key] = owner
		case s.kind == 't' && s.layer != "":
			idx.tiles[tileAt{s.layer, s.x, s.y}] = owner
		case s.kind == 'r' && s.cells == "" && s.w > 0 && s.h > 0:
			layer := s.layer
			if !c.image && layer == "" {
				if !strings.EqualFold(strings.TrimSpace(c.patchMode), "replace") {
					continue
				}
				layer = "*"
			}
			idx.rects[layer] = append(idx.rects[layer], ownedRect{owner, image.Rect(s.x, s.y, s.x+s.w, s.y+s.h)})
		}
	}
}

func (idx coverIndex) covers(s cpShape, img bool, used map[int]bool) bool {
	switch s.kind {
	case 'p':
		owner, ok := idx.keys[s.key]
		if ok {
			used[owner] = true
		}
		return ok
	case 't':
		return idx.coversRect(image.Rect(s.x, s.y, s.x+1, s.y+1), s.layer, used)
	case 'r':
		if s.cells == "" {
			return idx.coversRect(image.Rect(s.x, s.y, s.x+s.w, s.y+s.h), s.layer, used)
		}
		if !img {
			return false
		}
		// An overlay's cells are 16-pixel squares holding at least one opaque pixel; the whole square must be
		// rewritten, since which of its pixels are opaque is not kept.
		for cell := range internCellSet(s.cells) {
			xs, ys, _ := strings.Cut(cell, ",")
			cx, errX := strconv.Atoi(xs)
			cy, errY := strconv.Atoi(ys)
			if errX != nil || errY != nil || !idx.coversRect(image.Rect(cx*16, cy*16, cx*16+16, cy*16+16), "", used) {
				return false
			}
		}
		return true
	}
	return false
}

// coversRect subtracts the indexed areas on layer (and on every layer) from area, then needs a tile for
// each cell left over.
func (idx coverIndex) coversRect(area image.Rectangle, layer string, used map[int]bool) bool {
	if area.Empty() {
		return true
	}
	left := []image.Rectangle{area}
	for _, l := range []string{layer, "*"} {
		for _, o := range idx.rects[l] {
			if len(left) == 0 {
				return true
			}
			next, hit := subtractRect(left, o.r)
			if hit {
				used[o.owner] = true
			}
			left = next
		}
	}
	for _, r := range left {
		if layer == "" || r.Dx()*r.Dy() > 4096 {
			return false
		}
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				owner, ok := idx.tiles[tileAt{layer, x, y}]
				if !ok {
					return false
				}
				used[owner] = true
			}
		}
	}
	return true
}

// subtractRect removes c from every rectangle in rs, splitting each it cuts into up to four pieces.
func subtractRect(rs []image.Rectangle, c image.Rectangle) ([]image.Rectangle, bool) {
	out := rs[:0:0]
	hit := false
	for _, r := range rs {
		in := r.Intersect(c)
		if in.Empty() {
			out = append(out, r)
			continue
		}
		hit = true
		if r.Min.Y < in.Min.Y {
			out = append(out, image.Rect(r.Min.X, r.Min.Y, r.Max.X, in.Min.Y))
		}
		if in.Max.Y < r.Max.Y {
			out = append(out, image.Rect(r.Min.X, in.Max.Y, r.Max.X, r.Max.Y))
		}
		if r.Min.X < in.Min.X {
			out = append(out, image.Rect(r.Min.X, in.Min.Y, in.Min.X, in.Max.Y))
		}
		if in.Max.X < r.Max.X {
			out = append(out, image.Rect(in.Max.X, in.Min.Y, r.Max.X, in.Max.Y))
		}
	}
	return out, hit
}
