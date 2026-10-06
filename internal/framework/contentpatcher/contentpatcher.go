package contentpatcher

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/jsonc"
)

var contentPatcherID = mod.SMAPI("Pathoschild.ContentPatcher")

const (
	kindLoad      = "Load"
	kindEditImage = "EditImage"
	kindEditMap   = "EditMap"
	kindEditData  = "EditData"
)

type packHit struct {
	id           mod.ID
	name         string
	key          string
	priority     string
	mentions     map[string]bool
	root         string
	tokens       []cpPatch
	present      map[string]bool
	loads        []cpPatch
	edits        []cpPatch // the pack's active edits of this target
	eligible     []cpPatch // edits whose HasMod conditions hold, including config-off variants
	loadClashes  map[int]bool
	dependencies map[string]bool
	loadAfter    map[string]bool
	schema       map[string]cpSchema
	config       map[string]string
	clashes      map[int]bool // indices into edits that overlap an edit of a pack it was not built with
	// author is the manifest Author, normalised, so two packs by one author can be told apart from strangers.
	author string
	// sig is the pack's share of a target's memo key (packSig), and stable whether its stamps can be trusted.
	sig    string
	stable bool
}

// cpPatch is one Load or EditImage/EditMap change with the HasMod conditions that gate it.
type cpPatch struct {
	kind          string // "load" or "edit"
	target        string
	fromFile      string
	priority      string
	patchMode     string
	when          cpWhen
	shapes        []cpShape           // what an edit writes; see editShapes
	spouse        string              // the spouse the change requires, or ""
	places        map[string][]string // literal values the change requires of placeTokens
	image         bool                // an EditImage change, which only changes how something looks
	imageDigest   string
	imageFromArea string
	source        string
	index         int
	action        string
	toArea        string
	// tokenValue is a dynamic token's value; a token that yields a picker value only when a mod is
	// installed is how a pack says which mod that value is for.
	tokenName  string
	tokenValue string
	// extra marks a change that also does something its shapes do not show (warps, text operations,
	// appended list entries, a source map's properties), which a later edit never undoes.
	extra bool
}

type cpConfig struct {
	field         string
	values        []string
	allowMultiple bool
}

// cpWhen holds a change's HasMod conditions: each anyOf group needs one of its mods
// present, and no noneOf mod may be present. Dynamic token conditions are checked
// with the pack and profile context; other conditions are treated as met.
type cpWhen struct {
	anyOf   [][]string
	noneOf  []string
	config  []cpConfig
	dynamic []cpDynamicCondition
	flags   []cpFlagCondition
	spouse  string
	places  map[string][]string
	// assumed are the raw keys of the conditions this scan treats as met (time, queries, events), shown
	// with the conflict so the user knows it may not happen on their save.
	assumed []string
	// conditional is set when the change, or an Include around it, has any When condition at all, including
	// the assumed ones.
	conditional bool
}

type cpDynamicCondition struct {
	name     string
	values   []string
	contains string
	expected bool
}

type cpFlagCondition struct {
	name    string
	present bool
}

func (w cpWhen) with(o cpWhen) cpWhen {
	places := map[string][]string{}
	for key, values := range w.places {
		places[key] = slices.Clone(values)
	}
	for key, values := range o.places {
		places[key] = append(places[key], values...)
	}
	spouse := w.spouse
	if spouse == "" {
		spouse = o.spouse
	} else if o.spouse != "" && !strings.EqualFold(spouse, o.spouse) {
		spouse = "\x00"
	}
	return cpWhen{
		anyOf:       append(slices.Clone(w.anyOf), o.anyOf...),
		noneOf:      append(slices.Clone(w.noneOf), o.noneOf...),
		config:      append(slices.Clone(w.config), o.config...),
		dynamic:     append(slices.Clone(w.dynamic), o.dynamic...),
		flags:       append(slices.Clone(w.flags), o.flags...),
		spouse:      spouse,
		places:      places,
		assumed:     append(slices.Clone(w.assumed), o.assumed...),
		conditional: w.conditional || o.conditional,
	}
}

func (w cpWhen) withDefinition(o cpWhen) cpWhen {
	merged := w.with(o)
	// A definition's flags constrain its reachability, not the patch that selects it.
	merged.flags = slices.Clone(w.flags)
	return merged
}

func (w cpWhen) holds(present map[string]bool) bool {
	for _, group := range w.anyOf {
		if !slices.ContainsFunc(group, func(id string) bool { return present[id] }) {
			return false
		}
	}
	return !slices.ContainsFunc(w.noneOf, func(id string) bool { return present[id] })
}

type cachedPack struct {
	// values are the pack's config values (schema defaults, then config.json), by lower-case key;
	// used while scanning only, so not cached on disk.
	values      map[string]string
	fingerprint string
	files       []packFileStamp
	patches     []cpPatch
	tokens      []cpTokenDefinition
	mentions    map[string]bool
	schema      map[string]cpSchema
	skips       int
	// farms maps the map asset of each custom farm the pack adds to Data/AdditionalFarms to the farm's id.
	farms map[string]string
}

const contentPackParserVersion = 21

// absentSize stamps a file that was not there, so the cache is dropped when it appears.
const absentSize = -1

type packFileStamp struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime"`
}

type diskPackCache struct {
	Version int                      `json:"version"`
	Packs   map[string]diskPackEntry `json:"packs"`
}

type diskPackEntry struct {
	Fingerprint string          `json:"fingerprint"`
	Files       []packFileStamp `json:"files"`
	// Pack is the pack's diskCachedPack, encoded when the pack is parsed, so a save after one pack changed encodes
	// only that pack, and the packs not read yet stay compact in memory.
	Pack []byte `json:"pack"`
}

type diskCachedPack struct {
	Patches  []diskPatch           `json:"patches,omitempty"`
	Tokens   []diskTokenDefinition `json:"tokens,omitempty"`
	Mentions map[string]bool       `json:"mentions,omitempty"`
	Schema   map[string]diskSchema `json:"schema,omitempty"`
	Skips    int                   `json:"skips,omitempty"`
	Farms    map[string]string     `json:"farms,omitempty"`
}

type diskPatch struct {
	Kind          string      `json:"kind,omitempty"`
	Target        string      `json:"target,omitempty"`
	FromFile      string      `json:"fromFile,omitempty"`
	Priority      string      `json:"priority,omitempty"`
	PatchMode     string      `json:"patchMode,omitempty"`
	When          diskWhen    `json:"when,omitzero"`
	Shapes        []diskShape `json:"shapes,omitempty"`
	Image         bool        `json:"image,omitempty"`
	ImageDigest   string      `json:"imageDigest,omitempty"`
	ImageFromArea string      `json:"imageFromArea,omitempty"`
	Source        string      `json:"source,omitempty"`
	Index         int         `json:"index,omitempty"`
	Action        string      `json:"action,omitempty"`
	ToArea        string      `json:"toArea,omitempty"`
	TokenName     string      `json:"tokenName,omitempty"`
	TokenValue    string      `json:"tokenValue,omitempty"`
	Extra         bool        `json:"extra,omitempty"`
}

