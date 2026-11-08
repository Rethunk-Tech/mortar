package cp

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/go-lenientjson"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/go-tilemask"
)

// Shape is part of a target one edit writes: an area (image pixels, or map tiles on every layer), one map
// tile on one layer, one map property, or the whole asset when the area cannot be known without running
// Content Patcher (tokens, a map patched from a file without ToArea).
type Shape struct {
	Kind       byte // 'r' area, 't' tile, 'p' property, 'w' whole
	X, Y, W, H int
	Cells      string
	Layer, Key string
	// value is a property's literal value, "" when tokenized; tiny marks a whole-asset shape that is really one
	// or two tiles at a position only Content Patcher can work out.
	Value string
	Tiny  bool
}

func (s Shape) area() (x, y, w, h int) {
	if s.Kind == 't' {
		return s.X, s.Y, 1, 1
	}
	return s.X, s.Y, s.W, s.H
}

// overlaps reports whether two edits can write the same pixels, tiles or properties. A whole-asset edit is
// taken to touch every tile but no map property, since a map patch replaces tiles and keeps the properties
// it does not name.
func (s Shape) overlaps(o Shape) bool {
	if s.Kind == 'p' || o.Kind == 'p' {
		// Two packs setting a property to the same value agree; only different (or unknown) values clash.
		return s.Kind == o.Kind && s.Key == o.Key && (s.Value == "" || o.Value == "" || !strings.EqualFold(s.Value, o.Value))
	}
	if s.Kind == 'w' || o.Kind == 'w' {
		return true
	}
	if s.Layer != "" && o.Layer != "" && s.Layer != o.Layer {
		return false
	}
	if s.Cells != "" || o.Cells != "" {
		if s.Cells != "" && o.Cells != "" {
			return cellSetsOverlap(s.Cells, o.Cells)
		}
		cells, area := s.Cells, o
		if cells == "" {
			cells, area = o.Cells, s
		}
		return cellsHitArea(cells, area)
	}
	ax, ay, aw, ah := s.area()
	bx, by, bw, bh := o.area()
	return ax < bx+bw && bx < ax+aw && ay < by+bh && by < ay+ah
}

// cellsHitArea reports whether any encoded "x,y" 16-pixel cell intersects the shape's rectangle.
func cellsHitArea(cells string, area Shape) bool {
	ax, ay, aw, ah := area.area()
	for encoded := range strings.SplitSeq(cells, ";") {
		if encoded == "" {
			continue
		}
		parts := strings.SplitN(encoded, ",", 2)
		if len(parts) != 2 {
			continue
		}
		cxCell, errX := strconv.Atoi(parts[0])
		cyCell, errY := strconv.Atoi(parts[1])
		if errX != nil || errY != nil {
			continue
		}
		cx, cy := cxCell*16, cyCell*16
		if cx < ax+aw && ax < cx+16 && cy < ay+ah && ay < cy+16 {
			return true
		}
	}
	return false
}

func cellSetsOverlap(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	other := internCellSet(b)
	for cell := range internCellSet(a) {
		if other[cell] {
			return true
		}
	}
	return false
}

func internCellSet(s string) map[string]bool {
	if cached, ok := cellSetCache.Load(s); ok {
		if set, ok := cached.(map[string]bool); ok {
			return set
		}
	}
	set := map[string]bool{}
	for cell := range strings.SplitSeq(s, ";") {
		if cell != "" {
			set[cell] = true
		}
	}
	actual, _ := cellSetCache.LoadOrStore(s, set)
	if kept, ok := actual.(map[string]bool); ok {
		return kept
	}
	return set
}

// shapeSet is one patch's shapes with its properties grouped by key: a property only overlaps a property with the
// same key, so a large data edit is matched by key instead of against every key of the other patch.
type shapeSet struct {
	Props map[string][]Shape
	Rest  []Shape
}

func indexShapes(shapes []Shape) *shapeSet {
	set := &shapeSet{Props: map[string][]Shape{}}
	for _, s := range shapes {
		if s.Kind == 'p' {
			set.Props[s.Key] = append(set.Props[s.Key], s)
		} else {
			set.Rest = append(set.Rest, s)
		}
	}
	return set
}

// shapeIndexes holds each patch's shapeSet for the running Check, keyed by the patch's shapes backing array, so a
// target edited by many packs indexes each patch once.
var shapeIndexes = struct {
	sync.Mutex
	sets map[*Shape]*shapeSet
}{sets: map[*Shape]*shapeSet{}}

func shapeIndex(shapes []Shape) *shapeSet {
	if len(shapes) == 0 {
		return indexShapes(shapes)
	}
	shapeIndexes.Lock()
	defer shapeIndexes.Unlock()
	set, ok := shapeIndexes.sets[&shapes[0]]
	if !ok {
		set = indexShapes(shapes)
		shapeIndexes.sets[&shapes[0]] = set
	}
	return set
}

// shapeSummary is what a pack's edits touch, so a pair that touches nothing in common skips the edit-by-edit check.
type shapeSummary struct {
	Props map[string]struct{}
	Rest  bool
}

func summarize(index []*shapeSet) shapeSummary {
	s := shapeSummary{Props: map[string]struct{}{}}
	for _, set := range index {
		for k := range set.Props {
			s.Props[k] = struct{}{}
		}
		s.Rest = s.Rest || len(set.Rest) > 0
	}
	return s
}

// mayOverlap is false only when no shape of one pack can overlap a shape of the other: property shapes overlap only
// on the same key, and every other shape only another non-property shape.
func (s shapeSummary) mayOverlap(o shapeSummary) bool {
	if s.Rest && o.Rest {
		return true
	}
	small, large := s.Props, o.Props
	if len(small) > len(large) {
		small, large = large, small
	}
	for k := range small {
		if _, ok := large[k]; ok {
			return true
		}
	}
	return false
}

func (set *shapeSet) overlaps(shapes []Shape) bool {
	for _, x := range shapes {
		others := set.Rest
		if x.Kind == 'p' {
			others = set.Props[x.Key]
		}
		if slices.ContainsFunc(others, x.overlaps) {
			return true
		}
	}
	return false
}

func shapesOverlap(a, b []Shape) bool {
	for _, x := range a {
		if slices.ContainsFunc(b, x.overlaps) {
			return true
		}
	}
	return false
}

