package contentpatcher

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"compress/zlib"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"image"
	"image/png"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/jsonc"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// cpShape is part of a target one edit writes: an area (image pixels, or map tiles on every layer), one map
// tile on one layer, one map property, or the whole asset when the area cannot be known without running
// Content Patcher (tokens, a map patched from a file without ToArea).
type cpShape struct {
	kind       byte // 'r' area, 't' tile, 'p' property, 'w' whole
	x, y, w, h int
	cells      string
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
	if s.cells != "" || o.cells != "" {
		if s.cells != "" && o.cells != "" {
			return cellSetsOverlap(s.cells, o.cells)
		}
		cells, area := s.cells, o
		if cells == "" {
			cells, area = o.cells, s
		}
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
	ax, ay, aw, ah := s.area()
	bx, by, bw, bh := o.area()
	return ax < bx+bw && bx < ax+aw && ay < by+bh && by < ay+ah
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
	props map[string][]cpShape
	rest  []cpShape
}

func indexShapes(shapes []cpShape) *shapeSet {
	set := &shapeSet{props: map[string][]cpShape{}}
	for _, s := range shapes {
		if s.kind == 'p' {
			set.props[s.key] = append(set.props[s.key], s)
		} else {
			set.rest = append(set.rest, s)
		}
	}
	return set
}

// shapeIndexes holds each patch's shapeSet for the running Check, keyed by the patch's shapes backing array, so a
// target edited by many packs indexes each patch once.
var shapeIndexes = struct {
	sync.Mutex
	sets map[*cpShape]*shapeSet
}{sets: map[*cpShape]*shapeSet{}}

func shapeIndex(shapes []cpShape) *shapeSet {
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
	props map[string]struct{}
	rest  bool
}

func summarize(index []*shapeSet) shapeSummary {
	s := shapeSummary{props: map[string]struct{}{}}
	for _, set := range index {
		for k := range set.props {
			s.props[k] = struct{}{}
		}
		s.rest = s.rest || len(set.rest) > 0
	}
	return s
}

// mayOverlap is false only when no shape of one pack can overlap a shape of the other: property shapes overlap only
// on the same key, and every other shape only another non-property shape.
func (s shapeSummary) mayOverlap(o shapeSummary) bool {
	if s.rest && o.rest {
		return true
	}
	small, large := s.props, o.props
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

func (set *shapeSet) overlaps(shapes []cpShape) bool {
	for _, x := range shapes {
		others := set.rest
		if x.kind == 'p' {
			others = set.props[x.key]
		}
		if slices.ContainsFunc(others, x.overlaps) {
			return true
		}
	}
	return false
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

// packScopedKey keeps a data key that holds a token, such as {{ModId}} or a pack's own config token,
// from matching another pack's identical text: each pack fills its tokens with its own values.
func packScopedKey(root, key string) string {
	if hasToken(key) {
		return "@" + root + "|" + key
	}
	return key
}

func dataShapes(root string, ch cpChange, values map[string]string) []cpShape {
	var out []cpShape
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
			out = append(out, cpShape{kind: 'p', key: "entry:" + base + packScopedKey(root, key), value: dataLiteral(entries[key], values)})
		}
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(ch.Fields, &fields) == nil {
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			var inner map[string]json.RawMessage
			if json.Unmarshal(fields[key], &inner) == nil && len(inner) > 0 {
				for _, field := range slices.Sorted(maps.Keys(inner)) {
					out = append(out, cpShape{kind: 'p', key: "field:" + base + packScopedKey(root, key) + "." + field, value: dataLiteral(inner[field], values)})
				}
				continue
			}
			out = append(out, cpShape{kind: 'p', key: "field:" + base + packScopedKey(root, key), value: dataLiteral(fields[key], values)})
		}
	}
	return out
}

// packConfigValues are a pack's config values: its ConfigSchema defaults, then what config.json sets.
func packConfigValues(schema map[string]cpSchema, root string) map[string]string {
	values := make(map[string]string, len(schema))
	for key, field := range schema {
		values[strings.ToLower(key)] = field.defaultValue
	}
	maps.Copy(values, readPackConfig(root))
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
	clean := bytes.TrimSpace(jsonc.Clean(raw))
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
func editShapes(root string, ch cpChange, image bool) []cpShape {
	if image {
		if to, ok := areaOf(ch.ToArea); ok {
			if strings.EqualFold(strings.TrimSpace(ch.PatchMode), "overlay") {
				return imagePatchShapes(root, ch, to.x, to.y)
			}
			return []cpShape{to}
		} else if len(ch.ToArea) > 0 {
			return unplaced(ch.ToArea)
		}
		// Without ToArea the source lands at the top-left, sized like FromArea or the whole file.
		if from, ok := areaOf(ch.FromArea); ok {
			if strings.EqualFold(strings.TrimSpace(ch.PatchMode), "overlay") {
				return imagePatchShapes(root, ch, 0, 0)
			}
			return []cpShape{{kind: 'r', w: from.w, h: from.h}}
		}
		if strings.EqualFold(strings.TrimSpace(ch.PatchMode), "overlay") {
			return imagePatchShapes(root, ch, 0, 0)
		}
		if hasToken(ch.FromFile) {
			return imagePatchShapes(root, ch, 0, 0)
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
		out = append(out, cpShape{kind: 'p', key: strings.ToLower(key), value: compactLiteral(value)})
	}
	return out
}

var (
	pngShapeCache sync.Map
	pngAlphaCache sync.Map
	pngPixPools   sync.Map
	cellSetCache  sync.Map
	// SkipImageOverlap turns off the pixel-overlap scan of image edits, trading accuracy for speed.
	SkipImageOverlap bool
)

type pngAlpha struct {
	w, h int
	a    []byte
}

func pngPixPool(n int) *sync.Pool {
	actual, _ := pngPixPools.LoadOrStore(n, &sync.Pool{New: func() any {
		b := make([]byte, n)
		return &b
	}})
	p, ok := actual.(*sync.Pool)
	if !ok {
		p = &sync.Pool{New: func() any {
			b := make([]byte, n)
			return &b
		}}
		pngPixPools.Store(n, p)
	}
	return p
}

func acquireAlpha(n int) []byte {
	p := pngPixPool(n)
	got := p.Get()
	buf, ok := got.(*[]byte)
	if !ok || cap(*buf) < n {
		b := make([]byte, n)
		return b
	}
	*buf = (*buf)[:n]
	return *buf
}

func releaseAlpha(b []byte) {
	if b == nil {
		return
	}
	pngPixPool(cap(b)).Put(&b)
}

func imagePatchShapes(root string, ch cpChange, x, y int) []cpShape {
	files := sourceFiles(root, ch.FromFile)
	if len(files) == 0 {
		return whole
	}
	if SkipImageOverlap || !strings.EqualFold(strings.TrimSpace(ch.PatchMode), "overlay") {
		var out []cpShape
		for _, file := range files {
			if from, ok := areaOf(ch.FromArea); ok {
				out = append(out, cpShape{kind: 'r', x: x, y: y, w: from.w, h: from.h})
				continue
			}
			if w, h, ok := pngSize(root, file); ok {
				out = append(out, cpShape{kind: 'r', x: x, y: y, w: w, h: h})
			}
		}
		if len(out) > 0 {
			return out
		}
		return whole
	}
	var out []cpShape
	for _, file := range files {
		if shape, ok := opaqueImageShape(root, file, ch.FromArea, x, y); ok && len(shape.cells) > 0 {
			out = append(out, shape)
		}
	}
	if len(out) > 0 {
		return out
	}
	return whole
}

func opaqueImageShape(root, rel string, fromRaw json.RawMessage, x, y int) (cpShape, bool) {
	fileKey, ok := pngFileKey(root, rel)
	if !ok {
		return cpShape{}, false
	}
	fromKey := "full"
	var from cpShape
	if len(fromRaw) > 0 {
		var parsed bool
		from, parsed = areaOf(fromRaw)
		if !parsed {
			return cpShape{}, false
		}
		fromKey = strconv.Itoa(from.x) + "," + strconv.Itoa(from.y) + "," +
			strconv.Itoa(from.w) + "," + strconv.Itoa(from.h)
	}
	key := fileKey + "\x00" + fromKey + "\x00" + strconv.Itoa(x) + "," + strconv.Itoa(y)
	if cached, ok := pngShapeCache.Load(key); ok {
		if cells, ok := cached.(string); ok {
			return cpShape{kind: 'r', cells: cells}, true
		}
	}
	decoded, ok := loadPNGAlpha(root, rel, fileKey)
	if !ok {
		return cpShape{}, false
	}
	if fromKey == "full" {
		from = cpShape{w: decoded.w, h: decoded.h}
	}
	minX, maxX := max(from.x, 0), min(from.x+from.w, decoded.w)
	minY, maxY := max(from.y, 0), min(from.y+from.h, decoded.h)
	if minX >= maxX || minY >= maxY {
		return cpShape{kind: 'r'}, true
	}
	firstCellX := (x + minX - from.x) / 16
	lastCellX := (x + maxX - 1 - from.x) / 16
	firstCellY := (y + minY - from.y) / 16
	lastCellY := (y + maxY - 1 - from.y) / 16
	width := lastCellX - firstCellX + 1
	bitmap := make([]bool, width*(lastCellY-firstCellY+1))
	for py := minY; py < maxY; py++ {
		row := py * decoded.w
		for px := minX; px < maxX; px++ {
			if !maskSet(decoded.a, row+px) {
				continue
			}
			cellX := (x + px - from.x) / 16
			cellY := (y + py - from.y) / 16
			bitmap[(cellY-firstCellY)*width+cellX-firstCellX] = true
		}
	}
	var cells strings.Builder
	for cellY := firstCellY; cellY <= lastCellY; cellY++ {
		for cellX := firstCellX; cellX <= lastCellX; cellX++ {
			if bitmap[(cellY-firstCellY)*width+cellX-firstCellX] {
				cells.WriteByte(';')
				cells.WriteString(strconv.Itoa(cellX))
				cells.WriteByte(',')
				cells.WriteString(strconv.Itoa(cellY))
			}
		}
	}
	pngShapeCache.Store(key, cells.String())
	return cpShape{kind: 'r', cells: cells.String()}, true
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
func warmOverlayImages(root, rel string, changes []json.RawMessage) {
	if SkipImageOverlap {
		return
	}
	var wg sync.WaitGroup
	for _, raw := range changes {
		var ch cpChange
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

func loadPNGAlpha(root, rel, fileKey string) (pngAlpha, bool) {
	if cached, ok := pngAlphaCache.Load(fileKey); ok {
		if pix, ok := cached.(pngAlpha); ok {
			return pix, pix.a != nil
		}
	}
	file, err := os.OpenInRoot(root, filepath.FromSlash(rel))
	if err != nil {
		pngAlphaCache.Store(fileKey, pngAlpha{})
		return pngAlpha{}, false
	}
	pix, ok := decodePNGAlpha(file)
	_ = file.Close()
	if !ok {
		pngAlphaCache.Store(fileKey, pngAlpha{})
		return pngAlpha{}, false
	}
	pix.a = packNonzeroMask(pix.a)
	actual, loaded := pngAlphaCache.LoadOrStore(fileKey, pix)
	if loaded {
		releaseAlpha(pix.a)
		if kept, ok := actual.(pngAlpha); ok {
			return kept, kept.a != nil
		}
		return pngAlpha{}, false
	}
	return pix, true
}

func packNonzeroMask(a []byte) []byte {
	n := (len(a) + 7) / 8
	mask := acquireAlpha(n)
	clear(mask)
	for i, v := range a {
		if v != 0 {
			mask[i>>3] |= 1 << (i & 7)
		}
	}
	releaseAlpha(a)
	return mask
}

func maskSet(mask []byte, i int) bool {
	if i < 0 || i>>3 >= len(mask) {
		return false
	}
	return mask[i>>3]&(1<<(uint(i)&7)) != 0
}

func decodePNGAlpha(r io.Reader) (pngAlpha, bool) {
	raw, err := io.ReadAll(r)
	if err != nil || len(raw) < 8 || string(raw[:8]) != "\x89PNG\r\n\x1a\n" {
		return pngAlpha{}, false
	}
	var (
		w, h, bitDepth, colorType, interlace int
		trns, idat                           []byte
		gotIHDR                              bool
	)
	for i := 8; i+12 <= len(raw); {
		n := int(binary.BigEndian.Uint32(raw[i:]))
		i += 4
		if i+4+n+4 > len(raw) {
			return pngAlpha{}, false
		}
		kind := string(raw[i : i+4])
		data := raw[i+4 : i+4+n]
		i += 4 + n + 4
		switch kind {
		case "IHDR":
			if n < 13 {
				return pngAlpha{}, false
			}
			w = int(binary.BigEndian.Uint32(data[0:4]))
			h = int(binary.BigEndian.Uint32(data[4:8]))
			bitDepth = int(data[8])
			colorType = int(data[9])
			interlace = int(data[12])
			gotIHDR = true
		case "tRNS":
			trns = data
		case "IDAT":
			idat = append(idat, data...)
		case "IEND":
			i = len(raw)
		}
	}
	if !gotIHDR || w <= 0 || h <= 0 || interlace != 0 || bitDepth != 8 {
		return decodePNGAlphaStd(bytes.NewReader(raw))
	}
	n := w * h
	alpha := acquireAlpha(n)
	opaque := colorType == 0 || colorType == 2
	if opaque && len(trns) == 0 {
		for i := range alpha {
			alpha[i] = 255
		}
		return pngAlpha{w: w, h: h, a: alpha}, true
	}
	if !inflatePNGAlpha(idat, alpha, w, h, colorType, trns) {
		releaseAlpha(alpha)
		return decodePNGAlphaStd(bytes.NewReader(raw))
	}
	return pngAlpha{w: w, h: h, a: alpha}, true
}

func inflatePNGAlpha(idat, alpha []byte, w, h, colorType int, trns []byte) bool {
	cpp := pngChannels(colorType)
	if cpp == 0 {
		return false
	}
	zr, err := zlib.NewReader(bytes.NewReader(idat))
	if err != nil {
		return false
	}
	defer func() { _ = zr.Close() }()
	stride := w*cpp + 1
	row := make([]byte, stride)
	prev := make([]byte, stride)
	i := 0
	for range h {
		if _, err := io.ReadFull(zr, row); err != nil {
			return false
		}
		if !pngUnfilter(row, prev, cpp) {
			return false
		}
		pix := row[1:]
		switch colorType {
		case 0:
			for x := range w {
				a := byte(255)
				if len(trns) >= 2 && pix[x] == trns[1] {
					a = 0
				}
				alpha[i] = a
				i++
			}
		case 2:
			for x := range w {
				off := x * 3
				a := byte(255)
				if len(trns) >= 6 && pix[off] == trns[1] && pix[off+1] == trns[3] && pix[off+2] == trns[5] {
					a = 0
				}
				alpha[i] = a
				i++
			}
		case 3:
			for x := range w {
				idx := int(pix[x])
				a := byte(255)
				if idx < len(trns) {
					a = trns[idx]
				}
				alpha[i] = a
				i++
			}
		case 4:
			for x := range w {
				alpha[i] = pix[x*2+1]
				i++
			}
		case 6:
			for x := range w {
				alpha[i] = pix[x*4+3]
				i++
			}
		default:
			return false
		}
		copy(prev, row)
	}
	return i == len(alpha)
}

func pngChannels(colorType int) int {
	switch colorType {
	case 0, 3:
		return 1
	case 2:
		return 3
	case 4:
		return 2
	case 6:
		return 4
	default:
		return 0
	}
}

func pngUnfilter(row, prev []byte, cpp int) bool {
	filter := row[0]
	cur := row[1:]
	prior := prev[1:]
	switch filter {
	case 0:
		return true
	case 1:
		for i := cpp; i < len(cur); i++ {
			cur[i] += cur[i-cpp]
		}
		return true
	case 2:
		for i := range cur {
			cur[i] += prior[i]
		}
		return true
	case 3:
		for i := range cur {
			var a byte
			if i >= cpp {
				a = cur[i-cpp]
			}
			cur[i] += a>>1 + prior[i]>>1 + a&prior[i]&1
		}
		return true
	case 4:
		for i := range cur {
			var a, c byte
			if i >= cpp {
				a = cur[i-cpp]
				c = prior[i-cpp]
			}
			cur[i] += paeth(a, prior[i], c)
		}
		return true
	default:
		return false
	}
}

func paeth(a, b, c byte) byte {
	ia, ib, ic := int(a), int(b), int(c)
	p := ia + ib - ic
	pa, pb, pc := absInt(p-ia), absInt(p-ib), absInt(p-ic)
	if pa <= pb && pa <= pc {
		return a
	}
	if pb <= pc {
		return b
	}
	return c
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func decodePNGAlphaStd(r io.Reader) (pngAlpha, bool) {
	decoded, err := png.Decode(r)
	if err != nil {
		return pngAlpha{}, false
	}
	b := decoded.Bounds()
	n := b.Dx() * b.Dy()
	alpha := acquireAlpha(n)
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			alpha[i] = imageAlpha(decoded, x, y)
			i++
		}
	}
	return pngAlpha{w: b.Dx(), h: b.Dy(), a: alpha}, true
}

func imageAlpha(img image.Image, x, y int) uint8 {
	switch img := img.(type) {
	case *image.NRGBA:
		return img.Pix[img.PixOffset(x, y)+3]
	case *image.RGBA:
		return img.Pix[img.PixOffset(x, y)+3]
	default:
		_, _, _, a := img.At(x, y).RGBA()
		if a > 0 {
			return 1
		}
		return 0
	}
}

func sourceFiles(root, rel string) []string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return nil
	}
	if !hasToken(rel) {
		if path, ok := caseInsensitivePath(root, rel); ok {
			relative, err := filepath.Rel(root, path)
			if err == nil {
				return []string{filepath.ToSlash(relative)}
			}
		}
		return nil
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	var out []string
	var walk func(string, int, string)
	walk = func(dir string, index int, prefix string) {
		if index == len(parts) {
			out = append(out, strings.TrimPrefix(filepath.ToSlash(prefix), "./"))
			return
		}
		part := parts[index]
		if hasToken(part) {
			for {
				start := strings.Index(part, "{{")
				if start < 0 {
					break
				}
				end := strings.Index(part[start+2:], "}}")
				if end < 0 {
					break
				}
				end += start + 2
				part = part[:start] + "*" + part[end+2:]
			}
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if !matchFold(part, entry.Name()) {
				continue
			}
			next := filepath.Join(dir, entry.Name())
			if index == len(parts)-1 {
				if !entry.IsDir() {
					walk(next, index+1, filepath.Join(prefix, entry.Name()))
				}
				continue
			}
			if entry.IsDir() {
				walk(next, index+1, filepath.Join(prefix, entry.Name()))
			}
		}
	}
	walk(root, 0, "")
	return out
}

func matchFold(pattern, value string) bool {
	pattern, value = strings.ToLower(pattern), strings.ToLower(value)
	matched, err := filepath.Match(pattern, value)
	return err == nil && matched
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

type decodedMapResult struct {
	value decodedMap
	ok    bool
}

var mapCache sync.Map

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
	info, err := os.Stat(abs)
	if err != nil {
		return decodedMap{}, false
	}
	key := abs + "\x00" + strconv.FormatInt(info.Size(), 10) + "\x00" +
		strconv.FormatInt(info.ModTime().UnixNano(), 10)
	if cached, ok := mapCache.Load(key); ok {
		result, ok := cached.(decodedMapResult)
		if ok {
			return result.value, result.ok
		}
	}
	raw, err := fsx.ReadFile(abs)
	if err != nil {
		return decodedMap{}, false
	}
	if strings.EqualFold(filepath.Ext(rel), ".tmj") {
		var doc tmjMapDocument
		if json.Unmarshal(raw, &doc) != nil {
			mapCache.Store(key, decodedMapResult{ok: false})
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
		mapCache.Store(key, decodedMapResult{value: out, ok: true})
		return out, true
	}

	var doc tmxMapDocument
	if xml.Unmarshal(raw, &doc) != nil {
		mapCache.Store(key, decodedMapResult{ok: false})
		return decodedMap{}, false
	}
	out := decodedMap{width: doc.Width, height: doc.Height}
	for _, layer := range doc.Layers {
		decoded, ok := decodeTMXLayer(layer)
		if !ok {
			mapCache.Store(key, decodedMapResult{ok: false})
			return decodedMap{}, false
		}
		decoded.name = strings.ToLower(strings.TrimSpace(layer.Name))
		out.layers = append(out.layers, decoded)
	}
	mapBounds(&out)
	mapCache.Store(key, decodedMapResult{value: out, ok: true})
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
		for value := range strings.SplitSeq(text, ",") {
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
	if len(m.layers) == 0 {
		to.layer = ""
		return []cpShape{to}
	}
	mode := strings.ToLower(strings.TrimSpace(ch.PatchMode))
	mode = cmp.Or(mode, "replacebylayer")
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
func spouseOf(raw map[string]json.RawMessage, tokens []cpTokenDefinition) string {
	for k, v := range raw {
		key := strings.TrimSpace(k)
		if inner, ok := strings.CutPrefix(key, "{{"); ok {
			if key, ok = strings.CutSuffix(inner, "}}"); !ok {
				continue
			}
			key = strings.TrimSpace(key)
		}
		if strings.EqualFold(key, "spouse") {
			var npc string
			if json.Unmarshal(v, &npc) == nil && !hasToken(npc) && strings.TrimSpace(npc) != "" {
				return strings.ToLower(strings.TrimSpace(npc))
			}
			continue
		}
		if name, expr, ok := strings.Cut(key, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "query") {
			var on bool
			if json.Unmarshal(v, &on) != nil || !on {
				continue
			}
			if npc := querySpouse(expr, tokens); npc != "" {
				return npc
			}
			continue
		}
		var flags []string
		if len(tokens) > 0 && !strings.ContainsAny(key, ":|") && condValues(v, &flags) && len(flags) == 1 && strings.EqualFold(flags[0], "true") {
			if npc := tokenSpouse(tokenName(key), tokens); npc != "" {
				return npc
			}
		}
		name, npc, ok := strings.Cut(key, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "relationship") || hasToken(npc) {
			continue
		}
		var states []string
		if condValues(v, &states) && !slices.ContainsFunc(states, func(s string) bool {
			return !strings.EqualFold(s, "married") && !strings.EqualFold(s, "engaged")
		}) {
			return strings.ToLower(strings.TrimSpace(npc))
		}
	}
	return ""
}

// querySpouse is the partner an AND-only Query expression compares {{Spouse}}, or a token that is just
// querySpouse is the partner a Query expression requires: every OR branch compares {{Spouse}} or {{Roommate}}
// (or a token that is just {{Spouse}}) with the same NPC in one of its AND terms. A player lives with one partner
// at a time, so a branch per partner token still names one partner.
func querySpouse(expr string, tokens []cpTokenDefinition) string {
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

func branchSpouse(branch string, tokens []cpTokenDefinition) string {
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
func tokenSpouse(name string, tokens []cpTokenDefinition) string {
	partner := ""
	for _, definition := range tokens {
		if definition.name != name {
			continue
		}
		value := strings.TrimSpace(definition.value)
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
func withTokenFarmTypes(p cpPatch, tokens []cpTokenDefinition, present map[string]bool, schema map[string]cpSchema, config map[string]string) cpPatch {
	if len(p.when.dynamic) == 0 || len(p.places["farmtype"]) > 0 {
		return p
	}
	reachable := cachedReachableTokens(tokens, present, schema, config, p.when.flags)
	for _, condition := range p.when.dynamic {
		var types []string
		pinned := true
		for _, definition := range tokens {
			if definition.name != condition.name {
				continue
			}
			if hasToken(definition.value) {
				pinned = false
				break
			}
			if !dynamicConditionMatchesValue(condition, definition.value) ||
				dynamicWhenState(definition.when, tokens, reachable, present, schema, config, p.when.flags) == cpConditionFalse {
				continue
			}
			farms := placesOf(definition.when)["farmtype"]
			if len(farms) == 0 {
				pinned = false
				break
			}
			types = append(types, farms...)
		}
		if pinned && len(types) > 0 {
			places := maps.Clone(p.places)
			if places == nil {
				places = map[string][]string{}
			}
			places["farmtype"] = types
			p.places = places
			return p
		}
	}
	return p
}

func spouseAlias(name string, tokens []cpTokenDefinition) bool {
	found := false
	for _, definition := range tokens {
		if definition.name != name {
			continue
		}
		if !strings.EqualFold(strings.Join(strings.Fields(definition.value), ""), "{{spouse}}") {
			return false
		}
		found = true
	}
	return found
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
	// One gated on a farm type and one on a farm map only another farm type loads.
	_, aFarm := a.places["farmtype"]
	_, bFarm := b.places["farmtype"]
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
func farmTypesOf(p cpPatch, custom string) []string {
	if types := p.places["farmtype"]; len(types) > 0 {
		return types
	}
	if t, ok := farmMaps[p.target]; ok {
		return []string{t}
	}
	if custom != "" {
		return []string{custom}
	}
	return nil
}

// editsClash reports whether any active edit of one pack can overwrite one of the other's, and whether every
// such overlap is harmless (see harmless).
func editsClash(a, b []cpPatch) (clash, minor bool) {
	clash, minor, _ = editsClashIndexed(a, b, indexesOf(b), false)
	return clash, minor
}

// indexesOf is the shape index of each patch, looked up once so a pair loop does not repeat it per comparison.
func indexesOf(patches []cpPatch) []*shapeSet {
	out := make([]*shapeSet, len(patches))
	for i, p := range patches {
		out[i] = shapeIndex(p.shapes)
	}
	return out
}

// editsClashIndexed is editsClash with bIndex[i] the shape index of b[i], and the note that explains why a
// minor clash is harmless.
// One author ordering two of their packs' edits with an explicit priority meant the result.
func editsClashIndexed(a, b []cpPatch, bIndex []*shapeSet, sameAuthor bool) (clash, minor bool, why *framework.ConflictNote) {
	minor = true
	var note noteAgreement
	for _, x := range a {
		for j, y := range b {
			if mapOverlayHasUnknownLayer(x, y) {
				continue
			}
			if x.image && y.image && x.imageDigest != "" && x.imageDigest == y.imageDigest && x.imageFromArea == y.imageFromArea {
				continue
			}
			if x.image && y.image && strings.EqualFold(strings.TrimSpace(x.patchMode), "overlay") && strings.EqualFold(strings.TrimSpace(y.patchMode), "overlay") {
				continue
			}
			if exclusive(x, y) || sameAuthor && (explicitOrder(x) || explicitOrder(y)) {
				continue
			}
			if bIndex[j].overlaps(x.shapes) {
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

// explicitOrder is an edit given a Late, Early or Low priority, set to run before or after the others.
func explicitOrder(p cpPatch) bool {
	base := strings.ToLower(strings.TrimSpace(p.priority))
	if i := strings.IndexAny(base, "+-"); i > 0 {
		base = strings.TrimSpace(base[:i])
	}
	return base == "late" || base == "early" || base == "low"
}

// noteAgreement keeps a ConflictNote only while every harmless overlap gives the same one.
type noteAgreement struct {
	note  *framework.ConflictNote
	seen  bool
	mixed bool
}

func (a *noteAgreement) add(n *framework.ConflictNote) {
	switch {
	case !a.seen:
		a.note, a.seen = n, true
	case n == nil || a.note == nil || *n != *a.note:
		a.mixed = true
	}
}

func (a *noteAgreement) result() *framework.ConflictNote {
	if a.mixed {
		return nil
	}
	return a.note
}

func mapOverlayHasUnknownLayer(a, b cpPatch) bool {
	if a.image || b.image {
		return false
	}
	aOverlay := strings.EqualFold(strings.TrimSpace(a.patchMode), "overlay")
	bOverlay := strings.EqualFold(strings.TrimSpace(b.patchMode), "overlay")
	return (aOverlay && hasUnknownMapLayer(b)) || (bOverlay && hasUnknownMapLayer(a))
}

func hasUnknownMapLayer(p cpPatch) bool {
	return slices.ContainsFunc(p.shapes, func(shape cpShape) bool {
		return shape.kind != 'p' && shape.layer == ""
	})
}

// harmless reports an overlap that cannot hurt play: two image edits only change how something looks; an edit
// that applies in one location or weather, or on one day or festival, only matters there; two that only set
// prices let one mod's numbers win; and a one-tile edit at a computed spot is too small to place, so it is
// shown without counting as a problem.
func harmless(x, y cpPatch, yIndex *shapeSet) (bool, *framework.ConflictNote) {
	if balanceOverlap(x, y, yIndex) {
		return true, &framework.ConflictNote{Kind: "balance"}
	}
	for _, p := range []cpPatch{x, y} {
		if day := oneDay(p); day != "" {
			return true, &framework.ConflictNote{Kind: "day", Value: day}
		}
	}
	return (x.image && y.image) || (textOnly(x) && textOnly(y)) || overlayPriorityHarmless(x, y) || situational(x) || situational(y) || tinyOnly(x) || tinyOnly(y), nil
}

// balanceFields are data fields that only tune prices and costs: when two collide, one mod's numbers win.
var balanceFields = map[string]bool{"buildcost": true, "buildmaterials": true, "purchaseprice": true, "sellprice": true, "price": true, "pricemodifiers": true}

// balanceOverlap is two data edits that only collide on balance fields (yIndex indexes y's shapes); what
// else either sets, the other leaves alone.
func balanceOverlap(x, y cpPatch, yIndex *shapeSet) bool {
	if x.action != kindEditData || y.action != kindEditData {
		return false
	}
	return !slices.ContainsFunc(x.shapes, func(s cpShape) bool {
		return !balanceKey(s.key) && slices.ContainsFunc(yIndex.props[s.key], s.overlaps)
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
func textOnly(p cpPatch) bool {
	if p.action != kindEditData {
		return false
	}
	t := strings.ToLower(p.target)
	switch {
	case strings.HasPrefix(t, "characters/dialogue/"), strings.HasPrefix(t, "strings/"), t == "data/extradialogue":
		return true
	case strings.HasPrefix(t, "data/festivals/"):
		return !slices.ContainsFunc(p.shapes, func(s cpShape) bool {
			k := strings.ToLower(s.key)
			return strings.Contains(k, "set-up") || strings.Contains(k, "mainevent") || strings.Contains(k, "shop") || strings.Contains(k, "conditions")
		})
	}
	return false
}

func overlayPriorityHarmless(x, y cpPatch) bool {
	xOverlay := strings.EqualFold(strings.TrimSpace(x.patchMode), "overlay")
	yOverlay := strings.EqualFold(strings.TrimSpace(y.patchMode), "overlay")
	if xOverlay == yOverlay {
		return false
	}
	if xOverlay {
		return contentPatcherPriority("edit", x.priority) > contentPatcherPriority("edit", y.priority)
	}
	return contentPatcherPriority("edit", y.priority) > contentPatcherPriority("edit", x.priority)
}

func situational(p cpPatch) bool {
	_, location := p.places["locationname"]
	_, context := p.places["locationcontext"]
	_, weather := p.places["weather"]
	return location || context || weather
}

// oneDay names the single day ("Summer 28") or festival a patch is limited to, or "".
func oneDay(p cpPatch) string {
	if events := p.places["dayevent"]; len(events) == 1 {
		return titleWords(events[0])
	}
	season, day := p.places["season"], p.places["day"]
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

func tinyOnly(p cpPatch) bool {
	return len(p.shapes) > 0 && !slices.ContainsFunc(p.shapes, func(s cpShape) bool { return !s.tiny })
}

// markClashes records which edits of a and b overlap each other.
func markClashes(a, b *packHit) {
	ordered := sameAuthor(*a, *b)
	for i, x := range a.edits {
		for j, y := range b.edits {
			if exclusive(x, y) || ordered && (explicitOrder(x) || explicitOrder(y)) || !shapesOverlap(x.shapes, y.shapes) {
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
// from the conflict, such as Better Things' DesertMinecart for an expansion that redraws the desert. Of the
// fields that would, it offers the narrowest: one named for what the target is about, then the one gating
// the fewest of the pack's patches. When only the pack's master switch would, it offers nothing.
func switchOff(h packHit, peerSets ...[]packHit) (framework.ConflictFix, bool) {
	if len(h.clashes) == 0 {
		return framework.ConflictFix{}, false
	}
	clashing := slices.Sorted(maps.Keys(h.clashes))
	subject := h.edits[clashing[0]].target
	subject = strings.ToLower(subject[strings.LastIndex(subject, "/")+1:])
	var best framework.ConflictFix
	bestGated, bestNamed, found := 0, false, false
	for _, c := range h.edits[clashing[0]].when.config {
		field, ok := h.schema[strings.ToLower(c.field)]
		if !ok || !field.toggle() {
			continue
		}
		value, ok := offValue(h, clashing, field, peerSets...)
		if !ok {
			continue
		}
		named := strings.Contains(strings.ToLower(field.key), subject)
		gated, total, left := 0, 0, 0
		config := maps.Clone(h.config)
		config[strings.ToLower(field.key)] = value
		for _, p := range h.tokens {
			if p.kind == "other" {
				continue
			}
			total++
			if slices.ContainsFunc(p.when.config, func(o cpConfig) bool { return strings.EqualFold(o.field, field.key) }) {
				gated++
			}
			if configHolds(p.when.config, h.schema, config) {
				left++
			}
		}
		// A field that switches off the whole pack, or a fifth of it without being named for this target, is
		// the pack's master switch (Better Water's Color, a schedule pack's PlotSchedules): offering it
		// would trade the conflict for most of the mod.
		// ponytail: fixed one-fifth share; weigh by what the gated patches touch if it misjudges a pack.
		if left == 0 || !named && gated*5 >= total {
			continue
		}
		if found && (bestNamed && !named || bestNamed == named && (gated > bestGated || gated == bestGated && field.key >= best.Field)) {
			continue
		}
		current, set := h.config[strings.ToLower(field.key)]
		if !set {
			current = field.defaultValue
		}
		best = framework.ConflictFix{Key: h.key, ID: h.id, Name: h.name, Field: field.key, Current: current, Value: value}
		bestGated, bestNamed, found = gated, named, true
	}
	return best, found
}

// offValue is the one value of field that switches off every clashing edit of h without leaving another of
// its edits clashing with peerSets[0].
func offValue(h packHit, clashing []int, field cpSchema, peerSets ...[]packHit) (string, bool) {
	needed := map[string]bool{}
	for _, i := range clashing {
		j := slices.IndexFunc(h.edits[i].when.config, func(o cpConfig) bool { return strings.EqualFold(o.field, field.key) })
		if j < 0 {
			return "", false
		}
		for _, v := range h.edits[i].when.config[j].values {
			needed[strings.ToLower(v)] = true
		}
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

func settingStillClashes(h packHit, peers []packHit, field cpSchema, value string) bool {
	config := make(map[string]string, len(h.config))
	maps.Copy(config, h.config)
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
		if mod.Equal(peer.id, h.id) {
			continue
		}
		if clash, _ := editsClash(active, peer.edits); clash {
			return true
		}
	}
	return false
}