type diskShape struct {
	Kind  byte   `json:"kind,omitempty"`
	X     int    `json:"x,omitempty"`
	Y     int    `json:"y,omitempty"`
	W     int    `json:"w,omitempty"`
	H     int    `json:"h,omitempty"`
	Cells []byte `json:"cells,omitempty"`
	Layer string `json:"layer,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	Tiny  bool   `json:"tiny,omitempty"`
}

type diskWhen struct {
	AnyOf   [][]string             `json:"anyOf,omitempty"`
	NoneOf  []string               `json:"noneOf,omitempty"`
	Config  []diskConfig           `json:"config,omitempty"`
	Dynamic []diskDynamicCondition `json:"dynamic,omitempty"`
	Flags   []diskFlagCondition    `json:"flags,omitempty"`
	Spouse  string                 `json:"spouse,omitempty"`
	Places  map[string][]string    `json:"places,omitempty"`
	Assumed []string               `json:"assumed,omitempty"`
	Cond    bool                   `json:"cond,omitempty"`
}

type diskConfig struct {
	Field         string   `json:"field,omitempty"`
	Values        []string `json:"values,omitempty"`
	AllowMultiple bool     `json:"allowMultiple,omitempty"`
}

type diskDynamicCondition struct {
	Name     string   `json:"name,omitempty"`
	Values   []string `json:"values,omitempty"`
	Contains string   `json:"contains,omitempty"`
	Expected bool     `json:"expected,omitempty"`
}

type diskFlagCondition struct {
	Name    string `json:"name,omitempty"`
	Present bool   `json:"present,omitempty"`
}

type diskTokenDefinition struct {
	Name  string                     `json:"name,omitempty"`
	Value string                     `json:"value,omitempty"`
	When  map[string]json.RawMessage `json:"when,omitempty"`
}

type diskSchema struct {
	Key           string   `json:"key,omitempty"`
	DefaultValue  string   `json:"defaultValue,omitempty"`
	AllowMultiple bool     `json:"allowMultiple,omitempty"`
	AllowValues   []string `json:"allowValues,omitempty"`
	AllowBlank    bool     `json:"allowBlank,omitempty"`
	Description   string   `json:"description,omitempty"`
}

var packDiskState struct {
	sync.Mutex
	loaded  bool
	path    string
	entries map[string]diskPackEntry
	dirty   bool
}

type cpTokenDefinition struct {
	name  string
	value string
	when  map[string]json.RawMessage
}

func diskPackOf(pack cachedPack) diskCachedPack {
	out := diskCachedPack{
		Mentions: pack.mentions,
		Skips:    pack.skips,
		Farms:    pack.farms,
	}
	if pack.patches != nil {
		out.Patches = make([]diskPatch, len(pack.patches))
		for i, patch := range pack.patches {
			out.Patches[i] = diskPatchOf(patch)
		}
	}
	if pack.tokens != nil {
		out.Tokens = make([]diskTokenDefinition, len(pack.tokens))
		for i, token := range pack.tokens {
			out.Tokens[i] = diskTokenDefinition{Name: token.name, Value: token.value, When: token.when}
		}
	}
	if pack.schema != nil {
		out.Schema = make(map[string]diskSchema, len(pack.schema))
		for key, schema := range pack.schema {
			out.Schema[key] = diskSchema{
				Key:           schema.key,
				DefaultValue:  schema.defaultValue,
				AllowMultiple: schema.allowMultiple,
				AllowValues:   schema.allowValues,
				AllowBlank:    schema.allowBlank,
				Description:   schema.description,
			}
		}
	}
	return out
}

func cachedPackOfDisk(root string, disk diskCachedPack, entry diskPackEntry) cachedPack {
	out := cachedPack{
		fingerprint: entry.Fingerprint,
		files:       entry.Files,
		mentions:    disk.Mentions,
		schema:      make(map[string]cpSchema, len(disk.Schema)),
		skips:       disk.Skips,
		farms:       disk.Farms,
	}
	if disk.Patches != nil {
		out.patches = make([]cpPatch, len(disk.Patches))
		for i, patch := range disk.Patches {
			out.patches[i] = cpPatchOfDisk(root, patch)
		}
	}
	if disk.Tokens != nil {
		out.tokens = make([]cpTokenDefinition, len(disk.Tokens))
		for i, token := range disk.Tokens {
			out.tokens[i] = cpTokenDefinition{name: token.Name, value: token.Value, when: token.When}
		}
	}
	for key, schema := range disk.Schema {
		out.schema[key] = cpSchema{
			key:           schema.Key,
			defaultValue:  schema.DefaultValue,
			allowMultiple: schema.AllowMultiple,
			allowValues:   schema.AllowValues,
			allowBlank:    schema.AllowBlank,
			description:   schema.Description,
		}
	}
	if disk.Schema == nil {
		out.schema = nil
	}
	return out
}

func diskPatchOf(patch cpPatch) diskPatch {
	out := diskPatch{
		Kind:          patch.kind,
		Target:        patch.target,
		FromFile:      patch.fromFile,
		Priority:      patch.priority,
		PatchMode:     patch.patchMode,
		When:          diskWhenOf(patch.when),
		Image:         patch.image,
		ImageDigest:   patch.imageDigest,
		ImageFromArea: patch.imageFromArea,
		Source:        patch.source,
		Index:         patch.index,
		Action:        patch.action,
		ToArea:        patch.toArea,
		TokenName:     patch.tokenName,
		TokenValue:    patch.tokenValue,
		Extra:         patch.extra,
	}
	if patch.shapes != nil {
		out.Shapes = make([]diskShape, len(patch.shapes))
		for i, shape := range patch.shapes {
			out.Shapes[i] = diskShape{
				Kind:  shape.kind,
				X:     shape.x,
				Y:     shape.y,
				W:     shape.w,
				H:     shape.h,
				Cells: encodeCellSet(shape.cells),
				Layer: shape.layer,
				Key:   shape.key,
				Value: shape.value,
				Tiny:  shape.tiny,
			}
		}
	}
	return out
}

func cpPatchOfDisk(_ string, patch diskPatch) cpPatch {
	when := cpWhenOfDisk(patch.When)
	out := cpPatch{
		kind:          patch.Kind,
		target:        patch.Target,
		fromFile:      patch.FromFile,
		priority:      patch.Priority,
		patchMode:     patch.PatchMode,
		when:          when,
		spouse:        when.spouse,
		places:        when.places,
		image:         patch.Image,
		imageDigest:   patch.ImageDigest,
		imageFromArea: patch.ImageFromArea,
		source:        patch.Source,
		index:         patch.Index,
		action:        patch.Action,
		toArea:        patch.ToArea,
		tokenName:     patch.TokenName,
		tokenValue:    patch.TokenValue,
		extra:         patch.Extra,
	}
	if patch.Shapes != nil {
		out.shapes = make([]cpShape, len(patch.Shapes))
		for i, shape := range patch.Shapes {
			out.shapes[i] = cpShape{
				kind:  shape.Kind,
				x:     shape.X,
				y:     shape.Y,
				w:     shape.W,
				h:     shape.H,
				cells: decodeCellSet(shape.Cells),
				layer: shape.Layer,
				key:   shape.Key,
				value: shape.Value,
				tiny:  shape.Tiny,
			}
		}
	}
	return out
}

// encodeCellSet packs "x,y;..." into a bounding-box bitmap, or ordered int16/int32 pairs when sparse.
func encodeCellSet(s string) []byte {
	if s == "" {
		return nil
	}
	var xs, ys []int
	minX, minY := 1<<30, 1<<30
	maxX, maxY := -1<<30, -1<<30
	for cell := range strings.SplitSeq(s, ";") {
		if cell == "" {
			continue
		}
		parts := strings.SplitN(cell, ",", 2)
		if len(parts) != 2 {
			continue
		}
		x, errX := strconv.Atoi(parts[0])
		y, errY := strconv.Atoi(parts[1])
		if errX != nil || errY != nil {
			continue
		}
		xs = append(xs, x)
		ys = append(ys, y)
		minX, minY = min(minX, x), min(minY, y)
		maxX, maxY = max(maxX, x), max(maxY, y)
	}
	n := len(xs)
	if n == 0 {
		return nil
	}
	width := maxX - minX + 1
	height := maxY - minY + 1
	bitBytes := (width*height + 7) / 8
	pairBytes := 1 + 4*n
	if width > 0 && height > 0 && width <= 65535 && height <= 65535 && bitBytes+10 < pairBytes && bitBytes <= 1<<20 &&
		minX >= -32768 && maxX <= 32767 && minY >= -32768 && maxY <= 32767 {
		out := make([]byte, 9+bitBytes)
		out[0] = 0
		putInt16(out[1:], minX)
		putInt16(out[3:], minY)
		putUint16(out[5:], width)
		putUint16(out[7:], height)
		for i := range n {
			bit := (ys[i]-minY)*width + (xs[i] - minX)
			out[9+bit/8] |= 1 << (bit % 8)
		}
		return out
	}
	fit16 := minX >= -32768 && maxX <= 32767 && minY >= -32768 && maxY <= 32767
	if fit16 {
		out := make([]byte, 1+4*n)
		out[0] = 1
		for i := range n {
			putInt16(out[1+4*i:], xs[i])
			putInt16(out[3+4*i:], ys[i])
		}
		return out
	}
	out := make([]byte, 1+8*n)
	out[0] = 2
	for i := range n {
		putInt32(out[1+8*i:], xs[i])
		putInt32(out[5+8*i:], ys[i])
	}
	return out
}

func putInt16(b []byte, v int) {
	binary.LittleEndian.PutUint16(b, uint16(uint(v)&0xffff))
}

func putInt32(b []byte, v int) {
	binary.LittleEndian.PutUint32(b, uint32(uint(v)&0xffffffff))
}

func putUint16(b []byte, v int) {
	binary.LittleEndian.PutUint16(b, uint16(uint(v)&0xffff))
}

func getUint16(b []byte) int {
	return int(b[0]) | int(b[1])<<8
}

func getInt16(b []byte) int {
	v := int(b[0]) | int(b[1])<<8
	if v >= 0x8000 {
		return v - 0x10000
	}
	return v
}

func getInt32(b []byte) int {
	v := int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
	if v >= 0x80000000 {
		return v - 0x100000000
	}
	return v
}

func decodeCellSet(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var b strings.Builder
	write := func(x, y int) {
		b.WriteByte(';')
		b.WriteString(strconv.Itoa(x))
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(y))
	}
	switch raw[0] {
	case 0:
		if len(raw) < 9 {
			return ""
		}
		minX := getInt16(raw[1:])
		minY := getInt16(raw[3:])
		width := getUint16(raw[5:])
		height := getUint16(raw[7:])
		bits := raw[9:]
		if width <= 0 || height <= 0 {
			return ""
		}
		for y := range height {
			for x := range width {
				bit := y*width + x
				if bit/8 >= len(bits) {
					return b.String()
				}
				if bits[bit/8]&(1<<(bit%8)) != 0 {
					write(minX+x, minY+y)
				}
			}
		}
	case 1:
		rest := raw[1:]
		if len(rest)%4 != 0 {
			return ""
		}
		for i := 0; i < len(rest); i += 4 {
			write(getInt16(rest[i:]), getInt16(rest[i+2:]))
		}
	case 2:
		rest := raw[1:]
		if len(rest)%8 != 0 {
			return ""
		}
		for i := 0; i < len(rest); i += 8 {
			write(getInt32(rest[i:]), getInt32(rest[i+4:]))
		}
	default:
		return ""
	}
	return b.String()
}

func diskWhenOf(when cpWhen) diskWhen {
	out := diskWhen{
		AnyOf:   when.anyOf,
		NoneOf:  when.noneOf,
		Spouse:  when.spouse,
		Places:  when.places,
		Assumed: when.assumed,
		Cond:    when.conditional,
	}
	if when.config != nil {
		out.Config = make([]diskConfig, len(when.config))
		for i, config := range when.config {
			out.Config[i] = diskConfig{Field: config.field, Values: config.values, AllowMultiple: config.allowMultiple}
		}
	}
	if when.dynamic != nil {
		out.Dynamic = make([]diskDynamicCondition, len(when.dynamic))
		for i, condition := range when.dynamic {
			out.Dynamic[i] = diskDynamicCondition{
				Name:     condition.name,
				Values:   condition.values,
				Contains: condition.contains,
				Expected: condition.expected,
			}
		}
	}
	if when.flags != nil {
		out.Flags = make([]diskFlagCondition, len(when.flags))
		for i, flag := range when.flags {
			out.Flags[i] = diskFlagCondition{Name: flag.name, Present: flag.present}
		}
	}
	return out
}

func cpWhenOfDisk(when diskWhen) cpWhen {
	out := cpWhen{
		anyOf:       when.AnyOf,
		noneOf:      when.NoneOf,
		spouse:      when.Spouse,
		places:      when.Places,
		assumed:     when.Assumed,
		conditional: when.Cond,
	}
	if when.Config != nil {
		out.config = make([]cpConfig, len(when.Config))
		for i, config := range when.Config {
			out.config[i] = cpConfig{field: config.Field, values: config.Values, allowMultiple: config.AllowMultiple}
		}
	}
	if when.Dynamic != nil {
		out.dynamic = make([]cpDynamicCondition, len(when.Dynamic))
		for i, condition := range when.Dynamic {
			out.dynamic[i] = cpDynamicCondition{
				name:     condition.Name,
				values:   condition.Values,
				contains: condition.Contains,
				expected: condition.Expected,
			}
		}
	}
	if when.Flags != nil {
		out.flags = make([]cpFlagCondition, len(when.Flags))
		for i, flag := range when.Flags {
			out.flags[i] = cpFlagCondition{name: flag.Name, present: flag.Present}
		}
	}
	return out
}

var packCache sync.Map // folder path -> cachedPack

// packValidated holds the folders whose cached pack was checked against its files during the running Check, so a
// pack read many times in one Check is stat'ed once. Check clears it at both ends, never mid-check.
var packValidated sync.Map

func readContentPack(im framework.Mod) cachedPack {
	return readContentPackWithEnabled(im, true)
}

func readContentPackForCleanup(im framework.Mod) cachedPack {
	return readContentPackWithEnabled(im, false)
}

func readContentPackWithEnabled(im framework.Mod, requireEnabled bool) cachedPack {
	if (requireEnabled && !im.Enabled) || im.Folder == "" || !isContentPatcherPack(im) {
		return cachedPack{}
	}
	root := filepath.Clean(im.Folder)
	if c, ok := packCache.Load(root); ok {
		if _, fresh := packValidated.Load(root); fresh {
			if got, ok := c.(cachedPack); ok {
				return got
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "content.json")); err != nil {
		return cachedPack{}
	}
	if c, ok := packCache.Load(root); ok {
		got, ok := c.(cachedPack)
		if ok && packFingerprintValid(root, got.files, got.fingerprint) {
			packValidated.Store(root, true)
			return got
		}
	}
	cachePath := loadPackDiskCache()
	if cachePath != "" {
		if entry, ok := diskPackEntryFor(root); ok && packFingerprintValid(root, entry.Files, entry.Fingerprint) {
			if disk, ok := decodeDiskPack(entry.Pack); ok {
				pack := cachedPackOfDisk(root, disk, entry)
				packCache.Store(root, pack)
				packValidated.Store(root, true)
				return pack
			}
		}
	}
	// Every file is stamped before it is read, and a file changed since its stamp leaves the pack uncached, so a cache
	// entry never pairs a fresh stamp with what was read before the change.
	pack := cachedPack{mentions: map[string]bool{}}
	pack.recordPackFile(root, filepath.Join(root, "content.json"))
	pack.recordPackFile(root, filepath.Join(root, "config.json"))
	pack.schema = readConfigSchema(root)
	pack.values = packConfigValues(pack.schema, root)
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	slices.SortFunc(pack.files, func(a, b packFileStamp) int {
		return strings.Compare(a.Path, b.Path)
	})
	pack.fingerprint = packFilesFingerprint(pack.files)
	if !packFingerprintValid(root, pack.files, pack.fingerprint) {
		noteUnstampable()
		dropPNGAlphaUnder(root)
		return pack
	}
	packCache.Store(root, pack)
	packValidated.Store(root, true)
	if cachePath != "" {
		if disk, err := encodeDiskPack(diskPackOf(pack)); err == nil {
			storeDiskPackEntry(root, diskPackEntry{Fingerprint: pack.fingerprint, Files: slices.Clone(pack.files), Pack: disk})
		}
	}
	dropPNGAlphaUnder(root)
	return pack
}

func (p *cachedPack) recordPackFile(root, abs string) {
	if !datadir.UnderRoot(root, abs) {
		return
	}
	relative, err := filepath.Rel(root, abs)
	if err != nil {
		return
	}
	stamp := packFileStamp{Path: filepath.ToSlash(relative), Size: absentSize}
	info, err := os.Stat(abs)
	switch {
	case err == nil:
		stamp.Size, stamp.ModTime = info.Size(), info.ModTime().UnixNano()
	case !errors.Is(err, fs.ErrNotExist):
		return
	}
	// The first stamp is the one taken before the file was read.
	if slices.ContainsFunc(p.files, func(f packFileStamp) bool { return f.Path == stamp.Path }) {
		return
	}
	p.files = append(p.files, stamp)
}

func packFilesFingerprint(files []packFileStamp) string {
	raw, _ := json.Marshal(files)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func packFingerprintValid(root string, files []packFileStamp, fingerprint string) bool {
	if len(files) == 0 || fingerprint == "" {
		return false
	}
	current := make([]packFileStamp, len(files))
	for i, expected := range files {
		abs, ok := inside(root, expected.Path)
		if !ok {
			return false
		}
		info, err := os.Stat(abs)
		if expected.Size == absentSize {
			if !errors.Is(err, fs.ErrNotExist) {
				return false
			}
		} else if err != nil || info.Size() != expected.Size || info.ModTime().UnixNano() != expected.ModTime {
			return false
		}
		current[i] = expected
	}
	return packFilesFingerprint(current) == fingerprint
}

func contentPackCachePath() (string, error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cache", "problems-content-packs.json"), nil
}

func zipPackCache(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(raw); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unzipPackCache(raw []byte) ([]byte, bool) {
	if len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		reader, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, false
		}
		payload, err := io.ReadAll(reader)
		_ = reader.Close()
		return payload, err == nil
	}
	return raw, true
}

func loadPackDiskCache() string {
	path, err := contentPackCachePath()
	if err != nil {
		return ""
	}
	packDiskState.Lock()
	defer packDiskState.Unlock()
	if packDiskState.loaded && packDiskState.path == path {
		return path
	}
	packDiskState.loaded = true
	packDiskState.path = path
	packDiskState.entries = map[string]diskPackEntry{}
	packDiskState.dirty = false
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return path
	}
	if cache, ok := decodePackCache(raw); ok {
		packDiskState.entries = cache.Packs
	}
	return path
}

// The pack cache is gob, not JSON: it holds every pack's cell sets, tens of MB that a start decodes in full, and gob
// reads them several times faster. It is left uncompressed for the same reason.
func encodePackCache(cache diskPackCache) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(cache)
	return buf.Bytes(), err
}

// decodePackCache reads a cache this parser version wrote; anything else reads as no cache.
func decodePackCache(raw []byte) (diskPackCache, bool) {
	var cache diskPackCache
	if gob.NewDecoder(bytes.NewReader(raw)).Decode(&cache) != nil || cache.Version != contentPackParserVersion || cache.Packs == nil {
		return diskPackCache{}, false
	}
	return cache, true
}

func encodeDiskPack(pack diskCachedPack) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(pack)
	return buf.Bytes(), err
}

func decodeDiskPack(raw []byte) (diskCachedPack, bool) {
	var pack diskCachedPack
	return pack, gob.NewDecoder(bytes.NewReader(raw)).Decode(&pack) == nil
}

func diskPackEntryFor(root string) (diskPackEntry, bool) {
	packDiskState.Lock()
	defer packDiskState.Unlock()
	entry, ok := packDiskState.entries[root]
	return entry, ok
}

func storeDiskPackEntry(root string, entry diskPackEntry) {
	packDiskState.Lock()
	defer packDiskState.Unlock()
	if packDiskState.entries == nil {
		packDiskState.entries = map[string]diskPackEntry{}
	}
	packDiskState.entries[root] = entry
	packDiskState.dirty = true
}

func flushPackDiskCache(mods []framework.Mod) {
	present := map[string]bool{}
	for _, im := range mods {
		if im.Folder != "" {
			present[filepath.Clean(im.Folder)] = true
		}
	}
	defer prunePackCache(present)
	path := loadPackDiskCache()
	if path == "" {
		return
	}
	packDiskState.Lock()
	for root := range packDiskState.entries {
		if _, err := os.Stat(root); err != nil {
			delete(packDiskState.entries, root)
			packDiskState.dirty = true
		}
	}
	if !packDiskState.dirty {
		packDiskState.Unlock()
		return
	}
	packed, err := encodePackCache(diskPackCache{
		Version: contentPackParserVersion,
		Packs:   packDiskState.entries,
	})
	if err != nil {
		packDiskState.Unlock()
		return
	}
	packDiskState.dirty = false
	packDiskState.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		packDiskState.Lock()
		packDiskState.dirty = true
		packDiskState.Unlock()
		return
	}
	if err := datadir.WriteFile(path, packed, 0o600); err != nil {
		packDiskState.Lock()
		packDiskState.dirty = true
		packDiskState.Unlock()
	}
}

// prunePackCache bounds the memoised packs to the folders of the mods just checked.
func prunePackCache(present map[string]bool) {
	packCache.Range(func(k, _ any) bool {
		if root, ok := k.(string); !ok || !present[root] {
			packCache.Delete(k)
		}
		return true
	})
}

type cpSchema struct {
	key           string
	defaultValue  string
	allowMultiple bool
	allowValues   []string
	allowBlank    bool
	description   string
}

func readConfigSchema(root string) map[string]cpSchema {
	raw, err := fsx.ReadFile(filepath.Join(root, "content.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		ConfigSchema map[string]json.RawMessage `json:"ConfigSchema"`
	}
	if json.Unmarshal(jsonc.Clean(raw), &doc) != nil {
		return nil
	}
	out := make(map[string]cpSchema, len(doc.ConfigSchema))
	for key, raw := range doc.ConfigSchema {
		var entry struct {
			Default       json.RawMessage `json:"Default"`
			AllowMultiple bool            `json:"AllowMultiple"`
			AllowValues   string          `json:"AllowValues"`
			AllowBlank    bool            `json:"AllowBlank"`
			Description   string          `json:"Description"`
		}
		if json.Unmarshal(raw, &entry) != nil {
			continue
		}
		defaultValue, _ := scalarValue(entry.Default)
		key = strings.TrimSpace(key)
		if key == "" || hasToken(key) {
			continue
		}
		out[strings.ToLower(key)] = cpSchema{
			key:           key,
			defaultValue:  defaultValue,
			allowMultiple: entry.AllowMultiple,
			allowValues:   splitTargets(entry.AllowValues),
			allowBlank:    entry.AllowBlank,
			description:   entry.Description,
		}
	}
	return out
}

func isContentPatcherPack(im framework.Mod) bool {
	return mod.Equal(im.ContentPackForID(), contentPatcherID)
}

func scanContentFile(root, rel string, seen map[string]bool, outer cpWhen, pack *cachedPack) {
	rel = filepath.ToSlash(rel)
	key := strings.ToLower(rel)
	if rel == "" || seen[key] {
		return
	}
	seen[key] = true
	abs, ok := inside(root, rel)
	if !ok {
		return
	}
	pack.recordPackFile(root, abs)
	raw, err := fsx.ReadFile(abs)
	if err != nil {
		return
	}
	var doc struct {
		Changes       []json.RawMessage `json:"Changes"`
		DynamicTokens []struct {
			Name  string                     `json:"Name"`
			Value json.RawMessage            `json:"Value"`
			When  map[string]json.RawMessage `json:"When"`
		} `json:"DynamicTokens"`
	}
	if err := json.Unmarshal(jsonc.Clean(raw), &doc); err != nil {
		return
	}
	// A dynamic token's value gates content like a change does, so its conditions count for compatibility
	// settings too. Content Patcher reads DynamicTokens only from content.json.
	if key == "content.json" {
		for _, tok := range doc.DynamicTokens {
			withConfigValues(tok.When, pack.values)
		}
		for _, tok := range doc.DynamicTokens {
			value, ok := scalarValue(tok.Value)
			if !ok {
				continue
			}
			pack.tokens = append(pack.tokens, cpTokenDefinition{name: tokenName(tok.Name), value: value, when: tok.When})
		}
		for _, tok := range doc.DynamicTokens {
			value, ok := scalarValue(tok.Value)
			if !ok {
				continue
			}
			pack.patches = append(pack.patches, cpPatch{kind: "other", when: outer.with(parseWhen(tok.When, pack.mentions, pack.schema)), tokenName: strings.ToLower(strings.TrimSpace(tok.Name)), tokenValue: value})
		}
	}
	warmOverlayImages(root, rel, doc.Changes)
	var dynamicValues map[string]string
	for i, rawChange := range doc.Changes {
		var ch cpChange
		if json.Unmarshal(rawChange, &ch) != nil {
			continue
		}
		action := strings.TrimSpace(ch.Action)
		withConfigValues(ch.When, pack.values)
		when := outer.with(parseWhenWithTokens(ch.When, pack.mentions, pack.schema, pack.tokens))
		when.conditional = when.conditional || len(ch.When) > 0
		var kind string
		var shapes []cpShape
		switch {
		case strings.EqualFold(action, "Include"):
			for _, from := range splitTargets(ch.FromFile) {
				from = contentSourceReference(root, rel, from)
				if hasToken(from) {
					pack.skips++
					continue
				}
				scanContentFile(root, from, seen, when, pack)
			}
			continue
		case strings.EqualFold(action, kindLoad):
			kind = "load"
			action = kindLoad
			ch.FromFile = contentSourceReference(root, rel, ch.FromFile)
		case strings.EqualFold(action, kindEditImage), strings.EqualFold(action, kindEditMap):
			kind = "edit"
			if strings.EqualFold(action, kindEditImage) {
				action = kindEditImage
			} else {
				action = kindEditMap
			}
			ch.FromFile = contentSourceReference(root, rel, ch.FromFile)
			recordReferencedPackFiles(root, ch.FromFile, action == kindEditImage, pack)
			shapes = editShapes(root, ch, action == kindEditImage)
			if len(shapes) == 0 {
				kind = "other"
			}
		case strings.EqualFold(action, kindEditData):
			kind = "edit"
			action = kindEditData
			if normalizeTarget(ch.Target) == "data/additionalfarms" {
				pack.recordFarms(ch.Entries)
			}
			shapes = dataShapes(root, ch, pack.values)
			if len(shapes) == 0 {
				kind = "other"
			}
		default:
			kind = "other"
		}
		// Changes that cannot conflict still count for compatibility settings, whatever their target.
		if kind == "other" {
			pack.patches = append(pack.patches, cpPatch{kind: kind, when: when, source: rel, index: i, action: action})
			continue
		}
		fromArea := string(jsonc.Clean(ch.FromArea))
		toArea := string(jsonc.Clean(ch.ToArea))
		priority, _ := scalarValue(ch.Priority)
		extra := changeDoesMore(rawChange, ch)
		for _, t := range splitTargets(ch.Target) {
			targets := []tokenTarget{{target: t, fromFile: ch.FromFile, when: when}}
			if hasToken(t) {
				if dynamicValues == nil {
					dynamicValues = singleDynamicValues(pack)
				}
				if targets = expandTargetTokens(t, ch.FromFile, when, pack.values, dynamicValues); targets == nil {
					pack.skips++
					continue
				}
			}
			for _, tt := range targets {
				pack.patches = append(pack.patches, cpPatch{
					kind: kind, target: normalizeTarget(tt.target), fromFile: tt.fromFile, priority: strings.TrimSpace(priority),
					patchMode: strings.TrimSpace(ch.PatchMode), when: tt.when,
					shapes: shapes, spouse: tt.when.spouse, places: tt.when.places, image: action == kindEditImage,
					imageDigest:   imageFileDigest(root, tt.fromFile, action == kindEditImage),
					imageFromArea: fromArea,
					source:        rel, index: i, action: action, toArea: toArea,
					extra: extra,
				})
			}
		}
	}
}

type tokenTarget struct {
	target   string
	fromFile string
	when     cpWhen
}

var (
	plainToken  = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.]+)\s*\}\}`)
	seasonToken = regexp.MustCompile(`(?i)\{\{\s*season\s*\}\}`)
	seasons     = []string{"spring", "summer", "fall", "winter"}
)