var whole = []Shape{{Kind: 'w'}}

// packScopedKey keeps a data key that holds a token, such as {{ModId}} or a pack's own config token,
// from matching another pack's identical text: each pack fills its tokens with its own values.
func packScopedKey(root, key string) string {
	if hasToken(key) {
		return "@" + root + "|" + key
	}
	return key
}

func dataShapes(root string, ch contentChange, values map[string]string) []Shape {
	var out []Shape
	// With TargetField, Entries and Fields address keys inside that field of one entry, not the asset's
	// top-level entries, so the path is part of every key.
	base := ""
	if len(ch.TargetField) > 0 {
		parts := make([]string, len(ch.TargetField))
		for i, part := range ch.TargetField {
			parts[i] = packScopedKey(root, part)
		}
		base = strings.Join(parts, "/") + "/"
	}
	var entries map[string]json.RawMessage
	if json.Unmarshal(ch.Entries, &entries) == nil {
		for _, key := range slices.Sorted(maps.Keys(entries)) {
			if key == "#-1" {
				// Content Patcher appends "#-1" entries to the list: any number of packs can.
				continue
			}
			out = append(out, Shape{Kind: 'p', Key: intern("entry:" + base + packScopedKey(root, key)), Value: intern(dataLiteral(entries[key], values))})
		}
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(ch.Fields, &fields) == nil {
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			var inner map[string]json.RawMessage
			if json.Unmarshal(fields[key], &inner) == nil && len(inner) > 0 {
				for _, field := range slices.Sorted(maps.Keys(inner)) {
					out = append(out, Shape{Kind: 'p', Key: intern("field:" + base + packScopedKey(root, key) + "." + field), Value: intern(dataLiteral(inner[field], values))})
				}
				continue
			}
			out = append(out, Shape{Kind: 'p', Key: intern("field:" + base + packScopedKey(root, key)), Value: intern(dataLiteral(fields[key], values))})
		}
	}
	return out
}

// packConfigValues are a pack's config values: its ConfigSchema defaults, then what config.json sets.
func packConfigValues(schema map[string]Schema, root string) map[string]string {
	values := make(map[string]string, len(schema))
	for key, field := range schema {
		values[strings.ToLower(key)] = field.DefaultValue
	}
	maps.Copy(values, ReadPackConfig(root))
	return values
}

// singleToken matches a value that is exactly one token, such as "{{Incubation time}}".
var singleToken = regexp.MustCompile(`^"\s*\{\{\s*([^:|}]+?)\s*\}\}\s*"$`)

// literalDigestOver is the length past which a data value is kept as a digest: overlaps only compare two values for
// equality, and an entry's whole JSON text per shape is most of what a parsed pack holds.
const literalDigestOver = 24

// compactLiteral keeps a short value as it is and a longer one as a digest of its lower-cased text, so two values
// compare equal exactly when their text does.
func compactLiteral(s string) string {
	if len(s) <= literalDigestOver {
		return s
	}
	sum := sha256.Sum256([]byte(strings.ToLower(s)))
	return "#" + hex.EncodeToString(sum[:12])
}

func dataLiteral(raw json.RawMessage, values map[string]string) string {
	return compactLiteral(rawDataLiteral(raw, values))
}

func rawDataLiteral(raw json.RawMessage, values map[string]string) string {
	if hasToken(string(raw)) {
		// A value that is just one of the pack's config tokens is that token's configured value,
		// so two packs set to the same value agree.
		if m := singleToken.FindStringSubmatch(strings.TrimSpace(string(raw))); m != nil {
			if v, ok := values[strings.ToLower(m[1])]; ok {
				return scalarLiteral(v)
			}
		}
		return ""
	}
	clean := bytes.TrimSpace(lenientjson.Clean(raw))
	if len(clean) == 0 || clean[0] == '{' || clean[0] == '[' {
		return string(clean)
	}
	var text string
	if clean[0] == '"' && json.Unmarshal(clean, &text) == nil {
		return scalarLiteral(text)
	}
	return scalarLiteral(string(clean))
}

// scalarLiteral spells a scalar data value one way whether a pack wrote it as JSON or as text: the game reads
// "true" and true, or "5" and 5, alike, and a config token always fills in text.
func scalarLiteral(v string) string {
	t := strings.TrimSpace(v)
	switch {
	case strings.EqualFold(t, "true"), strings.EqualFold(t, "false"):
		return strings.ToLower(t)
	case t != "" && (t[0] == '-' || (t[0] >= '0' && t[0] <= '9')) && json.Valid([]byte(t)):
		return t
	}
	return strconv.Quote(v)
}

// editShapes is what one EditImage or EditMap change writes. An edit that only adds warps, rewrites text
// properties or sets tile properties has no shape: Content Patcher merges those, so they never overwrite
// another pack's work.
func editShapes(root string, ch contentChange, image, skip bool) []Shape {
	if image {
		return editImageShapes(root, ch, skip)
	}
	var out []Shape
	if strings.TrimSpace(ch.FromFile) != "" {
		shapes, done := mapSourceShapes(root, ch)
		if done {
			return shapes
		}
		out = shapes
	}
	tiles, ok := mapTileShapes(ch.MapTiles)
	if !ok {
		return whole
	}
	out = append(out, tiles...)
	for key, raw := range ch.MapProperties {
		if hasToken(key) {
			return whole
		}
		value, _ := ScalarValue(raw)
		if hasToken(value) {
			value = ""
		}
		out = append(out, Shape{Kind: 'p', Key: intern(strings.ToLower(key)), Value: intern(compactLiteral(value))})
	}
	return out
}

func editImageShapes(root string, ch contentChange, skip bool) []Shape {
	overlay := strings.EqualFold(strings.TrimSpace(ch.PatchMode), "overlay")
	if to, ok := areaOf(ch.ToArea); ok {
		if overlay {
			return imagePatchShapes(root, ch, to.X, to.Y, skip)
		}
		return []Shape{to}
	} else if len(ch.ToArea) > 0 {
		return unplaced(ch.ToArea)
	}
	// Without ToArea the source lands at the top-left, sized like FromArea or the whole file.
	if from, ok := areaOf(ch.FromArea); ok {
		if overlay {
			return imagePatchShapes(root, ch, 0, 0, skip)
		}
		return []Shape{{Kind: 'r', W: from.W, H: from.H}}
	}
	if overlay || hasToken(ch.FromFile) {
		return imagePatchShapes(root, ch, 0, 0, skip)
	}
	if w, h, ok := pngSize(root, ch.FromFile); ok {
		return []Shape{{Kind: 'r', W: w, H: h}}
	}
	return whole
}

