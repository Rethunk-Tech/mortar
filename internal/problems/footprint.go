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

	"github.com/Rethunk-AI/mortar/internal/fsx"
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
		to, ok := areaOf(ch.ToArea)
		switch {
		case ok:
			out = append(out, to)
		case len(ch.ToArea) > 0:
			return whole
		default:
			// Without ToArea the source map lands at the top-left at its own size (or FromArea's).
			if from, ok := areaOf(ch.FromArea); ok {
				out = append(out, cpShape{kind: 'r', w: from.w, h: from.h})
			} else if w, h, ok := mapSize(root, ch.FromFile); ok {
				out = append(out, cpShape{kind: 'r', w: w, h: h})
			} else {
				return whole
			}
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

var tmxSize = regexp.MustCompile(`<map\b[^>]*?\bwidth="(\d+)"[^>]*?\bheight="(\d+)"`)

// mapSize reads a .tmx or .tmj map's size in tiles; .tbin and tokenized paths are unknown.
func mapSize(root, rel string) (w, h int, ok bool) {
	rel = strings.TrimSpace(rel)
	ext := strings.ToLower(filepath.Ext(rel))
	if rel == "" || hasToken(rel) || (ext != ".tmx" && ext != ".tmj") {
		return 0, 0, false
	}
	abs, inRoot := inside(root, rel)
	if !inRoot {
		return 0, 0, false
	}
	raw, err := fsx.ReadFile(abs)
	if err != nil {
		return 0, 0, false
	}
	if ext == ".tmj" {
		var m struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}
		if json.Unmarshal(raw, &m) != nil || m.Width <= 0 || m.Height <= 0 {
			return 0, 0, false
		}
		return m.Width, m.Height, true
	}
	head := raw[:min(len(raw), 4096)]
	m := tmxSize.FindSubmatch(head)
	if m == nil {
		return 0, 0, false
	}
	w, _ = strconv.Atoi(string(m[1]))
	h, _ = strconv.Atoi(string(m[2]))
	return w, h, w > 0 && h > 0
}

// placeTokens are conditions that hold one value at a time for the player: two edits that need disjoint
// values of the same one never apply together (an edit for the East Scarp village and one for another map).
var placeTokens = map[string]bool{"locationname": true, "locationcontext": true, "season": true, "weather": true, "dayofweek": true}

// placesOf reads the literal values a When block requires of placeTokens ("LocationName": "A, B" or
// "Season |contains=Spring": true).
func placesOf(raw map[string]json.RawMessage) map[string][]string {
	var out map[string][]string
	for k, v := range raw {
		if hasToken(k) {
			continue
		}
		name, arg, _ := strings.Cut(k, "|")
		name = tokenName(name)
		if !placeTokens[name] {
			continue
		}
		var values []string
		if arg = strings.TrimSpace(arg); arg == "" {
			if !condValues(v, &values) {
				continue
			}
		} else {
			param, list, ok := strings.Cut(arg, "=")
			var flags []string
			if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) || !condValues(v, &flags) || len(flags) != 1 || !strings.EqualFold(flags[0], "true") {
				continue
			}
			values = splitTargets(list)
		}
		if out == nil {
			out = map[string][]string{}
		}
		for _, value := range values {
			out[name] = append(out[name], strings.ToLower(value))
		}
	}
	return out
}

// tokenName is a condition key's token, lower-cased, with an empty input ("HasMod:") dropped as Content
// Patcher does; a key with a real input ("Weather: Island") keeps it so it is not mistaken for the bare token.
func tokenName(name string) string {
	name = strings.TrimSpace(name)
	if base, input, ok := strings.Cut(name, ":"); ok && strings.TrimSpace(input) == "" {
		name = base
	}
	return strings.ToLower(strings.TrimSpace(name))
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
	if a.spouse != "" && b.spouse != "" && a.spouse != b.spouse {
		return true
	}
	for token, values := range a.places {
		other, ok := b.places[token]
		if ok && !slices.ContainsFunc(values, func(v string) bool { return slices.Contains(other, v) }) {
			return true
		}
	}
	return false
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