// expandTargetTokens resolves a tokenized target with the pack's config values and its dynamic tokens that
// have one value under that config. {{Season}} becomes one target per season, each gated on that season.
// It returns nil when a token stays unknown.
func expandTargetTokens(target, fromFile string, when cpWhen, config, dynamic map[string]string) []tokenTarget {
	resolve := func(s, season string) (string, bool) {
		ok := true
		out := plainToken.ReplaceAllStringFunc(s, func(m string) string {
			name := strings.ToLower(plainToken.FindStringSubmatch(m)[1])
			if name == "season" && season != "" {
				return season
			}
			if value, found := config[name]; found && value != "" && !strings.Contains(value, ",") {
				return value
			}
			if value, found := dynamic[name]; found {
				return value
			}
			ok = false
			return m
		})
		return out, ok && !hasToken(out)
	}
	cycle := []string{""}
	if seasonToken.MatchString(target) {
		cycle = seasons
		if limited := when.places["season"]; len(limited) > 0 {
			cycle = limited
		}
	}
	var out []tokenTarget
	for _, season := range cycle {
		t, ok := resolve(target, season)
		if !ok {
			return nil
		}
		from := fromFile
		if resolved, ok := resolve(fromFile, season); ok {
			from = resolved
		}
		w := when
		if season != "" {
			w = when.with(cpWhen{})
			w.places["season"] = []string{season}
		}
		out = append(out, tokenTarget{target: t, fromFile: from, when: w})
	}
	return out
}