// mapSourceShapes is the area a map FromFile lands on. done means shapes is the whole answer for the edit.
func mapSourceShapes(root string, ch contentChange) (shapes []Shape, done bool) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(ch.FromFile)))
	if ext == ".tmx" || ext == ".tmj" {
		if decoded, ok := decodeMap(root, ch.FromFile); ok {
			return mapFileShapes(decoded, ch), true
		}
	}
	to, ok := areaOf(ch.ToArea)
	switch {
	case ok:
		return []Shape{to}, false
	case len(ch.ToArea) > 0:
		return unplaced(ch.ToArea), true
	}
	// Without ToArea the source map lands at the top-left at its own size (or FromArea's).
	if from, ok := areaOf(ch.FromArea); ok {
		return []Shape{{Kind: 'r', W: from.W, H: from.H}}, false
	}
	if w, h, ok := mapSize(root, ch.FromFile); ok {
		return []Shape{{Kind: 'r', W: w, H: h}}, false
	}
	return whole, true
}

// mapTileShapes is one shape per tile the edit replaces; ok is false when a tile cannot be placed.
func mapTileShapes(raws []json.RawMessage) ([]Shape, bool) {
	var out []Shape
	for _, raw := range raws {
		var tile struct {
			Position      json.RawMessage `json:"Position"`
			Layer         string          `json:"Layer"`
			SetIndex      json.RawMessage `json:"SetIndex"`
			SetTilesheet  json.RawMessage `json:"SetTilesheet"`
			Remove        json.RawMessage `json:"Remove"`
			SetProperties json.RawMessage `json:"SetProperties"`
		}
		if json.Unmarshal(raw, &tile) != nil {
			return nil, false
		}
		if len(tile.SetIndex) == 0 && len(tile.SetTilesheet) == 0 && len(tile.Remove) == 0 {
			continue
		}
		pos, ok := areaOf(tile.Position)
		if !ok || hasToken(tile.Layer) {
			return nil, false
		}
		out = append(out, Shape{Kind: 't', X: pos.X, Y: pos.Y, Layer: intern(strings.ToLower(strings.TrimSpace(tile.Layer)))})
	}
	return out, true
}

var (
	pngShapeCache sync.Map
	pngAlphaCache sync.Map
	cellSetCache  sync.Map
)

func imagePatchShapes(root string, ch contentChange, x, y int, skip bool) []Shape {
	files := sourceFiles(root, ch.FromFile)
	if len(files) == 0 {
		return whole
	}
	if skip || !strings.EqualFold(strings.TrimSpace(ch.PatchMode), "overlay") {
		var out []Shape
		for _, file := range files {
			if from, ok := areaOf(ch.FromArea); ok {
				out = append(out, Shape{Kind: 'r', X: x, Y: y, W: from.W, H: from.H})
				continue
			}
			if w, h, ok := pngSize(root, file); ok {
				out = append(out, Shape{Kind: 'r', X: x, Y: y, W: w, H: h})
			}
		}
		if len(out) > 0 {
			return out
		}
		return whole
	}
	var out []Shape
	for _, file := range files {
		if shape, ok := opaqueImageShape(root, file, ch.FromArea, x, y); ok && len(shape.Cells) > 0 {
			out = append(out, shape)
		}
	}
	if len(out) > 0 {
		return out
	}
	return whole
}

func opaqueImageShape(root, rel string, fromRaw json.RawMessage, x, y int) (Shape, bool) {
	fileKey, ok := pngFileKey(root, rel)
	if !ok {
		return Shape{}, false
	}
	fromKey := "full"
	var from Shape
	if len(fromRaw) > 0 {
		var parsed bool
		from, parsed = areaOf(fromRaw)
		if !parsed {
			return Shape{}, false
		}
		fromKey = strconv.Itoa(from.X) + "," + strconv.Itoa(from.Y) + "," +
			strconv.Itoa(from.W) + "," + strconv.Itoa(from.H)
	}
	key := fileKey + "\x00" + fromKey + "\x00" + strconv.Itoa(x) + "," + strconv.Itoa(y)
	if cached, ok := pngShapeCache.Load(key); ok {
		if cells, ok := cached.(string); ok {
			return Shape{Kind: 'r', Cells: cells}, true
		}
	}
	decoded, ok := loadPNGAlpha(root, rel, fileKey)
	if !ok {
		return Shape{}, false
	}
	src := image.Rectangle{Min: image.Pt(from.X, from.Y), Max: image.Pt(from.X+from.W, from.Y+from.H)}
	if fromKey == "full" {
		src = image.Rect(0, 0, decoded.Width, decoded.Height)
	}
	var cells strings.Builder
	for c := range decoded.Cells(src, image.Pt(x, y), 16) {
		cells.WriteByte(';')
		cells.WriteString(strconv.Itoa(c.X))
		cells.WriteByte(',')
		cells.WriteString(strconv.Itoa(c.Y))
	}
	pngShapeCache.Store(key, cells.String())
	return Shape{Kind: 'r', Cells: cells.String()}, true
}

// pngFileKey names one version of an image file, so a decode is reused only while the file is unchanged.
func pngFileKey(root, rel string) (string, bool) {
	abs, ok := inside(root, rel)
	if !ok {
		return "", false
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", false
	}
	return abs + "\x00" + strconv.FormatInt(info.Size(), 10) + "\x00" + strconv.FormatInt(info.ModTime().UnixNano(), 10), true
}

// decodeSlots bounds the image decodes running at once across every pack being read.
var decodeSlots = make(chan struct{}, max(1, runtime.GOMAXPROCS(0)))

