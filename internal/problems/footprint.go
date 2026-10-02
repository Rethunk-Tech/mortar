package problems

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"image"
	_ "image/png" // DecodeConfig reads a FromFile's size from its PNG header.
	"io"
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
	// value is a property's literal value, "" when tokenized; tiny marks a whole-asset shape that is really one
	// or two tiles at a position only Content Patcher can work out.
	value string
	tiny  bool
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
		// Two packs setting a property to the same value agree; only different (or unknown) values clash.
		return s.kind == o.kind && s.key == o.key && (s.value == "" || o.value == "" || !strings.EqualFold(s.value, o.value))
	}
	if s.kind == 'w' || o.kind == 'w' {
		return true
	}
	if s.layer != "" && o.layer != "" && s.layer != o.layer {
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
			return unplaced(ch.ToArea)
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
		ext := strings.ToLower(filepath.Ext(strings.TrimSpace(ch.FromFile)))
		if ext == ".tmx" || ext == ".tmj" {
			if decoded, ok := decodeMap(root, ch.FromFile); ok {
				return mapFileShapes(decoded, ch)
			}
		}
		to, ok := areaOf(ch.ToArea)
		switch {
		case ok:
			out = append(out, to)
		case len(ch.ToArea) > 0:
			return unplaced(ch.ToArea)
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
	for key, raw := range ch.MapProperties {
		if hasToken(key) {
			return whole
		}
		value, _ := scalarValue(raw)
		if hasToken(value) {
			value = ""
		}
		out = append(out, cpShape{kind: 'p', key: strings.ToLower(key), value: value})
	}
	return out
}

// unplaced is the shape of an area whose position is tokenized: the whole asset, marked tiny when its literal
// size is at most two tiles (a trapdoor or a sign placed by a computed position).
func unplaced(raw json.RawMessage) []cpShape {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return whole
	}
	w, h := 0, 0
	for k, v := range fields {
		var n int
		if json.Unmarshal(v, &n) != nil {
			continue
		}
		switch strings.ToLower(k) {
		case "width":
			w = n
		case "height":
			h = n
		}
	}
	if w > 0 && h > 0 && w <= 2 && h <= 2 {
		return []cpShape{{kind: 'w', tiny: true}}
	}
	return whole
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

type decodedMap struct {
	width, height int
	layers        []decodedMapLayer
}

type decodedMapLayer struct {
	name  string
	tiles []mapTile
}

type mapTile struct {
	x, y int
}

type tmxMapDocument struct {
	Width  int        `xml:"width,attr"`
	Height int        `xml:"height,attr"`
	Layers []tmxLayer `xml:"layer"`
}

type tmxLayer struct {
	Name   string  `xml:"name,attr"`
	Width  int     `xml:"width,attr"`
	Height int     `xml:"height,attr"`
	Data   tmxData `xml:"data"`
}

type tmxData struct {
	Encoding    string     `xml:"encoding,attr"`
	Compression string     `xml:"compression,attr"`
	Text        string     `xml:",chardata"`
	Tiles       []tmxTile  `xml:"tile"`
	Chunks      []tmxChunk `xml:"chunk"`
}

type tmxTile struct {
	GID uint32 `xml:"gid,attr"`
}

type tmxChunk struct {
	X      int       `xml:"x,attr"`
	Y      int       `xml:"y,attr"`
	Width  int       `xml:"width,attr"`
	Height int       `xml:"height,attr"`
	Text   string    `xml:",chardata"`
	Tiles  []tmxTile `xml:"tile"`
}

type tmjMapDocument struct {
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Layers []tmjLayer `json:"layers"`
}

type tmjLayer struct {
	Type   string     `json:"type"`
	Name   string     `json:"name"`
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Data   []int      `json:"data"`
	Chunks []tmjChunk `json:"chunks"`
}

type tmjChunk struct {
	X      int   `json:"x"`
	Y      int   `json:"y"`
	Width  int   `json:"width"`
	Height int   `json:"height"`
	Data   []int `json:"data"`
}

func decodeMap(root, rel string) (decodedMap, bool) {
	abs, ok := inside(root, rel)
	if !ok {
		return decodedMap{}, false
	}
	raw, err := fsx.ReadFile(abs)
	if err != nil {
		return decodedMap{}, false
	}
	if strings.EqualFold(filepath.Ext(rel), ".tmj") {
		var doc tmjMapDocument
		if json.Unmarshal(raw, &doc) != nil {
			return decodedMap{}, false
		}
		out := decodedMap{width: doc.Width, height: doc.Height}
		for _, layer := range doc.Layers {
			if layer.Type != "" && !strings.EqualFold(layer.Type, "tilelayer") {
				continue
			}
			decoded := decodedMapLayer{name: strings.ToLower(strings.TrimSpace(layer.Name))}
			if len(layer.Chunks) > 0 {
				for _, chunk := range layer.Chunks {
					appendMapTiles(&decoded.tiles, chunk.X, chunk.Y, chunk.Width, chunk.Height, chunk.Data)
				}
			} else {
				width := layer.Width
				if width <= 0 {
					width = doc.Width
				}
				appendMapTiles(&decoded.tiles, 0, 0, width, layer.Height, layer.Data)
			}
			out.layers = append(out.layers, decoded)
		}
		mapBounds(&out)
		return out, true
	}

	var doc tmxMapDocument
	if xml.Unmarshal(raw, &doc) != nil {
		return decodedMap{}, false
	}
	out := decodedMap{width: doc.Width, height: doc.Height}
	for _, layer := range doc.Layers {
		decoded, ok := decodeTMXLayer(layer)
		if !ok {
			return decodedMap{}, false
		}
		decoded.name = strings.ToLower(strings.TrimSpace(layer.Name))
		out.layers = append(out.layers, decoded)
	}
	mapBounds(&out)
	return out, true
}

func decodeTMXLayer(layer tmxLayer) (decodedMapLayer, bool) {
	out := decodedMapLayer{}
	if len(layer.Data.Chunks) > 0 {
		for _, chunk := range layer.Data.Chunks {
			values, ok := decodeTMXValues(chunk.Text, layer.Data.Encoding, layer.Data.Compression)
			if len(chunk.Tiles) > 0 {
				values = make([]int, len(chunk.Tiles))
				for i, tile := range chunk.Tiles {
					values[i] = int(tile.GID)
				}
				ok = true
			}
			if !ok {
				return decodedMapLayer{}, false
			}
			appendMapTiles(&out.tiles, chunk.X, chunk.Y, chunk.Width, chunk.Height, values)
		}
		return out, true
	}
	values, ok := decodeTMXValues(layer.Data.Text, layer.Data.Encoding, layer.Data.Compression)
	if len(layer.Data.Tiles) > 0 {
		values = make([]int, len(layer.Data.Tiles))
		for i, tile := range layer.Data.Tiles {
			values[i] = int(tile.GID)
		}
		ok = true
	}
	if !ok {
		return decodedMapLayer{}, false
	}
	appendMapTiles(&out.tiles, 0, 0, layer.Width, layer.Height, values)
	return out, true
}

func decodeTMXValues(text, encoding, compression string) ([]int, bool) {
	text = strings.TrimSpace(text)
	if strings.EqualFold(encoding, "csv") {
		if text == "" {
			return nil, true
		}
		var out []int
		for _, value := range strings.Split(text, ",") {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, false
			}
			out = append(out, n)
		}
		return out, true
	}
	if !strings.EqualFold(encoding, "base64") {
		return nil, text == ""
	}
	encoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(text), ""))
	if err != nil {
		return nil, false
	}
	var data []byte
	switch strings.ToLower(strings.TrimSpace(compression)) {
	case "":
		data = encoded
	case "zlib":
		reader, err := zlib.NewReader(bytesReader(encoded))
		if err != nil {
			return nil, false
		}
		data, err = io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			return nil, false
		}
	case "gzip":
		reader, err := gzip.NewReader(bytesReader(encoded))
		if err != nil {
			return nil, false
		}
		data, err = io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			return nil, false
		}
	default:
		return nil, false
	}
	if len(data)%4 != 0 {
		return nil, false
	}
	out := make([]int, len(data)/4)
	for i := range out {
		out[i] = int(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return out, true
}

func bytesReader(data []byte) io.Reader {
	return &sliceReader{data: data}
}

type sliceReader struct {
	data []byte
	pos  int
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func appendMapTiles(out *[]mapTile, x, y, width, height int, values []int) {
	if width <= 0 {
		return
	}
	for i, value := range values {
		if value == 0 {
			continue
		}
		tx := x + i%width
		ty := y + i/width
		if height > 0 && i/width >= height {
			break
		}
		*out = append(*out, mapTile{x: tx, y: ty})
	}
}

func mapBounds(m *decodedMap) {
	for _, layer := range m.layers {
		for _, tile := range layer.tiles {
			m.width = max(m.width, tile.x+1)
			m.height = max(m.height, tile.y+1)
		}
	}
}

func mapFileShapes(m decodedMap, ch cpChange) []cpShape {
	fallback := cpShape{kind: 'r', w: m.width, h: m.height}
	if fallback.w <= 0 || fallback.h <= 0 {
		return nil
	}
	from, ok := mapArea(ch.FromArea, fallback)
	if !ok {
		return whole
	}
	toFallback := fallback
	toFallback.x, toFallback.y = 0, 0
	to, ok := mapArea(ch.ToArea, toFallback)
	if !ok {
		return unplaced(ch.ToArea)
	}
	mode := strings.ToLower(strings.TrimSpace(ch.PatchMode))
	if mode == "" {
		mode = "replacebylayer"
	}
	switch mode {
	case "overlay":
		var out []cpShape
		for _, layer := range m.layers {
			for _, tile := range layer.tiles {
				if tile.x < from.x || tile.x >= from.x+from.w || tile.y < from.y || tile.y >= from.y+from.h {
					continue
				}
				out = append(out, cpShape{
					kind: 't', x: to.x + tile.x - from.x, y: to.y + tile.y - from.y,
					layer: layer.name,
				})
			}
		}
		return out
	case "replace":
		to.layer = ""
		return []cpShape{to}
	case "replacebylayer":
		var out []cpShape
		for _, layer := range m.layers {
			area := to
			area.layer = layer.name
			out = append(out, area)
		}
		return out
	default:
		var out []cpShape
		for _, layer := range m.layers {
			area := to
			area.layer = layer.name
			out = append(out, area)
		}
		return out
	}
}

func mapArea(raw json.RawMessage, fallback cpShape) (cpShape, bool) {
	if len(raw) == 0 {
		return fallback, true
	}
	area, ok := areaOf(raw)
	if !ok {
		return cpShape{}, false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return cpShape{}, false
	}
	if !hasAreaField(fields, "x") {
		area.x = fallback.x
	}
	if !hasAreaField(fields, "y") {
		area.y = fallback.y
	}
	if !hasAreaField(fields, "width") {
		area.w = fallback.w
	}
	if !hasAreaField(fields, "height") {
		area.h = fallback.h
	}
	return area, area.w > 0 && area.h > 0
}

func hasAreaField(fields map[string]json.RawMessage, want string) bool {
	for key := range fields {
		if strings.EqualFold(strings.TrimSpace(key), want) {
			return true
		}
	}
	return false
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
var placeTokens = map[string]bool{"locationname": true, "locationcontext": true, "season": true, "weather": true, "dayofweek": true, "farmtype": true}

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
		if strings.EqualFold(key, "spouse") {
			var npc string
			if json.Unmarshal(v, &npc) == nil && !hasToken(npc) && strings.TrimSpace(npc) != "" {
				return strings.ToLower(strings.TrimSpace(npc))
			}
			continue
		}
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

// editsClash reports whether any active edit of one pack can overwrite one of the other's, and whether every
// such overlap is harmless (see harmless).
func editsClash(a, b []cpPatch) (clash, minor bool) {
	minor = true
	for _, x := range a {
		for _, y := range b {
			if x.image && y.image && strings.EqualFold(strings.TrimSpace(x.patchMode), "overlay") && strings.EqualFold(strings.TrimSpace(y.patchMode), "overlay") {
				continue
			}
			if !exclusive(x, y) && shapesOverlap(x.shapes, y.shapes) {
				clash = true
				minor = minor && harmless(x, y)
			}
		}
	}
	return clash, clash && minor
}

// harmless reports an overlap that cannot hurt play: two image edits only change how something looks; an edit
// that applies in one location or weather only matters there; and a one-tile edit at a computed spot is too
// small to place, so it is shown without counting as a problem.
func harmless(x, y cpPatch) bool {
	return (x.image && y.image) || situational(x) || situational(y) || tinyOnly(x) || tinyOnly(y)
}

func situational(p cpPatch) bool {
	_, location := p.places["locationname"]
	_, context := p.places["locationcontext"]
	_, weather := p.places["weather"]
	return location || context || weather
}

func tinyOnly(p cpPatch) bool {
	return len(p.shapes) > 0 && !slices.ContainsFunc(p.shapes, func(s cpShape) bool { return !s.tiny })
}

// markClashes records which edits of a and b overlap each other.
func markClashes(a, b *packHit) {
	for i, x := range a.edits {
		for j, y := range b.edits {
			if exclusive(x, y) || !shapesOverlap(x.shapes, y.shapes) {
				continue
			}
			if a.clashes == nil {
				a.clashes = map[int]bool{}
			}
			if b.clashes == nil {
				b.clashes = map[int]bool{}
			}
			a.clashes[i], b.clashes[j] = true, true
		}
	}
}

// switchOff finds an on/off field that every clashing edit of the pack needs; its other value removes the pack
// from the conflict, such as Better Things' DesertMinecart for an expansion that redraws the desert.
func switchOff(h packHit, peerSets ...[]packHit) (ConflictFix, bool) {
	if len(h.clashes) == 0 {
		return ConflictFix{}, false
	}
	var first cpPatch
	for i := range h.clashes {
		first = h.edits[i]
		break
	}
	for _, c := range first.when.config {
		field, ok := h.schema[strings.ToLower(c.field)]
		if !ok || !field.toggle() {
			continue
		}
		needed := map[string]bool{}
		all := true
		for i := range h.clashes {
			j := slices.IndexFunc(h.edits[i].when.config, func(o cpConfig) bool { return strings.EqualFold(o.field, field.key) })
			if j < 0 {
				all = false
				break
			}
			for _, v := range h.edits[i].when.config[j].values {
				needed[strings.ToLower(v)] = true
			}
		}
		if !all {
			continue
		}
		values := field.allowValues
		if len(values) == 0 {
			values = []string{"true", "false"}
		}
		var off []string
		for _, v := range values {
			if !needed[strings.ToLower(strings.TrimSpace(v))] {
				off = append(off, strings.TrimSpace(v))
			}
		}
		if len(off) != 1 {
			continue
		}
		if len(peerSets) > 0 && settingStillClashes(h, peerSets[0], field, off[0]) {
			continue
		}
		current, set := h.config[strings.ToLower(field.key)]
		if !set {
			current = field.defaultValue
		}
		return ConflictFix{Key: h.key, UniqueID: h.id, Name: h.name, Field: field.key, Current: current, Value: off[0]}, true
	}
	return ConflictFix{}, false
}

func settingStillClashes(h packHit, peers []packHit, field cpSchema, value string) bool {
	config := map[string]string{}
	for key, current := range h.config {
		config[key] = current
	}
	config[strings.ToLower(field.key)] = value
	var active []cpPatch
	for _, edit := range h.eligible {
		if configHolds(edit.when.config, h.schema, config) {
			active = append(active, edit)
		}
	}
	if len(h.eligible) == 0 {
		active = h.edits
	}
	for _, peer := range peers {
		if sameID(peer.id, h.id) {
			continue
		}
		if clash, _ := editsClash(active, peer.edits); clash {
			return true
		}
	}
	return false
}