// singleDynamicValues are the pack's dynamic tokens that hold exactly one value under its config. A
// definition whose When needs more than config fields may or may not apply, so its value stays possible.
func singleDynamicValues(pack *cachedPack) map[string]string {
	possible := map[string][]string{}
	unknown := map[string]bool{}
	for _, definition := range pack.tokens {
		if hasToken(definition.value) {
			unknown[definition.name] = true
			continue
		}
		switch configOnlyState(definition.when, pack) {
		case cpConditionTrue:
			possible[definition.name] = []string{definition.value}
		case cpConditionUnknown:
			if !slices.Contains(possible[definition.name], definition.value) {
				possible[definition.name] = append(possible[definition.name], definition.value)
			}
		case cpConditionFalse:
		}
	}
	out := map[string]string{}
	for name, values := range possible {
		if len(values) == 1 && !unknown[name] {
			out[name] = values[0]
		}
	}
	return out
}

func configOnlyState(raw map[string]json.RawMessage, pack *cachedPack) cpConditionState {
	if len(raw) == 0 {
		return cpConditionTrue
	}
	w := parseWhen(raw, map[string]bool{}, pack.schema)
	if len(w.config) != len(raw) || len(w.anyOf) > 0 || len(w.noneOf) > 0 || len(w.dynamic) > 0 || len(w.flags) > 0 {
		return cpConditionUnknown
	}
	if configHolds(w.config, pack.schema, pack.values) {
		return cpConditionTrue
	}
	return cpConditionFalse
}

func recordReferencedPackFiles(root, rel string, image bool, pack *cachedPack) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return
	}
	if image {
		for _, file := range sourceFiles(root, rel) {
			if abs, ok := inside(root, file); ok {
				pack.recordPackFile(root, abs)
			}
		}
		return
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if ext != ".tmx" && ext != ".tmj" {
		return
	}
	if abs, ok := inside(root, rel); ok {
		pack.recordPackFile(root, abs)
	}
}

// parseWhen reads the HasMod and schema-backed config conditions of a When block.
// It understands "HasMod": "A, B" and "HasMod |contains=A, B": true/false.
// withConfigValues replaces a condition value that is just one of the pack's config tokens, such as
// "FarmType": "{{FarmToReplace}}", with that field's configured value, which is what Content Patcher compares.
func withConfigValues(when map[string]json.RawMessage, values map[string]string) {
	for k, v := range when {
		if !hasToken(string(v)) {
			continue
		}
		m := singleToken.FindStringSubmatch(strings.TrimSpace(string(v)))
		if m == nil {
			continue
		}
		if value, ok := values[strings.ToLower(m[1])]; ok {
			when[k], _ = json.Marshal(value)
		}
	}
}

func parseWhen(raw map[string]json.RawMessage, mentions map[string]bool, schema map[string]cpSchema) cpWhen {
	return parseWhenWithTokens(raw, mentions, schema, nil)
}

func parseWhenWithTokens(raw map[string]json.RawMessage, mentions map[string]bool, schema map[string]cpSchema, tokens []cpTokenDefinition) cpWhen {
	return parseWhenDepth(raw, mentions, schema, tokens, 0)
}

func parseWhenDepth(raw map[string]json.RawMessage, mentions map[string]bool, schema map[string]cpSchema, tokens []cpTokenDefinition, depth int) cpWhen {
	var w cpWhen
	for k, v := range raw {
		if hasToken(k) {
			if merged, ok := dynamicTokenWhen(k, v, tokens, mentions, schema, depth); ok {
				w = w.withDefinition(merged)
			} else {
				w.assumed = appendAssumed(w.assumed, k, v, tokens)
			}
			continue
		}
		name, arg, _ := strings.Cut(k, "|")
		name = tokenName(name)
		if merged, ok := dynamicTokenWhen(k, v, tokens, mentions, schema, depth); ok {
			w = w.withDefinition(merged)
			continue
		}
		if strings.EqualFold(name, "hasmod") {
			arg = strings.TrimSpace(arg)
			if arg == "" {
				var list string
				if json.Unmarshal(v, &list) != nil || hasToken(list) {
					continue
				}
				ids := splitTargets(list)
				if len(ids) == 0 {
					continue
				}
				w.anyOf = append(w.anyOf, note(ids, mentions))
				continue
			}
			param, list, ok := strings.Cut(arg, "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) {
				continue
			}
			ids := note(splitTargets(list), mentions)
			var flags []string
			if !condValues(v, &flags) || len(flags) != 1 {
				continue
			}
			switch strings.ToLower(flags[0]) {
			case "true":
				w.anyOf = append(w.anyOf, ids)
			case "false":
				w.noneOf = append(w.noneOf, ids...)
			}
			continue
		}
		if flags, ok := flagConditions(k, v); ok {
			if len(flags) == 0 {
				w.assumed = appendAssumed(w.assumed, k, v, tokens)
			}
			w.flags = append(w.flags, flags...)
			continue
		}
		field, ok := schema[strings.ToLower(name)]
		if !ok {
			w.assumed = appendAssumed(w.assumed, k, v, tokens)
			continue
		}
		var values []string
		if !condValues(v, &values) {
			continue
		}
		if arg = strings.TrimSpace(arg); arg != "" {
			// "Field |contains=A, B": true accepts A or B; false accepts every other allowed value.
			param, list, ok := strings.Cut(arg, "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) || len(values) != 1 {
				continue
			}
			named := splitTargets(list)
			switch strings.ToLower(values[0]) {
			case "true":
				values = named
			case "false":
				values = slices.DeleteFunc(slices.Clone(field.allowValues), func(a string) bool {
					return slices.ContainsFunc(named, func(n string) bool { return strings.EqualFold(n, a) })
				})
			default:
				continue
			}
			if len(values) == 0 {
				continue
			}
		}
		w.config = append(w.config, cpConfig{field: field.key, values: values, allowMultiple: field.allowMultiple})
	}
	return w.with(cpWhen{spouse: spouseOf(raw, tokens), places: placesOf(raw)})
}

// appendAssumed records a condition this scan treats as met, unless it is a spouse or place condition the
// summary already names.
func appendAssumed(assumed []string, key string, raw json.RawMessage, tokens []cpTokenDefinition) []string {
	one := map[string]json.RawMessage{key: raw}
	if spouseOf(one, tokens) != "" || placesOf(one) != nil {
		return assumed
	}
	key = strings.TrimSpace(key)
	if _, arg, ok := strings.Cut(key, "|"); ok && strings.Contains(strings.ToLower(arg), "contains=") {
		var flags []string
		if condValues(raw, &flags) && len(flags) == 1 && strings.EqualFold(flags[0], "false") {
			key = "not " + key
		}
	}
	return append(assumed, key)
}

func dynamicTokenConditionParts(key string) (string, string, bool) {
	key = strings.TrimSpace(key)
	if strings.HasPrefix(key, "{{") {
		if !strings.HasSuffix(key, "}}") {
			return "", "", false
		}
		key = strings.TrimSpace(key[2 : len(key)-2])
	}
	name, arg, _ := strings.Cut(key, "|")
	name = tokenName(name)
	return name, strings.TrimSpace(arg), name != ""
}

func dynamicTokenWhen(key string, raw json.RawMessage, tokens []cpTokenDefinition, mentions map[string]bool, schema map[string]cpSchema, depth int) (cpWhen, bool) {
	condition, ok := dynamicTokenCondition(key, raw)
	if !ok || !slices.ContainsFunc(tokens, func(definition cpTokenDefinition) bool {
		return definition.name == condition.name
	}) {
		return cpWhen{}, false
	}
	out := cpWhen{dynamic: []cpDynamicCondition{condition}}
	if condition.contains != "" && !condition.expected {
		return out, true
	}
	match := -1
	for i, definition := range tokens {
		if definition.name != condition.name || !dynamicConditionMatchesValue(condition, definition.value) {
			continue
		}
		if match >= 0 {
			match = -1
			break
		}
		match = i
	}
	if match >= 0 && depth < 8 {
		out = out.withDefinition(parseWhenDepth(tokens[match].when, mentions, schema, tokens, depth+1))
	}
	return out, true
}

func dynamicTokenCondition(key string, raw json.RawMessage) (cpDynamicCondition, bool) {
	name, arg, ok := dynamicTokenConditionParts(key)
	if !ok {
		return cpDynamicCondition{}, false
	}
	if arg == "" {
		var values []string
		if !condValues(raw, &values) {
			return cpDynamicCondition{}, false
		}
		return cpDynamicCondition{name: name, values: values, expected: true}, true
	}
	param, list, ok := strings.Cut(arg, "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) {
		return cpDynamicCondition{}, false
	}
	var flags []string
	if !condValues(raw, &flags) || len(flags) != 1 {
		return cpDynamicCondition{}, false
	}
	expected, err := strconv.ParseBool(strings.TrimSpace(flags[0]))
	if err != nil {
		return cpDynamicCondition{}, false
	}
	return cpDynamicCondition{name: name, contains: strings.TrimSpace(list), expected: expected}, true
}