// warmOverlayImages decodes the images of rel's overlay edits on every core before the serial scan reads their
// shapes: a recolour pack overlays hundreds of large sheets, and decoding them one by one made that single pack
// the longest part of a cold check.
func warmOverlayImages(root, rel string, changes []json.RawMessage, skip bool) {
	if skip {
		return
	}
	var wg sync.WaitGroup
	for _, raw := range changes {
		var ch contentChange
		if json.Unmarshal(raw, &ch) != nil || !strings.EqualFold(strings.TrimSpace(ch.Action), kindEditImage) ||
			!strings.EqualFold(strings.TrimSpace(ch.PatchMode), "overlay") {
			continue
		}
		for _, file := range sourceFiles(root, contentSourceReference(root, rel, ch.FromFile)) {
			wg.Go(func() {
				decodeSlots <- struct{}{}
				defer func() { <-decodeSlots }()
				if key, ok := pngFileKey(root, file); ok {
					_, _ = loadPNGAlpha(root, file, key)
				}
			})
		}
	}
	wg.Wait()
}

// loadPNGAlpha returns the image's alpha mask, or false when it cannot be decoded; a nil entry in pngAlphaCache
// memoises a failure.
func loadPNGAlpha(root, rel, fileKey string) (*tilemask.Mask, bool) {
	if cached, ok := pngAlphaCache.Load(fileKey); ok {
		if pix, ok := cached.(*tilemask.Mask); ok {
			return pix, pix != nil
		}
	}
	diskPath := alphaDiskPath(fileKey)
	var pix *tilemask.Mask
	if diskPath != "" {
		pix = readAlphaDisk(diskPath)
	}
	if pix == nil {
		file, err := os.OpenInRoot(root, filepath.FromSlash(rel))
		if err != nil {
			pngAlphaCache.Store(fileKey, (*tilemask.Mask)(nil))
			return nil, false
		}
		pix, err = tilemask.DecodePNG(safefs.CapReader(file, footprintReadCap), tilemask.Options{})
		_ = file.Close()
		if err != nil {
			pngAlphaCache.Store(fileKey, (*tilemask.Mask)(nil))
			return nil, false
		}
		if diskPath != "" {
			writeAlphaDisk(diskPath, pix)
		}
	}
	actual, loaded := pngAlphaCache.LoadOrStore(fileKey, pix)
	if loaded {
		if kept, ok := actual.(*tilemask.Mask); ok {
			return kept, kept != nil
		}
		return nil, false
	}
	return pix, true
}

// footprintReadCap bounds each map or overlay file read, matching the pack file cap.
var footprintReadCap int64 = maxPackFileBytes

func readCappedFile(path string) ([]byte, error) {
	file, err := safefs.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return safefs.ReadCapped(file, footprintReadCap)
}

// pngDecodable reports whether raw's declared size is within the pixel cap, reading only the header.
func pngDecodable(raw []byte) bool {
	cfg, err := png.DecodeConfig(bytes.NewReader(raw))
	return err == nil && int64(cfg.Width) <= tilemask.DefaultMaxPixels/max(int64(cfg.Height), 1)
}

func sourceFiles(root, rel string) []string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return nil
	}
	if !hasToken(rel) {
		if relative, ok := foldedRel(root, rel); ok {
			return []string{relative}
		}
		return nil
	}
	var out []string
	walkSourceParts(strings.Split(filepath.ToSlash(rel), "/"), root, 0, "", &out)
	return out
}

// globToken replaces each {{token}} in a path part with a wildcard.
func globToken(part string) string {
	for {
		start := strings.Index(part, "{{")
		if start < 0 {
			return part
		}
		end := strings.Index(part[start+2:], "}}")
		if end < 0 {
			return part
		}
		end += start + 2
		part = part[:start] + "*" + part[end+2:]
	}
}

func walkSourceParts(parts []string, dir string, index int, prefix string, out *[]string) {
	if index == len(parts) {
		*out = append(*out, strings.TrimPrefix(filepath.ToSlash(prefix), "./"))
		return
	}
	part := parts[index]
	if hasToken(part) {
		part = globToken(part)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	last := index == len(parts)-1
	for _, entry := range entries {
		// Only the final part names a file; the parts before it are folders.
		if !matchFold(part, entry.Name()) || entry.IsDir() == last {
			continue
		}
		walkSourceParts(parts, filepath.Join(dir, entry.Name()), index+1, filepath.Join(prefix, entry.Name()), out)
	}
}

func matchFold(pattern, value string) bool {
	pattern, value = strings.ToLower(pattern), strings.ToLower(value)
	matched, err := filepath.Match(pattern, value)
	return err == nil && matched
}

// unplaced is the shape of an area whose position is tokenized: the whole asset, marked tiny when its literal
// size is at most two tiles (a trapdoor or a sign placed by a computed position).
func unplaced(raw json.RawMessage) []Shape {
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
		return []Shape{{Kind: 'w', Tiny: true}}
	}
	return whole
}

