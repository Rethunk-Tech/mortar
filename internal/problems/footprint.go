package problems

import (
	"encoding/json"
	"image"
	_ "image/png" // DecodeConfig reads a FromFile's size from its PNG header.
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// cpShape is part of a target one edit writes: an area (image pixels, or map tiles on every layer), one map
// tile on one layer, one map property, or the whole asset when the area cannot be known without running
// Content Patcher (tokens, a map patched from a file without ToArea).
type cpShape struct {
	kind       byte // 'r' area, 't' tile, 'p' property, 'w' whole
	x, y, w, h int
	layer, key string
}

func (s cpShape) area() (x, y, w, h int) {
	if s.kind == 't' {
		return s.x, s.y, 1, 1
	}
	return s.x, s.y, s.w, s.h
}

// overlaps reports whether two edits can write the same pixels, tiles or properties. A whole-asset edit is
// taken to touch every tile but no map property, since a map patch replaces tiles and keeps the properties
// it does not name.
func (s cpShape) overlaps(o cpShape) bool {
	if s.kind == 'p' || o.kind == 'p' {
		return s.kind == o.kind && s.key == o.key
	}
	if s.kind == 'w' || o.kind == 'w' {
		return true
	}
	if s.kind == 't' && o.kind == 't' && s.layer != o.layer {
		return false
	}
	ax, ay, aw, ah := s.area()
	bx, by, bw, bh := o.area()
	return ax < bx+bw && bx < ax+aw && ay < by+bh && by < ay+ah
}

func shapesOverlap(a, b []cpShape) bool {
	for _, x := range a {
		if slices.ContainsFunc(b, x.overlaps) {
			return true
		}
	}
	return false
}

var whole = []cpShape{{kind: 'w'}}

// editShapes is what one EditImage or EditMap change writes. An edit that only adds warps, rewrites text
// properties or sets tile properties has no shape: Content Patcher merges those, so they never overwrite
// another pack's work.
func editShapes(root string, ch cpChange, image bool) []cpShape {
	if image {
		if to, ok := areaOf(ch.ToArea); ok {
			return []cpShape{to}
		} else if len(ch.ToArea) > 0 {
			return whole
		}
		// Without ToArea the source lands at the top-left, sized like FromArea or the whole file.
		if from, ok := areaOf(ch.FromArea); ok {
			return []cpShape{{kind: 'r', w: from.w, h: from.h}}
		}
		if w, h, ok := pngSize(root, ch.FromFile); ok {
			return []cpShape{{kind: 'r', w: w, h: h}}
		}
		return whole
	}
	var out []cpShape
	if strings.TrimSpace(ch.FromFile) != "" {
		if to, ok := areaOf(ch.ToArea); ok {
			out = append(out, to)
		} else {
			return whole
		}
	}
	for _, raw := range ch.MapTiles {
		var tile struct {
			Position      json.RawMessage `json:"Position"`
			Layer         string          `json:"Layer"`
			SetIndex      json.RawMessage `json:"SetIndex"`
			SetTilesheet  json.RawMessage `json:"SetTilesheet"`
			Remove        json.RawMessage `json:"Remove"`
			SetProperties json.RawMessage `json:"SetProperties"`
		}
		if json.Unmarshal(raw, &tile) != nil {
			return whole
		}
		if len(tile.SetIndex) == 0 && len(tile.SetTilesheet) == 0 && len(tile.Remove) == 0 {
			continue
		}
		pos, ok := areaOf(tile.Position)
		if !ok || hasToken(tile.Layer) {
			return whole
		}
		out = append(out, cpShape{kind: 't', x: pos.x, y: pos.y, layer: strings.ToLower(strings.TrimSpace(tile.Layer))})
	}
	for key := range ch.MapProperties {
		if hasToken(key) {
			return whole
		}
		out = append(out, cpShape{kind: 'p', key: strings.ToLower(key)})
	}
	return out
}

// areaOf reads {X, Y, Width, Height} (or a Position's {X, Y}) given as numbers or numeric strings; false when
// it is missing or holds a token.
func areaOf(raw json.RawMessage) (cpShape, bool) {
	if len(raw) == 0 {
		return cpShape{}, false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return cpShape{}, false
	}
	s := cpShape{kind: 'r', w: 1, h: 1}
	for k, v := range fields {
		var n int
		var str string
		switch {
		case json.Unmarshal(v, &n) == nil:
		case json.Unmarshal(v, &str) == nil:
			parsed, err := strconv.Atoi(strings.TrimSpace(str))
			if err != nil {
				return cpShape{}, false
			}
			n = parsed
		default:
			return cpShape{}, false
		}
		switch strings.ToLower(k) {
		case "x":
			s.x = n
		case "y":
			s.y = n
		case "width":
			s.w = n
		case "height":
			s.h = n
		}
	}
	return s, true
}

func pngSize(root, rel string) (w, h int, ok bool) {
	rel = strings.TrimSpace(rel)
	if rel == "" || hasToken(rel) || !strings.EqualFold(filepath.Ext(rel), ".png") {
		return 0, 0, false
	}
	f, err := os.OpenInRoot(root, filepath.FromSlash(rel))
	if err != nil {
		return 0, 0, false
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

var spouseQuery = regexp.MustCompile(`(?i)^query:\s*'\{\{\s*spouse\s*\}\}'\s*=\s*'([^']+)'$`)

// spouseOf is the NPC a When block requires the player to be married to ("Relationship:Abigail": "Married" or
// "Query: '{{Spouse}}' = 'Abigail'": true), or "". A player has one spouse, so two patches that need
// different spouses never apply together.
func spouseOf(raw map[string]json.RawMessage) string {
	for k, v := range raw {
		key := strings.TrimSpace(k)
		if m := spouseQuery.FindStringSubmatch(key); m != nil {
			var on bool
			if json.Unmarshal(v, &on) == nil && on {
				return strings.ToLower(strings.TrimSpace(m[1]))
			}
			continue
		}
		name, npc, ok := strings.Cut(key, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "relationship") || hasToken(npc) {
			continue
		}
		var state string
		if json.Unmarshal(v, &state) == nil && strings.EqualFold(strings.TrimSpace(state), "married") {
			return strings.ToLower(strings.TrimSpace(npc))
		}
	}
	return ""
}

// exclusive reports whether two edits can never be active together.
func exclusive(a, b cpPatch) bool {
	return a.spouse != "" && b.spouse != "" && a.spouse != b.spouse
}

// editsClash reports whether any active edit of one pack can overwrite one of the other's.
func editsClash(a, b []cpPatch) bool {
	for _, x := range a {
		for _, y := range b {
			if !exclusive(x, y) && shapesOverlap(x.shapes, y.shapes) {
				return true
			}
		}
	}
	return false
}