func dynamicConditionMatchesValue(condition cpDynamicCondition, value string) bool {
	if condition.contains != "" {
		return condition.expected && strings.EqualFold(strings.TrimSpace(condition.contains), strings.TrimSpace(value))
	}
	return slices.ContainsFunc(condition.values, func(want string) bool {
		return strings.EqualFold(strings.TrimSpace(want), strings.TrimSpace(value))
	})
}

type cpConditionState uint8

const (
	cpConditionUnknown cpConditionState = iota
	cpConditionFalse
	cpConditionTrue
)

type cpTokenReachability struct {
	values map[string]bool
	known  bool
}

func reachableDynamicTokens(tokens []cpTokenDefinition, present map[string]bool, schema map[string]cpSchema, config map[string]string, flags []cpFlagCondition) map[string]cpTokenReachability {
	reachable := map[string]cpTokenReachability{}
	for _, definition := range tokens {
		if definition.name == "" {
			continue
		}
		state := dynamicWhenState(definition.when, tokens, reachable, present, schema, config, flags)
		current, exists := reachable[definition.name]
		if !exists {
			current.known = true
			current.values = map[string]bool{}
		}
		switch state {
		case cpConditionTrue:
			current.values = map[string]bool{}
			current.known = true
		case cpConditionUnknown:
		case cpConditionFalse:
			reachable[definition.name] = current
			continue
		}
		if hasToken(definition.value) {
			current.known = false
			reachable[definition.name] = current
			continue
		}
		current.values[strings.ToLower(strings.TrimSpace(definition.value))] = true
		reachable[definition.name] = current
	}
	return reachable
}

func dynamicWhenState(raw map[string]json.RawMessage, tokens []cpTokenDefinition, reachable map[string]cpTokenReachability, present map[string]bool, schema map[string]cpSchema, config map[string]string, flags []cpFlagCondition) cpConditionState {
	state := cpConditionTrue
	for key, value := range raw {
		current := dynamicConditionState(key, value, tokens, reachable, present, schema, config, flags)
		if current == cpConditionFalse {
			return cpConditionFalse
		}
		if current == cpConditionUnknown {
			state = cpConditionUnknown
		}
	}
	return state
}

func dynamicConditionState(key string, raw json.RawMessage, tokens []cpTokenDefinition, reachable map[string]cpTokenReachability, present map[string]bool, schema map[string]cpSchema, config map[string]string, flags []cpFlagCondition) cpConditionState {
	name, arg, ok := dynamicTokenConditionParts(key)
	if !ok {
		return cpConditionUnknown
	}
	if strings.EqualFold(name, "hasmod") {
		return hasModConditionState(arg, raw, present)
	}
	if flagValues, recognized := flagConditions(key, raw); recognized {
		if len(flagValues) == 0 {
			return cpConditionUnknown
		}
		state := cpConditionTrue
		for _, flag := range flagValues {
			current := cpConditionUnknown
			for _, assumption := range flags {
				if assumption.name != flag.name {
					continue
				}
				if assumption.present != flag.present {
					return cpConditionFalse
				}
				current = cpConditionTrue
			}
			if current == cpConditionUnknown {
				state = cpConditionUnknown
			}
		}
		return state
	}
	if tokenDefinitionExists(tokens, name) {
		reachableToken, computed := reachable[name]
		if !computed || !reachableToken.known {
			return cpConditionUnknown
		}
		condition, valid := dynamicTokenCondition(key, raw)
		if !valid {
			return cpConditionUnknown
		}
		return dynamicConditionStateForReachable(condition, reachableToken)
	}
	if field, known := schema[strings.ToLower(name)]; known {
		condition, valid := configCondition(field, arg, raw)
		if !valid {
			return cpConditionUnknown
		}
		if configHolds([]cpConfig{condition}, schema, config) {
			return cpConditionTrue
		}
		return cpConditionFalse
	}
	return cpConditionUnknown
}

func dynamicConditionStateForReachable(condition cpDynamicCondition, reachable cpTokenReachability) cpConditionState {
	if !reachable.known {
		return cpConditionUnknown
	}
	if len(reachable.values) == 0 {
		return cpConditionFalse
	}
	matched := 0
	if condition.contains != "" {
		has := reachable.values[strings.ToLower(strings.TrimSpace(condition.contains))]
		if condition.expected {
			if !has {
				return cpConditionFalse
			}
			for value := range reachable.values {
				if value != strings.ToLower(strings.TrimSpace(condition.contains)) {
					return cpConditionUnknown
				}
			}
			return cpConditionTrue
		}
		if has {
			matched = len(reachable.values) - 1
		} else {
			matched = len(reachable.values)
		}
		if matched == 0 {
			return cpConditionFalse
		}
		if matched == len(reachable.values) {
			return cpConditionTrue
		}
		return cpConditionUnknown
	}
	for value := range reachable.values {
		if slices.ContainsFunc(condition.values, func(want string) bool {
			return strings.EqualFold(strings.TrimSpace(want), value)
		}) {
			matched++
		}
	}
	if matched == 0 {
		return cpConditionFalse
	}
	if matched == len(reachable.values) {
		return cpConditionTrue
	}
	return cpConditionUnknown
}

type reachableKey struct {
	tokens  *cpTokenDefinition
	present uintptr
	flags   string
}

// reachableMemo holds reachableDynamicTokens per pack, present set and flag set for the running Check; a pack's
// config is fixed for the Check, so it is not part of the key.
var reachableMemo = struct {
	sync.Mutex
	byKey map[reachableKey]map[string]cpTokenReachability
}{byKey: map[reachableKey]map[string]cpTokenReachability{}}

func cachedReachableTokens(tokens []cpTokenDefinition, present map[string]bool, schema map[string]cpSchema, config map[string]string, flags []cpFlagCondition) map[string]cpTokenReachability {
	if len(tokens) == 0 {
		return reachableDynamicTokens(tokens, present, schema, config, flags)
	}
	key := reachableKey{tokens: &tokens[0], present: reflect.ValueOf(present).Pointer(), flags: fmt.Sprint(flags)}
	reachableMemo.Lock()
	defer reachableMemo.Unlock()
	reachable, ok := reachableMemo.byKey[key]
	if !ok {
		reachable = reachableDynamicTokens(tokens, present, schema, config, flags)
		reachableMemo.byKey[key] = reachable
	}
	return reachable
}

func dynamicWhenHolds(when cpWhen, tokens []cpTokenDefinition, present map[string]bool, schema map[string]cpSchema, config map[string]string) bool {
	if len(when.dynamic) == 0 {
		return true
	}
	reachable := cachedReachableTokens(tokens, present, schema, config, when.flags)
	for _, condition := range when.dynamic {
		state, ok := reachable[condition.name]
		if !ok || !state.known {
			continue
		}
		if len(state.values) == 0 {
			return false
		}
		if condition.contains != "" {
			has := state.values[strings.ToLower(strings.TrimSpace(condition.contains))]
			if condition.expected && !has {
				return false
			}
			if !condition.expected && has && len(state.values) == 1 {
				return false
			}
			continue
		}
		if !slices.ContainsFunc(condition.values, func(value string) bool {
			return state.values[strings.ToLower(strings.TrimSpace(value))]
		}) {
			return false
		}
	}
	return true
}

func tokenDefinitionExists(tokens []cpTokenDefinition, name string) bool {
	return slices.ContainsFunc(tokens, func(definition cpTokenDefinition) bool {
		return definition.name == name
	})
}

func hasModConditionState(arg string, raw json.RawMessage, present map[string]bool) cpConditionState {
	if strings.TrimSpace(arg) == "" {
		var list string
		if json.Unmarshal(raw, &list) != nil || hasToken(list) {
			return cpConditionUnknown
		}
		ids := splitTargets(list)
		if len(ids) == 0 {
			return cpConditionUnknown
		}
		for _, id := range ids {
			if present[mod.SMAPI(id).Fold()] {
				return cpConditionTrue
			}
		}
		return cpConditionFalse
	}
	param, list, ok := strings.Cut(arg, "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) {
		return cpConditionUnknown
	}
	ids := splitTargets(list)
	var values []string
	if len(ids) == 0 || !condValues(raw, &values) || len(values) != 1 {
		return cpConditionUnknown
	}
	found := slices.ContainsFunc(ids, func(id string) bool { return present[mod.SMAPI(id).Fold()] })
	switch strings.ToLower(values[0]) {
	case "true":
		if found {
			return cpConditionTrue
		}
		return cpConditionFalse
	case "false":
		if found {
			return cpConditionFalse
		}
		return cpConditionTrue
	default:
		return cpConditionUnknown
	}
}

func configCondition(field cpSchema, arg string, raw json.RawMessage) (cpConfig, bool) {
	var values []string
	if !condValues(raw, &values) {
		return cpConfig{}, false
	}
	if strings.TrimSpace(arg) == "" {
		return cpConfig{field: field.key, values: values, allowMultiple: field.allowMultiple}, true
	}
	param, list, ok := strings.Cut(arg, "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) || len(values) != 1 {
		return cpConfig{}, false
	}
	named := splitTargets(list)
	switch strings.ToLower(values[0]) {
	case "true":
		values = named
	case "false":
		values = slices.DeleteFunc(slices.Clone(field.allowValues), func(a string) bool {
			return slices.ContainsFunc(named, func(n string) bool { return strings.EqualFold(n, a) })
		})
	default:
		return cpConfig{}, false
	}
	if len(values) == 0 {
		return cpConfig{}, false
	}
	return cpConfig{field: field.key, values: values, allowMultiple: field.allowMultiple}, true
}

func flagConditions(key string, raw json.RawMessage) ([]cpFlagCondition, bool) {
	name, arg, _ := strings.Cut(key, "|")
	if base, player, ok := strings.Cut(tokenName(name), ":"); ok && strings.TrimSpace(base) == "hasflag" && flagPlayers[strings.TrimSpace(player)] {
		name = base
	}
	if !strings.EqualFold(tokenName(name), "hasflag") {
		return nil, false
	}
	if strings.TrimSpace(arg) == "" {
		var values []string
		if !condValues(raw, &values) {
			return nil, true
		}
		out := make([]cpFlagCondition, 0, len(values))
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" || hasToken(value) {
				return nil, true
			}
			out = append(out, cpFlagCondition{name: strings.ToLower(value), present: true})
		}
		return out, true
	}
	param, list, ok := strings.Cut(arg, "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) {
		return nil, true
	}
	var values []string
	if !condValues(raw, &values) || len(values) != 1 {
		return nil, true
	}
	present, err := strconv.ParseBool(strings.TrimSpace(values[0]))
	if err != nil || strings.TrimSpace(list) == "" {
		return nil, true
	}
	return []cpFlagCondition{{name: strings.ToLower(strings.TrimSpace(list)), present: present}}, true
}

// flagPlayers are HasFlag's player inputs. A flag is read as the same condition whichever player holds it,
// since a profile's saves are not split by player.
var flagPlayers = map[string]bool{"currentplayer": true, "hostplayer": true, "anyplayer": true}

// condValues reads a condition value given as a string, comma list, bool or array; false when it holds a token.
func condValues(v json.RawMessage, out *[]string) bool {
	var str string
	var b bool
	var arr []json.RawMessage
	switch {
	case json.Unmarshal(v, &str) == nil:
	case json.Unmarshal(v, &b) == nil:
		str = strconv.FormatBool(b)
	case json.Unmarshal(v, &arr) == nil:
		for _, item := range arr {
			value, ok := scalarValue(item)
			if !ok || hasToken(value) {
				return false
			}
			str += "," + value
		}
	default:
		return false
	}
	if hasToken(str) {
		return false
	}
	*out = splitTargets(str)
	return len(*out) > 0
}