// areaOf reads {X, Y, Width, Height} (or a Position's {X, Y}) given as numbers or numeric strings; false when
// it is missing or holds a token.
func areaOf(raw json.RawMessage) (Shape, bool) {
	if len(raw) == 0 {
		return Shape{}, false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return Shape{}, false
	}
	s := Shape{Kind: 'r', W: 1, H: 1}
	for k, v := range fields {
		var n int
		var str string
		switch {
		case json.Unmarshal(v, &n) == nil:
		case json.Unmarshal(v, &str) == nil:
			parsed, err := strconv.Atoi(strings.TrimSpace(str))
			if err != nil {
				return Shape{}, false
			}
			n = parsed
		default:
			return Shape{}, false
		}
		switch strings.ToLower(k) {
		case "x":
			s.X = n
		case "y":
			s.Y = n
		case "width":
			s.W = n
		case "height":
			s.H = n
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

type decodedMapResult struct {
	Value tilemask.Map
	Ok    bool
}

var mapCache sync.Map

// mapFormat is the tilemask format of a .tmx or .tmj path.
func mapFormat(rel string) (tilemask.Format, bool) {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".tmx":
		return tilemask.TMX, true
	case ".tmj":
		return tilemask.TMJ, true
	}
	return 0, false
}

// decodeMap reads a map's occupied tiles; layer names come back lower-cased and trimmed.
func decodeMap(root, rel string) (tilemask.Map, bool) {
	abs, ok := inside(root, rel)
	format, known := mapFormat(rel)
	if !ok || !known {
		return tilemask.Map{}, false
	}
	info, err := os.Stat(abs)
	if err != nil {
		return tilemask.Map{}, false
	}
	key := abs + "\x00" + strconv.FormatInt(info.Size(), 10) + "\x00" +
		strconv.FormatInt(info.ModTime().UnixNano(), 10)
	if cached, ok := mapCache.Load(key); ok {
		result, ok := cached.(decodedMapResult)
		if ok {
			return result.Value, result.Ok
		}
	}
	raw, err := readCappedFile(abs)
	if err != nil {
		return tilemask.Map{}, false
	}
	decoded, err := tilemask.DecodeMap(bytes.NewReader(raw), format, tilemask.Options{})
	if err != nil {
		mapCache.Store(key, decodedMapResult{})
		return tilemask.Map{}, false
	}
	for i := range decoded.Layers {
		decoded.Layers[i].Name = strings.ToLower(strings.TrimSpace(decoded.Layers[i].Name))
	}
	mapCache.Store(key, decodedMapResult{Value: decoded, Ok: true})
	return decoded, true
}

func mapFileShapes(m tilemask.Map, ch contentChange) []Shape {
	fallback := Shape{Kind: 'r', W: m.Width, H: m.Height}
	if fallback.W <= 0 || fallback.H <= 0 {
		return nil
	}
	from, ok := mapArea(ch.FromArea, fallback)
	if !ok {
		return whole
	}
	toFallback := fallback
	toFallback.X, toFallback.Y = 0, 0
	to, ok := mapArea(ch.ToArea, toFallback)
	if !ok {
		return unplaced(ch.ToArea)
	}
	if len(m.Layers) == 0 {
		to.Layer = ""
		return []Shape{to}
	}
	mode := strings.ToLower(strings.TrimSpace(ch.PatchMode))
	mode = cmp.Or(mode, "replacebylayer")
	switch mode {
	case "overlay":
		var out []Shape
		for _, layer := range m.Layers {
			for _, tile := range layer.Tiles {
				if tile.X < from.X || tile.X >= from.X+from.W || tile.Y < from.Y || tile.Y >= from.Y+from.H {
					continue
				}
				out = append(out, Shape{
					Kind: 't', X: to.X + tile.X - from.X, Y: to.Y + tile.Y - from.Y,
					Layer: layer.Name,
				})
			}
		}
		return out
	case "replace":
		to.Layer = ""
		return []Shape{to}
	case "replacebylayer":
		var out []Shape
		for _, layer := range m.Layers {
			area := to
			area.Layer = layer.Name
			out = append(out, area)
		}
		return out
	default:
		var out []Shape
		for _, layer := range m.Layers {
			area := to
			area.Layer = layer.Name
			out = append(out, area)
		}
		return out
	}
}

func mapArea(raw json.RawMessage, fallback Shape) (Shape, bool) {
	if len(raw) == 0 {
		return fallback, true
	}
	area, ok := areaOf(raw)
	if !ok {
		return Shape{}, false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return Shape{}, false
	}
	if !hasAreaField(fields, "x") {
		area.X = fallback.X
	}
	if !hasAreaField(fields, "y") {
		area.Y = fallback.Y
	}
	if !hasAreaField(fields, "width") {
		area.W = fallback.W
	}
	if !hasAreaField(fields, "height") {
		area.H = fallback.H
	}
	return area, area.W > 0 && area.H > 0
}

func hasAreaField(fields map[string]json.RawMessage, want string) bool {
	for key := range fields {
		if strings.EqualFold(strings.TrimSpace(key), want) {
			return true
		}
	}
	return false
}

// mapSize reads a .tmx or .tmj map's size in tiles; .tbin and tokenized paths are unknown.
func mapSize(root, rel string) (w, h int, ok bool) {
	rel = strings.TrimSpace(rel)
	format, known := mapFormat(rel)
	if rel == "" || hasToken(rel) || !known {
		return 0, 0, false
	}
	abs, inRoot := inside(root, rel)
	if !inRoot {
		return 0, 0, false
	}
	raw, err := readCappedFile(abs)
	if err != nil {
		return 0, 0, false
	}
	w, h, err = tilemask.MapSize(bytes.NewReader(raw), format)
	return w, h, err == nil
}

// placeTokens are conditions that hold one value at a time for the player: two edits that need disjoint
// values of the same one never apply together (an edit for the East Scarp village and one for another map).
var placeTokens = map[string]bool{"locationname": true, "locationcontext": true, "season": true, "weather": true, "dayofweek": true, "farmtype": true, "day": true, "dayevent": true}

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
			values = SplitTargets(list)
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

var (
	spouseQuery    = regexp.MustCompile(`(?i)^'\{\{\s*([^{}\s]+)\s*\}\}'\s*=\s*'([^'{}]+)'$`)
	spouseQueryRev = regexp.MustCompile(`(?i)^'([^'{}]+)'\s*=\s*'\{\{\s*([^{}\s]+)\s*\}\}'$`)
	queryAnd       = regexp.MustCompile(`(?i)\s+and\s+`)
	queryOr        = regexp.MustCompile(`(?i)\bor\b|\|\|`)
)

// spouseOf is the NPC a When block requires the player to be engaged or married to ("Relationship:Abigail":
// "Engaged, Married", or a Query that querySpouse reads, written in the condition or as the value of a dynamic
// token the condition needs true), or "". A player has one partner at a time, so two patches that need
// different partners never apply together.
func spouseOf(raw map[string]json.RawMessage, tokens []TokenDefinition) string {
	for k, v := range raw {
		if npc := spouseOfCondition(k, v, tokens); npc != "" {
			return npc
		}
	}
	return ""
}

func spouseOfCondition(k string, v json.RawMessage, tokens []TokenDefinition) string {
	key := strings.TrimSpace(k)
	if inner, ok := strings.CutPrefix(key, "{{"); ok {
		if key, ok = strings.CutSuffix(inner, "}}"); !ok {
			return ""
		}
		key = strings.TrimSpace(key)
	}
	if strings.EqualFold(key, "spouse") {
		var npc string
		if json.Unmarshal(v, &npc) == nil && !hasToken(npc) && strings.TrimSpace(npc) != "" {
			return strings.ToLower(strings.TrimSpace(npc))
		}
		return ""
	}
	if name, expr, ok := strings.Cut(key, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "query") {
		var on bool
		if json.Unmarshal(v, &on) != nil || !on {
			return ""
		}
		return querySpouse(expr, tokens)
	}
	var flags []string
	if len(tokens) > 0 && !strings.ContainsAny(key, ":|") && condValues(v, &flags) && len(flags) == 1 && strings.EqualFold(flags[0], "true") {
		if npc := tokenSpouse(tokenName(key), tokens); npc != "" {
			return npc
		}
	}
	name, npc, ok := strings.Cut(key, ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(name), "relationship") || hasToken(npc) {
		return ""
	}
	var states []string
	if condValues(v, &states) && !slices.ContainsFunc(states, func(s string) bool {
		return !strings.EqualFold(s, "married") && !strings.EqualFold(s, "engaged")
	}) {
		return strings.ToLower(strings.TrimSpace(npc))
	}
	return ""
}

// querySpouse is the partner an AND-only Query expression compares {{Spouse}}, or a token that is just
// querySpouse is the partner a Query expression requires: every OR branch compares {{Spouse}} or {{Roommate}}
// (or a token that is just {{Spouse}}) with the same NPC in one of its AND terms. A player lives with one partner
// at a time, so a branch per partner token still names one partner.
func querySpouse(expr string, tokens []TokenDefinition) string {
	partner := ""
	for _, branch := range queryOr.Split(strings.TrimSpace(expr), -1) {
		npc := branchSpouse(branch, tokens)
		if npc == "" || partner != "" && npc != partner {
			return ""
		}
		partner = npc
	}
	return partner
}

func branchSpouse(branch string, tokens []TokenDefinition) string {
	for _, term := range queryAnd.Split(strings.TrimSpace(branch), -1) {
		term = strings.TrimSpace(term)
		var token, npc string
		if m := spouseQuery.FindStringSubmatch(term); m != nil {
			token, npc = m[1], m[2]
		} else if m := spouseQueryRev.FindStringSubmatch(term); m != nil {
			token, npc = m[2], m[1]
		} else {
			continue
		}
		if strings.EqualFold(token, "spouse") || strings.EqualFold(token, "roommate") || spouseAlias(tokenName(token), tokens) {
			return strings.ToLower(strings.TrimSpace(npc))
		}
	}
	return ""
}

// tokenSpouse is the partner a dynamic token is true for when every definition of it is a Query that names
// one partner, such as "LivingWithSen": "{{Query: '{{Spouse}}' = 'SenS' OR '{{Roommate}}' = 'SenS'}}".
func tokenSpouse(name string, tokens []TokenDefinition) string {
	partner := ""
	for _, definition := range tokens {
		if definition.Name != name {
			continue
		}
		value := strings.TrimSpace(definition.Value)
		inner, ok := strings.CutPrefix(value, "{{")
		if inner, ok2 := strings.CutSuffix(inner, "}}"); ok && ok2 {
			if head, expr, ok := strings.Cut(inner, ":"); ok && strings.EqualFold(strings.TrimSpace(head), "query") {
				if npc := querySpouse(expr, tokens); npc != "" && (partner == "" || npc == partner) {
					partner = npc
					continue
				}
			}
		}
		return ""
	}
	return partner
}

// withTokenFarmTypes pins a patch to the farm types its dynamic token conditions can only be true on, which a
// FarmType condition written in the patch itself does not show: once config elimination has dropped the
// definitions that cannot hold, every definition left that yields a wanted value is gated on a FarmType.
func withTokenFarmTypes(p Patch, tokens []TokenDefinition, present map[string]bool, schema map[string]Schema, config map[string]string) Patch {
	if len(p.When.Dynamic) == 0 || len(p.Places["farmtype"]) > 0 {
		return p
	}
	reachable := cachedReachableTokens(tokens, present, schema, config, p.When.Flags)
	for _, condition := range p.When.Dynamic {
		var types []string
		pinned := true
		for _, definition := range tokens {
			if definition.Name != condition.Name {
				continue
			}
			if hasToken(definition.Value) {
				pinned = false
				break
			}
			if !dynamicConditionMatchesValue(condition, definition.Value) ||
				dynamicWhenState(definition.When, tokens, reachable, present, schema, config, p.When.Flags) == cpConditionFalse {
				continue
			}
			farms := placesOf(definition.When)["farmtype"]
			if len(farms) == 0 {
				pinned = false
				break
			}
			types = append(types, farms...)
		}
		if pinned && len(types) > 0 {
			places := maps.Clone(p.Places)
			if places == nil {
				places = map[string][]string{}
			}
			places["farmtype"] = types
			p.Places = places
			return p
		}
	}
	return p
}

func spouseAlias(name string, tokens []TokenDefinition) bool {
	found := false
	for _, definition := range tokens {
		if definition.Name != name {
			continue
		}
		if !strings.EqualFold(strings.Join(strings.Fields(definition.Value), ""), "{{spouse}}") {
			return false
		}
		found = true
	}
	return found
}

// exclusive reports whether two edits can never be active together.
func exclusive(a, b Patch) bool {
	if a.Spouse != "" && b.Spouse != "" && a.Spouse != b.Spouse {
		return true
	}
	for token, values := range a.Places {
		other, ok := b.Places[token]
		if ok && !slices.ContainsFunc(values, func(v string) bool { return slices.Contains(other, v) }) {
			return true
		}
	}
	// One gated on a farm type and one on a farm map only another farm type loads.
	_, aFarm := a.Places["farmtype"]
	_, bFarm := b.Places["farmtype"]
	if aFarm != bFarm {
		x, y := farmTypesOf(a, ""), farmTypesOf(b, "")
		return len(x) > 0 && len(y) > 0 && !slices.ContainsFunc(x, func(v string) bool { return slices.Contains(y, v) })
	}
	return false
}

// farmMaps are the vanilla farm maps and the FarmType value of the farm that loads each.
var farmMaps = map[string]string{
	"maps/farm": "standard", "maps/farm_fishing": "riverland", "maps/farm_foraging": "forest", "maps/farm_mining": "hilltop",
	"maps/farm_combat": "wilderness", "maps/farm_fourcorners": "fourcorners", "maps/farm_island": "beach", "maps/farm_ranch": "meadowlandsfarm",
}

// farmTypesOf is the farm types a patch applies on, lower-cased: its FarmType condition, else the farm whose
// map it targets (custom is the id of the custom farm whose map is the target), else nil for any farm.
func farmTypesOf(p Patch, custom string) []string {
	if types := p.Places["farmtype"]; len(types) > 0 {
		return types
	}
	if t, ok := farmMaps[p.Target]; ok {
		return []string{t}
	}
	if custom != "" {
		return []string{custom}
	}
	return nil
}

// editsClash reports whether any active edit of one pack can overwrite one of the other's, and whether every
// such overlap is harmless (see harmless).
func editsClash(a, b []Patch) (clash, minor bool) {
	clash, minor, _ = editsClashIndexed(a, b, indexesOf(b), false)
	return clash, minor
}

// indexesOf is the shape index of each patch, looked up once so a pair loop does not repeat it per comparison.
func indexesOf(patches []Patch) []*shapeSet {
	out := make([]*shapeSet, len(patches))
	for i, p := range patches {
		out[i] = shapeIndex(p.Shapes)
	}
	return out
}

// editsClashIndexed is editsClash with bIndex[i] the shape index of b[i], and the note that explains why a
// minor clash is harmless.
// One author ordering two of their packs' edits with an explicit priority meant the result.
func editsClashIndexed(a, b []Patch, bIndex []*shapeSet, sameAuthor bool) (clash, minor bool, why *Note) {
	minor = true
	var note noteAgreement
	for _, x := range a {
		for j, y := range b {
			if mapOverlayHasUnknownLayer(x, y) {
				continue
			}
			if x.Image && y.Image && x.ImageDigest != "" && x.ImageDigest == y.ImageDigest && x.ImageFromArea == y.ImageFromArea {
				continue
			}
			if x.Image && y.Image && strings.EqualFold(strings.TrimSpace(x.PatchMode), "overlay") && strings.EqualFold(strings.TrimSpace(y.PatchMode), "overlay") {
				continue
			}
			if exclusive(x, y) || sameAuthor && (explicitOrder(x) || explicitOrder(y)) {
				continue
			}
			if bIndex[j].overlaps(x.Shapes) {
				clash = true
				ok, why := harmless(x, y, bIndex[j])
				// One overlap that matters settles it: nothing later can make the pair minor again.
				if !ok {
					return true, false, nil
				}
				note.add(why)
			}
		}
	}
	return clash, clash && minor, note.result()
}

// explicitOrder is an edit given a non-default Content Patcher priority, set to run before or after the others.
func explicitOrder(p Patch) bool {
	return contentPatcherPriority("edit", p.Priority) != 0
}

// noteAgreement keeps a ConflictNote only while every harmless overlap gives the same one.
type noteAgreement struct {
	Note  *Note
	Seen  bool
	Mixed bool
}

func (a *noteAgreement) add(n *Note) {
	switch {
	case !a.Seen:
		a.Note, a.Seen = n, true
	case n == nil || a.Note == nil || *n != *a.Note:
		a.Mixed = true
	}
}

func (a *noteAgreement) result() *Note {
	if a.Mixed {
		return nil
	}
	return a.Note
}

func mapOverlayHasUnknownLayer(a, b Patch) bool {
	if a.Image || b.Image {
		return false
	}
	aOverlay := strings.EqualFold(strings.TrimSpace(a.PatchMode), "overlay")
	bOverlay := strings.EqualFold(strings.TrimSpace(b.PatchMode), "overlay")
	return (aOverlay && hasUnknownMapLayer(b)) || (bOverlay && hasUnknownMapLayer(a))
}

func hasUnknownMapLayer(p Patch) bool {
	return slices.ContainsFunc(p.Shapes, func(shape Shape) bool {
		return shape.Kind != 'p' && shape.Layer == ""
	})
}

// harmless reports an overlap that cannot hurt play: two image edits only change how something looks; an edit
// that applies in one location or weather, or on one day or festival, only matters there; two that only set
// prices let one mod's numbers win; and a one-tile edit at a computed spot is too small to place, so it is
// shown without counting as a problem.
func harmless(x, y Patch, yIndex *shapeSet) (bool, *Note) {
	if balanceOverlap(x, y, yIndex) {
		return true, &Note{Kind: "balance"}
	}
	for _, p := range []Patch{x, y} {
		if day := oneDay(p); day != "" {
			return true, &Note{Kind: "day", Value: day}
		}
	}
	return (x.Image && y.Image) || (textOnly(x) && textOnly(y)) || overlayPriorityHarmless(x, y) || situational(x) || situational(y) || tinyOnly(x) || tinyOnly(y), nil
}

// balanceFields are data fields that only tune prices and costs: when two collide, one mod's numbers win.
var balanceFields = map[string]bool{"buildcost": true, "buildmaterials": true, "purchaseprice": true, "sellprice": true, "price": true, "pricemodifiers": true}

// balanceOverlap is two data edits that only collide on balance fields (yIndex indexes y's shapes); what
// else either sets, the other leaves alone.
func balanceOverlap(x, y Patch, yIndex *shapeSet) bool {
	if x.Action != kindEditData || y.Action != kindEditData {
		return false
	}
	return !slices.ContainsFunc(x.Shapes, func(s Shape) bool {
		return !balanceKey(s.Key) && slices.ContainsFunc(yIndex.Props[s.Key], s.overlaps)
	})
}

func balanceKey(key string) bool {
	for part := range strings.FieldsFuncSeq(key[strings.Index(key, ":")+1:], func(r rune) bool { return r == '/' || r == '.' }) {
		if balanceFields[strings.ToLower(part)] {
			return true
		}
	}
	return false
}

// textOnly is a data edit that only replaces lines of text: when two collide, one mod's line shows
// instead of the other's and nothing breaks. Festival files also hold set-up, which is not text.
func textOnly(p Patch) bool {
	if p.Action != kindEditData {
		return false
	}
	t := strings.ToLower(p.Target)
	switch {
	case strings.HasPrefix(t, "characters/dialogue/"), strings.HasPrefix(t, "strings/"), t == "data/extradialogue":
		return true
	case strings.HasPrefix(t, "data/festivals/"):
		return !slices.ContainsFunc(p.Shapes, func(s Shape) bool {
			k := strings.ToLower(s.Key)
			return strings.Contains(k, "set-up") || strings.Contains(k, "mainevent") || strings.Contains(k, "shop") || strings.Contains(k, "conditions")
		})
	}
	return false
}

func overlayPriorityHarmless(x, y Patch) bool {
	xOverlay := strings.EqualFold(strings.TrimSpace(x.PatchMode), "overlay")
	yOverlay := strings.EqualFold(strings.TrimSpace(y.PatchMode), "overlay")
	if xOverlay == yOverlay {
		return false
	}
	if xOverlay {
		return contentPatcherPriority("edit", x.Priority) > contentPatcherPriority("edit", y.Priority)
	}
	return contentPatcherPriority("edit", y.Priority) > contentPatcherPriority("edit", x.Priority)
}

func situational(p Patch) bool {
	_, location := p.Places["locationname"]
	_, context := p.Places["locationcontext"]
	_, weather := p.Places["weather"]
	return location || context || weather
}

// oneDay names the single day ("Summer 28") or festival a patch is limited to, or "".
func oneDay(p Patch) string {
	if events := p.Places["dayevent"]; len(events) == 1 {
		return titleWords(events[0])
	}
	season, day := p.Places["season"], p.Places["day"]
	if len(season) == 1 && len(day) == 1 {
		return titleWords(season[0]) + " " + day[0]
	}
	return ""
}

func titleWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func tinyOnly(p Patch) bool {
	return len(p.Shapes) > 0 && !slices.ContainsFunc(p.Shapes, func(s Shape) bool { return !s.Tiny })
}

// markClashes records which edits of a and b overlap each other.
func markClashes(a, b *packHit) {
	ordered := sameAuthor(*a, *b)
	for i, x := range a.Edits {
		for j, y := range b.Edits {
			if exclusive(x, y) || ordered && (explicitOrder(x) || explicitOrder(y)) || !shapesOverlap(x.Shapes, y.Shapes) {
				continue
			}
			if a.Clashes == nil {
				a.Clashes = map[int]bool{}
			}
			if b.Clashes == nil {
				b.Clashes = map[int]bool{}
			}
			a.Clashes[i], b.Clashes[j] = true, true
		}
	}
}

// fieldGateStats counts the pack's patches, those gated on field, and those still applying once field is value.
func fieldGateStats(h packHit, field Schema, value string) (gated, total, left int) {
	config := maps.Clone(h.Config)
	config[strings.ToLower(field.Key)] = value
	for _, p := range h.Tokens {
		if p.Kind == "other" {
			continue
		}
		total++
		if slices.ContainsFunc(p.When.Config, func(o Config) bool { return strings.EqualFold(o.Field, field.Key) }) {
			gated++
		}
		if ConfigHolds(p.When.Config, h.Schema, config) {
			left++
		}
	}
	return gated, total, left
}

// switchOff finds an on/off field that every clashing edit of the pack needs; its other value removes the pack
// from the conflict, such as Better Things' DesertMinecart for an expansion that redraws the desert. Of the
// fields that would, it offers the narrowest: one named for what the target is about, then the one gating
// the fewest of the pack's patches. When only the pack's master switch would, it offers nothing.
func switchOff(h packHit, peerSets ...[]packHit) (Fix, bool) {
	if len(h.Clashes) == 0 {
		return Fix{}, false
	}
	clashing := slices.Sorted(maps.Keys(h.Clashes))
	subject := h.Edits[clashing[0]].Target
	subject = strings.ToLower(subject[strings.LastIndex(subject, "/")+1:])
	var best Fix
	bestGated, bestNamed, found := 0, false, false
	for _, c := range h.Edits[clashing[0]].When.Config {
		field, ok := h.Schema[strings.ToLower(c.Field)]
		if !ok || !field.Toggle() {
			continue
		}
		value, ok := offValue(h, clashing, field, peerSets...)
		if !ok {
			continue
		}
		named := strings.Contains(strings.ToLower(field.Key), subject)
		gated, total, left := fieldGateStats(h, field, value)
		// A field that switches off the whole pack, or a fifth of it without being named for this target, is
		// the pack's master switch (Better Water's Color, a schedule pack's PlotSchedules): offering it
		// would trade the conflict for most of the mod.
		// ponytail: fixed one-fifth share of patches; one offered fix in fourteen sits within 10% of it on the
		// maintainer's profile, so weigh by what the gated patches touch only if a pack is misjudged.
		if left == 0 || !named && gated*5 >= total {
			continue
		}
		if found && (bestNamed && !named || bestNamed == named && (gated > bestGated || gated == bestGated && field.Key >= best.Field)) {
			continue
		}
		current, set := h.Config[strings.ToLower(field.Key)]
		if !set {
			current = field.DefaultValue
		}
		best = Fix{Key: h.Key, ID: h.ID, Name: h.Name, Field: field.Key, Current: current, Value: value}
		bestGated, bestNamed, found = gated, named, true
	}
	return best, found
}

// offValue is the one value of field that switches off every clashing edit of h without leaving another of
// its edits clashing with peerSets[0].
func offValue(h packHit, clashing []int, field Schema, peerSets ...[]packHit) (string, bool) {
	needed := map[string]bool{}
	for _, i := range clashing {
		j := slices.IndexFunc(h.Edits[i].When.Config, func(o Config) bool { return strings.EqualFold(o.Field, field.Key) })
		if j < 0 {
			return "", false
		}
		for _, v := range h.Edits[i].When.Config[j].Values {
			needed[strings.ToLower(v)] = true
		}
	}
	values := field.AllowValues
	if len(values) == 0 {
		values = []string{"true", "false"}
	}
	var off []string
	for _, v := range values {
		if !needed[strings.ToLower(strings.TrimSpace(v))] {
			off = append(off, strings.TrimSpace(v))
		}
	}
	if len(peerSets) > 0 {
		safe := off[:0]
		for _, value := range off {
			if !settingStillClashes(h, peerSets[0], field, value) {
				safe = append(safe, value)
			}
		}
		off = safe
	}
	if len(off) != 1 {
		return "", false
	}
	return off[0], true
}

func settingStillClashes(h packHit, peers []packHit, field Schema, value string) bool {
	config := make(map[string]string, len(h.Config))
	maps.Copy(config, h.Config)
	config[strings.ToLower(field.Key)] = value
	var active []Patch
	for _, edit := range h.Eligible {
		if ConfigHolds(edit.When.Config, h.Schema, config) {
			active = append(active, edit)
		}
	}
	if len(h.Eligible) == 0 {
		active = h.Edits
	}
	for _, peer := range peers {
		if Equal(peer.ID, h.ID) {
			continue
		}
		if clash, _ := editsClash(active, peer.Edits); clash {
			return true
		}
	}
	return false
}