func scalarValue(v json.RawMessage) (string, bool) {
	var str string
	if json.Unmarshal(v, &str) == nil {
		return str, true
	}
	var b bool
	if json.Unmarshal(v, &b) == nil {
		return strconv.FormatBool(b), true
	}
	var n json.Number
	if json.Unmarshal(v, &n) == nil {
		return string(n), true
	}
	return "", false
}

func note(ids []string, mentions map[string]bool) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = mod.SMAPI(id).Fold()
		mentions[out[i]] = true
	}
	return out
}

type cpChange struct {
	Action        string                     `json:"Action"`
	Target        string                     `json:"Target"`
	FromFile      string                     `json:"FromFile"`
	Priority      json.RawMessage            `json:"Priority"`
	PatchMode     string                     `json:"PatchMode"`
	When          map[string]json.RawMessage `json:"When"`
	FromArea      json.RawMessage            `json:"FromArea"`
	ToArea        json.RawMessage            `json:"ToArea"`
	MapTiles      []json.RawMessage          `json:"MapTiles"`
	MapProperties map[string]json.RawMessage `json:"MapProperties"`
	Fields        json.RawMessage            `json:"Fields"`
	TargetField   []string                   `json:"TargetField"`
	Entries       json.RawMessage            `json:"Entries"`
}

func splitTargets(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func ContentReference(_, rel string) string {
	rel = strings.ReplaceAll(strings.TrimSpace(rel), "\\", "/")
	if rel == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
}

func contentSourceReference(_, _, rel string) string {
	return ContentReference("content.json", rel)
}

func hasToken(s string) bool {
	_, rest, ok := strings.Cut(s, "{{")
	return ok && strings.Contains(rest, "}}")
}

func normalizeTarget(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\\", "/")
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}
	return strings.ToLower(s)
}

func inside(root, rel string) (string, bool) {
	if filepath.IsAbs(rel) {
		return "", false
	}
	joined := filepath.Join(root, filepath.FromSlash(rel))
	if !datadir.UnderRoot(root, joined) {
		return "", false
	}
	return joined, true
}

func preloadContentPacks(mods []framework.Mod) { preloadPacks(mods, readContentPack) }

// preloadPacks parses packs on every core, so the serial passes after it find them cached.
func preloadPacks(mods []framework.Mod, read func(framework.Mod) cachedPack) {
	workers := max(1, runtime.GOMAXPROCS(0))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for _, im := range mods {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			_ = read(im)
		})
	}
	wg.Wait()
}

func dropPNGAlphaMemo() {
	pngAlphaCache.Range(func(k, v any) bool {
		if pix, ok := v.(pngAlpha); ok {
			releaseAlpha(pix.a)
		}
		pngAlphaCache.Delete(k)
		return true
	})
}

func dropPNGAlphaUnder(root string) {
	root = filepath.Clean(root)
	sep := root + string(os.PathSeparator)
	pngAlphaCache.Range(func(k, v any) bool {
		key, ok := k.(string)
		if !ok {
			return true
		}
		abs, _, _ := strings.Cut(key, "\x00")
		if abs != root && !strings.HasPrefix(abs, sep) {
			return true
		}
		if pix, ok := v.(pngAlpha); ok {
			releaseAlpha(pix.a)
		}
		pngAlphaCache.Delete(k)
		return true
	})
}

func dropShapeMemos() {
	shapeIndexes.Lock()
	clear(shapeIndexes.sets)
	shapeIndexes.Unlock()
	reachableMemo.Lock()
	clear(reachableMemo.byKey)
	reachableMemo.Unlock()
}

func clearPackValidated() {
	dropShapeMemos()
	packValidated.Range(func(k, _ any) bool {
		packValidated.Delete(k)
		return true
	})
}

func dropCheckScratch() {
	dropShapeMemos()
	dropPNGAlphaMemo()
	pngShapeCache.Range(func(k, _ any) bool {
		pngShapeCache.Delete(k)
		return true
	})
	cellSetCache.Range(func(k, _ any) bool {
		cellSetCache.Delete(k)
		return true
	})
	mapCache.Range(func(k, _ any) bool {
		mapCache.Delete(k)
		return true
	})
}

// assetConflictScan also returns the packs whose every change later packs overwrite (see shadowedPacks).
func assetConflictScan(mods []framework.Mod, run *partsRun) ([]framework.AssetConflict, []framework.SettingHint, []framework.Redundant) {
	defer dropCheckScratch()
	preloadContentPacks(mods)
	present := map[string]bool{}
	for _, im := range mods {
		if im.Enabled {
			present[im.ModID().Fold()] = true
		}
	}
	at := map[string]map[string][]packHit{"load": {}, "edit": {}}
	var authored []authoredPack
	type hitAt struct{ kind, target, id string }
	index := map[hitAt]int{}
	for _, im := range mods {
		pack := readContentPack(im)
		knows := maps.Clone(pack.mentions)
		dependencies := map[string]bool{}
		loadAfter := map[string]bool{}
		for _, id := range im.LoadAfter {
			loadAfter[id.Fold()] = true
		}
		for _, d := range im.Dependencies {
			if knows == nil {
				knows = map[string]bool{}
			}
			id := d.ModID().Fold()
			if loadAfter[id] {
				continue
			}
			knows[id] = true
			dependencies[id] = true
		}
		config := map[string]string{}
		if len(pack.schema) > 0 {
			config = readPackConfig(im.Folder)
		}
		seen := presentFor(pack, present)
		sig, stable := "", false
		if len(pack.patches) > 0 {
			sig, stable = packSig(im, pack, seen)
		}
		own := authoredPack{key: im.Key, name: im.Name, author: normalAuthor(im.Author), root: im.Folder}
		for _, p := range pack.patches {
			if p.kind == "other" || !p.when.holds(seen) || !dynamicWhenHolds(p.when, pack.tokens, seen, pack.schema, config) {
				continue
			}
			hits := at[p.kind][p.target]
			k := hitAt{p.kind, p.target, im.ModID().Fold()}
			i, found := index[k]
			if !found {
				hits = append(hits, packHit{
					id: im.ModID(), name: im.Name, key: im.Key, priority: p.priority, mentions: knows,
					root: im.Folder, tokens: pack.patches, present: seen, schema: pack.schema, config: config,
					dependencies: dependencies, loadAfter: loadAfter, sig: sig, stable: stable, author: normalAuthor(im.Author),
				})
				i = len(hits) - 1
				index[k] = i
				at[p.kind][p.target] = hits
			} else {
				hits[i].priority = strongerContentPatcherPriority(hits[i].priority, p.priority, p.kind)
				// A second pack with the same id adds its patches to the first one's hit, so its share of the key too.
				if hits[i].root != im.Folder && !strings.Contains(hits[i].sig, sig) {
					hits[i].sig += "\x01" + sig
					hits[i].stable = hits[i].stable && stable
				}
			}
			if p.kind == "edit" {
				hits[i].eligible = append(hits[i].eligible, p)
			}
			if !configHolds(p.when.config, pack.schema, config) {
				continue
			}
			if p.kind == "edit" {
				hits[i].edits = append(hits[i].edits, p)
			} else {
				hits[i].loads = append(hits[i].loads, p)
			}
			own.patches = append(own.patches, p)
		}
		if own.author != "" {
			authored = append(authored, own)
		}
	}
	shadowed := shadowedPacks(mods, at)
	bundled := bundles(authored)
	for _, key := range slices.Sorted(maps.Keys(bundled)) {
		if slices.ContainsFunc(shadowed, func(r framework.Redundant) bool { return r.Key == key }) {
			continue
		}
		im := mods[slices.IndexFunc(mods, func(m framework.Mod) bool { return m.Key == key })]
		shadowed = append(shadowed, framework.Redundant{Kind: "bundled", Key: key, ID: im.ModID(), Name: im.Name, By: []framework.ModRef{bundled[key]}})
	}
	customFarms := map[string]string{}
	for _, im := range mods {
		for target, id := range readContentPack(im).farms {
			customFarms[target] = modIDToken.ReplaceAllLiteralString(id, im.UniqueID)
		}
	}
	out := []framework.AssetConflict{}
	settings := []framework.SettingHint{}
	for _, kind := range []string{"load", "edit"} {
		targets := at[kind]
		for _, t := range slices.Sorted(maps.Keys(targets)) {
			hits := targets[t]
			if len(hits) < 2 {
				continue
			}
			farm := customFarms[t]
			fields := []string{kind, t, farm}
			stable := true
			for _, h := range hits {
				fields = append(fields, h.sig)
				stable = stable && h.stable
			}
			e := run.part(partKey(fields...), stable, func() partEntry { return targetPart(kind, t, hits, farm, bundled) })
			if e.Conflict != nil {
				out = append(out, *e.Conflict)
			}
			settings = append(settings, e.Settings...)
		}
	}
	slices.SortFunc(out, func(a, b framework.AssetConflict) int {
		if a.Kind != b.Kind {
			if a.Kind == "load" {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Target, b.Target)
	})
	return out, settings, shadowed
}

// targetPart is the outcome of one target that two or more packs touch: a conflict, a setting that settles it, or
// nothing.
// farm is the custom farm type whose map target is, or "".
func targetPart(kind, target string, hits []packHit, farm string, bundled map[string]framework.ModRef) partEntry {
	hits = withoutBundled(kind, hits, bundled)
	e := conflictPart(kind, target, hits, farm)
	if kind == "load" {
		e.Settings = append(e.Settings, deadFallbackSettings(target, hits, e.Conflict)...)
	}
	return e
}

// deadFallbackSettings are the settings that only turn on a fallback load another pack's load always beats: the
// pack is left out of the conflict, and its setting has no effect.
func deadFallbackSettings(target string, hits []packHit, conflict *framework.AssetConflict) []framework.SettingHint {
	var out []framework.SettingHint
	for _, h := range hits {
		if conflict != nil && slices.Contains(conflict.Keys, h.key) {
			continue
		}
	hit:
		for _, load := range h.loads {
			for _, other := range hits {
				if other.key == h.key {
					continue
				}
				for _, otherLoad := range other.loads {
					if !fallbackLoad(h, load, other, otherLoad) {
						continue
					}
					if hint := settingForDeadLoad(h, load, other, framework.AssetConflict{Target: target}); hint != nil {
						out = append(out, *hint)
						break hit
					}
				}
			}
		}
	}
	return out
}

func conflictPart(kind, target string, hits []packHit, farm string) partEntry {
	cosmetic := false
	var note *framework.ConflictNote
	if kind == "edit" {
		hits, cosmetic, note = clashing(hits)
	} else {
		hits = clashingLoads(hits)
	}
	if len(hits) < 2 {
		return partEntry{}
	}
	c := conflictOf(kind, target, hits)
	c.Evidence = conflictEvidence(kind, hits)
	if kind == "load" {
		if allLoadFilesBlank(hits, target) || allLoadFilesIdentical(hits, target) {
			return partEntry{}
		}
		var hint *framework.SettingHint
		c.Cosmetic, hint = harmlessLoads(hits, c)
		if hint != nil {
			return partEntry{Settings: []framework.SettingHint{*hint}}
		}
	}
	if kind == "edit" {
		c.Cosmetic, c.Note = cosmetic, note
		markLoadAfterWinner(&c, hits)
	}
	c.Fixes = []framework.ConflictFix{}
	for _, h := range hits {
		if fix, ok := switchOff(h, hits); ok {
			c.Fixes = append(c.Fixes, fix)
		}
	}
	c.Farms = farmNeeds(kind, hits, farm)
	return partEntry{Conflict: &c}
}

// farmNeeds is, per pack in conflict order, the farm types every one of its clashing patches is limited to,
// nil for a pack with one that applies on any farm; nil when no pack is limited.
func farmNeeds(kind string, hits []packHit, farm string) [][]string {
	out := make([][]string, len(hits))
	limited := false
	for i, h := range hits {
		patches, clashes := h.edits, h.clashes
		if kind == "load" {
			patches, clashes = h.loads, h.loadClashes
		}
		var need []string
		for j, p := range patches {
			if len(clashes) > 0 && !clashes[j] {
				continue
			}
			types := farmTypesOf(p, farm)
			if len(types) == 0 {
				need = nil
				break
			}
			for _, t := range types {
				if !slices.Contains(need, t) {
					need = append(need, t)
				}
			}
		}
		out[i] = need
		limited = limited || need != nil
	}
	if !limited {
		return nil
	}
	return out
}

// modIDToken is {{ModId}}, which Content Patcher fills with the pack's unique id, as a save records it.
var modIDToken = regexp.MustCompile(`(?i)\{\{\s*modid\s*\}\}`)

// recordFarms notes the map asset and id of each custom farm in Data/AdditionalFarms entries.
func (p *cachedPack) recordFarms(raw json.RawMessage) {
	var entries map[string]struct {
		ID      string `json:"Id"`
		MapName string `json:"MapName"`
	}
	if json.Unmarshal(jsonc.Clean(raw), &entries) != nil {
		return
	}
	for key, entry := range entries {
		if entry.MapName == "" || hasToken(entry.MapName) {
			continue
		}
		id := entry.ID
		if id == "" {
			id = key
		}
		if p.farms == nil {
			p.farms = map[string]string{}
		}
		p.farms["maps/"+strings.ToLower(entry.MapName)] = id
	}
}

func clashingLoads(hits []packHit) (out []packHit) {
	in := make([]bool, len(hits))
	for i := range hits {
		for j := i + 1; j < len(hits); j++ {
			for ai, a := range hits[i].loads {
				for bj, b := range hits[j].loads {
					if exclusive(a, b) || fallbackLoad(hits[i], a, hits[j], b) || fallbackLoad(hits[j], b, hits[i], a) {
						continue
					}
					in[i], in[j] = true, true
					if hits[i].loadClashes == nil {
						hits[i].loadClashes = map[int]bool{}
					}
					if hits[j].loadClashes == nil {
						hits[j].loadClashes = map[int]bool{}
					}
					hits[i].loadClashes[ai], hits[j].loadClashes[bj] = true, true
				}
			}
		}
	}
	for i, hit := range hits {
		if in[i] {
			out = append(out, hit)
		}
	}
	return out
}

// fallbackLoad reports a load below default priority that another pack's load of the same asset outranks:
// the author made it the fallback for when no stronger load is installed. Under a Medium load it is a
// fallback only when its pack names the other one; one a blank load wipes stays reported.
func fallbackLoad(h packHit, load cpPatch, other packHit, otherLoad cpPatch) bool {
	rank := contentPatcherPriority("load", load.priority)
	otherRank := contentPatcherPriority("load", otherLoad.priority)
	if rank >= 0 || rank >= otherRank {
		return false
	}
	if h.mentions[other.id.Fold()] {
		return true
	}
	return otherRank >= 1000 && (loadFileBlank(h, load, load.target) || !loadFileBlank(other, otherLoad, otherLoad.target))
}

// clashing keeps the packs that share an overlapping edit of one target with a pack they were not built
// alongside. Packs in one entry, or where one names the other as a dependency or in a HasMod condition,
// were patched to work together, so their overlaps are intended.
func clashing(hits []packHit) (out []packHit, cosmetic bool, why *framework.ConflictNote) {
	in := make([]bool, len(hits))
	cosmetic = true
	var note noteAgreement
	indexes := make([][]*shapeSet, len(hits))
	summaries := make([]shapeSummary, len(hits))
	for i := range hits {
		indexes[i] = indexesOf(hits[i].edits)
		summaries[i] = summarize(indexes[i])
	}
	for i := range hits {
		for j := i + 1; j < len(hits); j++ {
			if aware(hits[i], hits[j]) || !summaries[i].mayOverlap(summaries[j]) {
				continue
			}
			clash, minor, pairNote := editsClashIndexed(hits[i].edits, hits[j].edits, indexes[j], sameAuthor(hits[i], hits[j]))
			if !clash {
				continue
			}
			in[i], in[j] = true, true
			markClashes(&hits[i], &hits[j])
			cosmetic = cosmetic && minor
			note.add(pairNote)
		}
	}
	for i, h := range hits {
		if in[i] {
			out = append(out, h)
		}
	}
	if !cosmetic {
		return out, false, nil
	}
	return out, true, note.result()
}

func sameAuthor(a, b packHit) bool {
	return a.author != "" && a.author == b.author
}

func normalAuthor(author string) string {
	return strings.Join(strings.Fields(strings.ToLower(author)), " ")
}

// authoredPack is a pack's active Load and edit patches, kept while scanning so packs by one author can be compared.
type authoredPack struct {
	key, name, author, root string
	patches                 []cpPatch
}

// bundles finds the packs whose every active patch writes where another pack by the same author also writes:
// the larger pack bundles the smaller one. Of two packs with the same footprint, the one listed later is bundled.
func bundles(packs []authoredPack) map[string]framework.ModRef {
	byAuthor := map[string][]authoredPack{}
	for _, p := range packs {
		if p.author != "" && len(p.patches) > 0 {
			byAuthor[p.author] = append(byAuthor[p.author], p)
		}
	}
	var out map[string]framework.ModRef
	for _, group := range byAuthor {
		if len(group) < 2 {
			continue
		}
		// Targets first: they rule out almost every pair before any footprint is spelled out.
		targets := make([]map[string]bool, len(group))
		for i, p := range group {
			targets[i] = map[string]bool{}
			for _, patch := range p.patches {
				targets[i][patch.target] = true
			}
		}
		sigs := make([]map[string]bool, len(group))
		footprints := func(i int) map[string]bool {
			if sigs[i] == nil {
				sigs[i] = make(map[string]bool, len(group[i].patches))
				for _, patch := range group[i].patches {
					sigs[i][patchFootprint(patch)] = true
				}
			}
			return sigs[i]
		}
		for i, small := range group {
			for j, big := range group {
				if i == j || len(small.patches) > len(big.patches) && len(targets[i]) > len(targets[j]) {
					continue
				}
				if _, bundledToo := out[big.key]; bundledToo || !subsetOf(targets[i], targets[j]) {
					continue
				}
				a, b := footprints(i), footprints(j)
				if len(a) > len(b) || len(a) == len(b) && i < j || !subsetOf(a, b) || !sameLoads(small, big) {
					continue
				}
				if out == nil {
					out = map[string]framework.ModRef{}
				}
				out[small.key] = framework.ModRef{Key: big.key, Name: big.name}
				break
			}
		}
	}
	return out
}

// sameLoads reports whether big loads every file small loads, to the same target.
func sameLoads(small, big authoredPack) bool {
	for _, p := range small.patches {
		if p.kind != "load" {
			continue
		}
		digest := imageFileDigest(small.root, p.fromFile, true)
		if digest == "" || !slices.ContainsFunc(big.patches, func(o cpPatch) bool {
			return o.kind == "load" && o.target == p.target && imageFileDigest(big.root, o.fromFile, true) == digest
		}) {
			return false
		}
	}
	return true
}

func subsetOf(a, b map[string]bool) bool {
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// withoutBundled drops the packs another pack in hits bundles. Loads are judged only between two packs.
func withoutBundled(kind string, hits []packHit, bundled map[string]framework.ModRef) []packHit {
	if len(bundled) == 0 || kind == "load" && len(hits) != 2 {
		return hits
	}
	return slices.DeleteFunc(slices.Clone(hits), func(h packHit) bool {
		by, ok := bundled[h.key]
		return ok && slices.ContainsFunc(hits, func(o packHit) bool { return o.key == by.Key })
	})
}

// patchFootprint is where an edit writes, not what: packs by one author that write the same places are one
// bundled in the other even when their sources are laid out differently. A load replaces the whole asset, so
// sameLoads compares its file too.
func patchFootprint(p cpPatch) string {
	b := make([]byte, 0, 64)
	b = append(b, p.target...)
	b = append(b, '|')
	b = append(b, p.action...)
	b = append(b, '|')
	b = append(b, strings.ToLower(strings.TrimSpace(p.patchMode))...)
	if len(p.when.places) > 0 {
		b = fmt.Appendf(b, "|%v", p.when.places)
	}
	for _, shape := range p.shapes {
		b = append(b, '|', shape.kind)
		for _, n := range []int{shape.x, shape.y, shape.w, shape.h} {
			b = strconv.AppendInt(append(b, ','), int64(n), 10)
		}
		b = append(append(append(append(append(b, ','), shape.cells...), ','), shape.layer...), ',')
		b = append(b, shape.key...)
	}
	return string(b)
}

func aware(a, b packHit) bool {
	return (a.key != "" && a.key == b.key) || a.mentions[b.id.Fold()] || b.mentions[a.id.Fold()]
}

func conflictOf(kind, target string, hits []packHit) framework.AssetConflict {
	slices.SortFunc(hits, func(a, b packHit) int { return strings.Compare(a.id.Fold(), b.id.Fold()) })
	c := framework.AssetConflict{Kind: kind, Target: target, PackIDs: make([]mod.ID, len(hits)), Names: make([]string, len(hits)), Keys: make([]string, len(hits))}
	for i, h := range hits {
		c.PackIDs[i], c.Names[i], c.Keys[i] = h.id, h.name, h.key
	}
	if kind == "edit" && len(hits) >= 3 && strings.Contains(strings.ToLower(target), "objects") {
		c.Info = itemConflictInfo(hits)
	}
	best := -1
	bestRank := -1 << 31
	var tied []int
	exclusive := 0
	for i, hit := range hits {
		rank := hitPriority(kind, hit)
		if kind == "load" && hasExclusiveLoad(hit) {
			exclusive++
		}
		if rank > bestRank {
			best, bestRank, tied = i, rank, []int{i}
		} else if rank == bestRank {
			tied = append(tied, i)
		}
	}
	if kind == "load" && exclusive >= 2 {
		c.WinnerName = "CP applies neither"
		log.Printf("Content Patcher error: exclusive loads for %s leave the asset unchanged", target)
		return c
	}
	if best >= 0 && len(tied) == 1 {
		c.WinnerID, c.WinnerName = hits[best].id, hits[best].name
		for i, h := range hits {
			if i != best {
				c.Overridden = append(c.Overridden, h.name)
			}
		}
		return c
	}
	if kind == "load" {
		if winner, ok := dependencyLoadWinner(hits, tied); ok {
			c.WinnerID = hits[winner].id
			c.WinnerName = "by load order"
			for i, h := range hits {
				if i != winner {
					c.Overridden = append(c.Overridden, h.name)
				}
			}
			return c
		}
	}
	c.WinnerName = "unclear"
	return c
}

func itemConflictInfo(hits []packHit) string {
	common := map[string]bool{}
	for _, patch := range hits[0].edits {
		for _, shape := range patch.shapes {
			for cell := range shapeCells(shape) {
				common[cell] = true
			}
		}
	}
	for _, hit := range hits[1:] {
		cells := map[string]bool{}
		for _, patch := range hit.edits {
			for _, shape := range patch.shapes {
				for cell := range shapeCells(shape) {
					cells[cell] = true
				}
			}
		}
		for cell := range common {
			if !cells[cell] {
				delete(common, cell)
			}
		}
	}
	if len(common) == 0 {
		return ""
	}
	cells := slices.Sorted(maps.Keys(common))
	if len(cells) == 0 {
		return ""
	}
	parts := strings.Split(cells[0], ",")
	if len(parts) == 2 {
		x, xErr := strconv.Atoi(parts[0])
		y, yErr := strconv.Atoi(parts[1])
		if xErr == nil && yErr == nil {
			return "cell (" + strconv.Itoa(x*16) + ", " + strconv.Itoa(y*16) + ")"
		}
	}
	return "cell (" + strings.ReplaceAll(cells[0], ",", ", ") + ")"
}

func shapeCells(shape cpShape) map[string]bool {
	out := map[string]bool{}
	if shape.cells != "" {
		for cell := range strings.SplitSeq(shape.cells, ";") {
			if cell != "" {
				out[cell] = true
			}
		}
		return out
	}
	x, y, w, h := shape.area()
	if w <= 0 || h <= 0 || x%16 != 0 || y%16 != 0 || w%16 != 0 || h%16 != 0 {
		return out
	}
	for cy := y / 16; cy < (y+h)/16; cy++ {
		for cx := x / 16; cx < (x+w)/16; cx++ {
			out[strconv.Itoa(cx)+","+strconv.Itoa(cy)] = true
		}
	}
	return out
}

func hitPriority(kind string, hit packHit) int {
	best := -1
	found := false
	if kind == "load" {
		for i, patch := range hit.loads {
			if hit.loadClashes != nil && !hit.loadClashes[i] {
				continue
			}
			rank := contentPatcherPriority(kind, patch.priority)
			if !found || rank > best {
				best, found = rank, true
			}
		}
	} else {
		for i, patch := range hit.edits {
			if hit.clashes != nil && !hit.clashes[i] {
				continue
			}
			rank := contentPatcherPriority(kind, patch.priority)
			if !found || rank > best {
				best, found = rank, true
			}
		}
	}
	if found {
		return best
	}
	return contentPatcherPriority(kind, hit.priority)
}

func hasExclusiveLoad(hit packHit) bool {
	for i, patch := range hit.loads {
		if hit.loadClashes != nil && !hit.loadClashes[i] {
			continue
		}
		if exclusiveLoadPriority(patch.priority) {
			return true
		}
	}
	return false
}

// exclusiveLoadPriority reports a Load at Exclusive priority, which is also what Content Patcher gives a Load
// that sets no Priority.
func exclusiveLoadPriority(priority string) bool {
	priority = strings.TrimSpace(priority)
	return priority == "" || strings.EqualFold(priority, "exclusive")
}

func dependencyLoadWinner(hits []packHit, tied []int) (int, bool) {
	if len(tied) != 2 {
		return -1, false
	}
	left, right := tied[0], tied[1]
	leftDepends := hits[left].dependencies[hits[right].id.Fold()]
	rightDepends := hits[right].dependencies[hits[left].id.Fold()]
	if leftDepends == rightDepends {
		return -1, false
	}
	if leftDepends {
		return left, true
	}
	return right, true
}

func harmlessLoads(hits []packHit, conflict framework.AssetConflict) (bool, *framework.SettingHint) {
	if len(hits) < 2 {
		return false, nil
	}
	if conflict.WinnerName == "CP applies neither" {
		return false, nil
	}
	if allLoadFilesIdentical(hits, conflict.Target) {
		return true, nil
	}
	winner := slices.IndexFunc(hits, func(h packHit) bool { return mod.Equal(h.id, conflict.WinnerID) })
	if winner < 0 {
		for _, hit := range hits {
			for _, load := range hit.loads {
				if !loadFileBlank(hit, load, conflict.Target) {
					return false, nil
				}
			}
		}
		return true, nil
	}

	winnerBlank := true
	for _, load := range hits[winner].loads {
		winnerBlank = winnerBlank && loadFileBlank(hits[winner], load, conflict.Target)
	}
	var hint *framework.SettingHint
	for i, hit := range hits {
		if i == winner {
			continue
		}
		for _, load := range hit.loads {
			blank := loadFileBlank(hit, load, conflict.Target)
			identical := false
			for _, winningLoad := range hits[winner].loads {
				if loadFilesEqual(hit, load, hits[winner], winningLoad, conflict.Target) {
					identical = true
					break
				}
			}
			if blank || identical {
				continue
			}
			if winnerBlank {
				return false, settingForDeadLoad(hit, load, hits[winner], conflict)
			}
			if !loadPriorityDecided(hits, conflict) {
				return false, nil
			}
			if hitPriority("load", hit) > -1000 && !hit.mentions[hits[winner].id.Fold()] {
				return false, nil
			}
			if candidate := settingForDeadLoad(hit, load, hits[winner], conflict); candidate != nil {
				hint = candidate
			}
		}
	}
	return true, hint
}

func settingForDeadLoad(hit packHit, load cpPatch, winner packHit, conflict framework.AssetConflict) *framework.SettingHint {
	if hitPriority("load", hit) > -1000 {
		return nil
	}
	for _, condition := range load.when.config {
		schema, ok := hit.schema[strings.ToLower(condition.field)]
		if !ok || schema.defaultValue == "" {
			continue
		}
		current := hit.config[strings.ToLower(condition.field)]
		if current == "" {
			current = schema.defaultValue
		}
		if current == schema.defaultValue || !configHolds([]cpConfig{condition}, hit.schema, hit.config) {
			continue
		}
		return &framework.SettingHint{
			Key: hit.key, ID: hit.id, Name: hit.name, Field: schema.key,
			Current: current, Suggested: []string{schema.defaultValue},
			Description: "This setting has no effect because " + winner.name + " loads " + conflict.Target + " over it.",
		}
	}
	return nil
}

func allLoadFilesIdentical(hits []packHit, target string) bool {
	var reference any
	found := false
	for _, hit := range hits {
		for _, load := range hit.loads {
			raw, ok := loadFileBytes(hit, load, target)
			if !ok {
				return false
			}
			value, isJSON := loadJSONValue(raw)
			if !isJSON {
				value = raw
			}
			if !found {
				reference, found = value, true
				continue
			}
			if !reflect.DeepEqual(reference, value) {
				return false
			}
		}
	}
	return found
}

func allLoadFilesBlank(hits []packHit, target string) bool {
	found := false
	for _, hit := range hits {
		for _, load := range hit.loads {
			found = true
			if !loadFileBlank(hit, load, target) {
				return false
			}
		}
	}
	return found
}

func loadPriorityDecided(hits []packHit, conflict framework.AssetConflict) bool {
	if conflict.WinnerID == "" {
		return false
	}
	winner := slices.IndexFunc(hits, func(h packHit) bool { return mod.Equal(h.id, conflict.WinnerID) })
	if winner < 0 {
		return false
	}
	best := hitPriority("load", hits[winner])
	for i, hit := range hits {
		if i != winner && hitPriority("load", hit) >= best {
			return false
		}
	}
	return true
}

func loadFileBlank(hit packHit, load cpPatch, target string) bool {
	raw, ok := loadFileBytes(hit, load, target)
	if !ok {
		return false
	}
	raw = bytes.TrimSpace(jsonc.Clean(raw))
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) == nil && object != nil && len(object) == 0 {
		return true
	}
	var array []json.RawMessage
	return json.Unmarshal(raw, &array) == nil && array != nil && len(array) == 0
}

func loadFilesEqual(a packHit, ap cpPatch, b packHit, bp cpPatch, target string) bool {
	left, ok := loadFileBytes(a, ap, target)
	if !ok {
		return false
	}
	right, ok := loadFileBytes(b, bp, target)
	if !ok {
		return false
	}
	leftValue, leftJSON := loadJSONValue(left)
	rightValue, rightJSON := loadJSONValue(right)
	if leftJSON && rightJSON {
		return reflect.DeepEqual(leftValue, rightValue)
	}
	return bytes.Equal(left, right)
}

func loadJSONValue(raw []byte) (any, bool) {
	var value any
	if json.Unmarshal(jsonc.Clean(raw), &value) != nil {
		return nil, false
	}
	return value, true
}

func loadFileBytes(hit packHit, load cpPatch, target string) ([]byte, bool) {
	path, ok := resolveLoadFile(load.fromFile, target, hit.tokens, hit.present, hit.schema, hit.config)
	if !ok {
		return nil, false
	}
	return readPackPath(hit.root, path)
}

func resolveLoadFile(file, target string, tokens []cpPatch, present map[string]bool, schema map[string]cpSchema, config map[string]string) (string, bool) {
	file = strings.TrimSpace(file)
	for range 16 {
		start := strings.Index(file, "{{")
		if start < 0 {
			return file, true
		}
		end := strings.Index(file[start+2:], "}}")
		if end < 0 {
			return "", false
		}
		end += start + 2
		name := tokenName(file[start+2 : end])
		var value string
		switch name {
		case "target":
			value = target
		case "targetwithoutpath":
			value = target
			if slash := strings.LastIndexAny(value, "/\\"); slash >= 0 {
				value = value[slash+1:]
			}
		default:
			found := false
			for _, token := range tokens {
				if token.tokenName != name || !token.when.holds(present) || !configHolds(token.when.config, schema, config) {
					continue
				}
				value, found = token.tokenValue, true
			}
			if !found {
				return "", false
			}
		}
		file = file[:start] + value + file[end+2:]
	}
	return "", false
}

func readPackPath(root, rel string) ([]byte, bool) {
	if root == "" {
		return nil, false
	}
	path, ok := caseInsensitivePath(root, rel)
	if !ok {
		if abs, inRoot := inside(root, rel); inRoot {
			noteRead(abs, nil)
		}
		return nil, false
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(relative) {
		return nil, false
	}
	info, err := os.Stat(path)
	if err != nil {
		noteUnstampable()
		return nil, false
	}
	noteRead(path, info)
	file, err := os.OpenInRoot(root, relative)
	if err != nil {
		return nil, false
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(file)
	return raw, err == nil
}

func imageFileDigest(root, rel string, image bool) string {
	if !image {
		return ""
	}
	raw, ok := readPackPath(root, rel)
	if !ok {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func caseInsensitivePath(root, rel string) (string, bool) {
	if filepath.IsAbs(rel) || strings.TrimSpace(rel) == "" {
		return "", false
	}
	current := filepath.Clean(root)
	for part := range strings.SplitSeq(filepath.ToSlash(rel), "/") {
		if part == "" || part == "." || part == ".." {
			if part == ".." {
				return "", false
			}
			continue
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return "", false
		}
		next := ""
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), part) {
				next = entry.Name()
				break
			}
		}
		if next == "" {
			return "", false
		}
		current = filepath.Join(current, next)
	}
	return current, true
}

func contentPatcherPriority(kind, priority string) int {
	priority = strings.TrimSpace(priority)
	if priority == "" {
		if kind == "load" {
			return 3000
		}
		return 0
	}
	if numeric, err := strconv.Atoi(priority); err == nil {
		return numeric
	}
	base := priority
	offset := 0
	for i := 1; i < len(priority); i++ {
		if priority[i] != '+' && priority[i] != '-' {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(priority[i+1:]))
		if err != nil {
			return 0
		}
		if priority[i] == '-' {
			value = -value
		}
		base, offset = priority[:i], value
		break
	}
	switch strings.ToLower(strings.TrimSpace(base)) {
	case "low", "early":
		return -1000 + offset
	case "medium", "default":
		return offset
	case "high", "late":
		return 1000 + offset
	case "exclusive":
		return 3000 + offset
	default:
		return 0
	}
}

func strongerContentPatcherPriority(current, candidate, kind string) string {
	if contentPatcherPriority(kind, candidate) >= contentPatcherPriority(kind, current) {
		return strings.TrimSpace(candidate)
	}
	return strings.TrimSpace(current)
}
