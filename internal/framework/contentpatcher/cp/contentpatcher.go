package cp

import (
	"bufio"
	"bytes"
	"cmp"
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
	"iter"
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
	"unique"

	"github.com/Rethunk-Tech/go-lenientjson"
	"github.com/Rethunk-Tech/go-safefs"
)

var contentPatcherID = SMAPI("Pathoschild.ContentPatcher")

const (
	kindLoad      = "Load"
	kindEditImage = "EditImage"
	kindEditMap   = "EditMap"
	kindEditData  = "EditData"
)

type packHit struct {
	ID           ModID
	Name         string
	Key          string
	Priority     string
	Mentions     map[string]bool
	Root         string
	Tokens       []Patch
	Present      map[string]bool
	Loads        []Patch
	Edits        []Patch // the pack's active edits of this target
	Eligible     []Patch // edits whose HasMod conditions hold, including config-off variants
	ConfigOff    []int   // positions in eligible whose config conditions fail; only used while the scan collects
	LoadClashes  map[int]bool
	Dependencies map[string]bool
	LoadAfter    map[string]bool
	Schema       map[string]Schema
	Config       map[string]string
	Clashes      map[int]bool // indices into edits that overlap an edit of a pack it was not built with
	Rivals       []ModID      // the packs of the hit set whose edits clash with this pack's
	// author is the manifest Author, normalised, so two packs by one author can be told apart from strangers.
	Author string
	// sig is the pack's share of a target's memo key (PackSig), and stable whether its stamps can be trusted.
	Sig    string
	Stable bool
}

// Patch is one Load or EditImage/EditMap change with the HasMod conditions that gate it.
type Patch struct {
	Kind          string // "load" or "edit"
	Target        string
	FromFile      string
	Priority      string
	PatchMode     string
	When          When
	Shapes        []Shape             // what an edit writes; see editShapes
	Spouse        string              // the spouse the change requires, or ""
	Places        map[string][]string // literal values the change requires of placeTokens
	Image         bool                // an EditImage change, which only changes how something looks
	ImageDigest   string
	ImageFromArea string
	Source        string
	Index         int
	Action        string
	ToArea        string
	// tokenValue is a dynamic token's value; a token that yields a picker value only when a mod is
	// installed is how a pack says which mod that value is for.
	TokenName  string
	TokenValue string
	// extra marks a change that also does something its shapes do not show (warps, text operations,
	// appended list entries, a source map's properties), which a later edit never undoes.
	Extra bool
}

type Config struct {
	Field         string   `json:"field"`
	Values        []string `json:"values"`
	AllowMultiple bool     `json:"allowMultiple,omitempty"`
}

// When holds a change's HasMod conditions: each anyOf group needs one of its mods
// present, and no noneOf mod may be present. Dynamic token conditions are checked
// with the pack and profile context; other conditions are treated as met.
type When struct {
	AnyOf   [][]string          `json:"anyOf,omitempty"`
	NoneOf  []string            `json:"noneOf,omitempty"`
	Config  []Config            `json:"config,omitempty"`
	Dynamic []DynamicCondition  `json:"dynamic,omitempty"`
	Flags   []FlagCondition     `json:"flags,omitempty"`
	Spouse  string              `json:"spouse,omitempty"`
	Places  map[string][]string `json:"places,omitempty"`
	// Assumed are the raw keys of the conditions this scan treats as met (time, queries, events), shown with
	// the conflict so the user knows it may not happen on their save.
	Assumed []string `json:"assumed,omitempty"`
	// Conditional is set when the change, or an Include around it, has any When condition at all, including the
	// assumed ones.
	Conditional bool `json:"conditional,omitempty"`
}

type DynamicCondition struct {
	Name     string   `json:"name"`
	Values   []string `json:"values,omitempty"`
	Contains string   `json:"contains,omitempty"`
	Expected bool     `json:"expected,omitempty"`
}

type FlagCondition struct {
	Name    string `json:"name"`
	Present bool   `json:"present,omitempty"`
}

func (w When) with(o When) When {
	// Most changes name no place, so an empty map is not allocated for each.
	var places map[string][]string
	if len(w.Places)+len(o.Places) > 0 {
		places = map[string][]string{}
	}
	for key, values := range w.Places {
		places[key] = slices.Clone(values)
	}
	for key, values := range o.Places {
		places[key] = append(places[key], values...)
	}
	spouse := w.Spouse
	if spouse == "" {
		spouse = o.Spouse
	} else if o.Spouse != "" && !strings.EqualFold(spouse, o.Spouse) {
		spouse = "\x00"
	}
	return When{
		AnyOf:       append(slices.Clone(w.AnyOf), o.AnyOf...),
		NoneOf:      append(slices.Clone(w.NoneOf), o.NoneOf...),
		Config:      append(slices.Clone(w.Config), o.Config...),
		Dynamic:     append(slices.Clone(w.Dynamic), o.Dynamic...),
		Flags:       append(slices.Clone(w.Flags), o.Flags...),
		Spouse:      spouse,
		Places:      places,
		Assumed:     append(slices.Clone(w.Assumed), o.Assumed...),
		Conditional: w.Conditional || o.Conditional,
	}
}

func (w When) withDefinition(o When) When {
	merged := w.with(o)
	// A definition's flags constrain its reachability, not the patch that selects it.
	merged.Flags = slices.Clone(w.Flags)
	return merged
}

// Holds reports whether the conditions are met by present, a set of folded mod ids.
func (w When) Holds(present map[string]bool) bool {
	for _, group := range w.AnyOf {
		if !slices.ContainsFunc(group, func(id string) bool { return present[id] }) {
			return false
		}
	}
	return !slices.ContainsFunc(w.NoneOf, func(id string) bool { return present[id] })
}

type Pack struct {
	// values are the pack's config values (schema defaults, then config.json), by lower-case key;
	// used while scanning only, so not cached on disk.
	Values      map[string]string
	Fingerprint string
	Files       []FileStamp
	Patches     []Patch
	Tokens      []TokenDefinition
	Mentions    map[string]bool
	Schema      map[string]Schema
	Skips       int
	// skipImages is whether the pack's image-edit shapes were read without the pixel-overlap scan.
	SkipImages bool
	// farms maps the map asset of each custom farm the pack adds to Data/AdditionalFarms to the farm's id.
	Farms map[string]string
}

const contentPackParserVersion = 27

// absentSize stamps a file that was not there, so the cache is dropped when it appears.
const absentSize = -1

type FileStamp struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime"`
}

type diskPackCache struct {
	Version int                      `json:"version"`
	Packs   map[string]diskPackEntry `json:"packs"`
}

type diskPackEntry struct {
	Fingerprint string      `json:"fingerprint"`
	Files       []FileStamp `json:"files"`
	SkipImages  bool        `json:"skipImages,omitempty"`
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
	base    string
	entries map[string]diskPackEntry
	dirty   bool
}

// packKey names a pack folder in the disk cache: the path under the data folder when it is inside, so a moved or
// copied data folder still finds its packs, else the absolute path.
func packKey(base, root string) string {
	if base != "" && safefs.UnderRoot(base, root) {
		if rel, err := filepath.Rel(base, root); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return root
}

func packKeyRoot(base, key string) string {
	if filepath.IsAbs(key) {
		return key
	}
	return filepath.Join(base, filepath.FromSlash(key))
}

type TokenDefinition struct {
	Name  string
	Value string
	When  map[string]json.RawMessage
}

func diskPackOf(pack Pack) diskCachedPack {
	out := diskCachedPack{
		Mentions: pack.Mentions,
		Skips:    pack.Skips,
		Farms:    pack.Farms,
	}
	if pack.Patches != nil {
		out.Patches = make([]diskPatch, len(pack.Patches))
		for i, patch := range pack.Patches {
			out.Patches[i] = diskPatchOf(patch)
		}
	}
	if pack.Tokens != nil {
		out.Tokens = make([]diskTokenDefinition, len(pack.Tokens))
		for i, token := range pack.Tokens {
			out.Tokens[i] = diskTokenDefinition(token)
		}
	}
	if pack.Schema != nil {
		out.Schema = make(map[string]diskSchema, len(pack.Schema))
		for key, schema := range pack.Schema {
			out.Schema[key] = diskSchema(schema)
		}
	}
	return out
}

func cachedPackOfDisk(root string, disk diskCachedPack, entry diskPackEntry) Pack {
	out := Pack{
		Fingerprint: entry.Fingerprint,
		Files:       entry.Files,
		Mentions:    disk.Mentions,
		Schema:      make(map[string]Schema, len(disk.Schema)),
		Skips:       disk.Skips,
		SkipImages:  entry.SkipImages,
		Farms:       disk.Farms,
	}
	if disk.Patches != nil {
		out.Patches = make([]Patch, len(disk.Patches))
		for i, patch := range disk.Patches {
			out.Patches[i] = cpPatchOfDisk(root, patch)
		}
	}
	if disk.Tokens != nil {
		out.Tokens = make([]TokenDefinition, len(disk.Tokens))
		for i, token := range disk.Tokens {
			out.Tokens[i] = TokenDefinition(token)
		}
	}
	if disk.Schema != nil {
		out.Schema = make(map[string]Schema, len(disk.Schema))
		for key, schema := range disk.Schema {
			out.Schema[key] = Schema(schema)
		}
	}
	return out
}

func diskPatchOf(patch Patch) diskPatch {
	out := diskPatch{
		Kind:          patch.Kind,
		Target:        patch.Target,
		FromFile:      patch.FromFile,
		Priority:      patch.Priority,
		PatchMode:     patch.PatchMode,
		When:          diskWhenOf(patch.When),
		Image:         patch.Image,
		ImageDigest:   patch.ImageDigest,
		ImageFromArea: patch.ImageFromArea,
		Source:        patch.Source,
		Index:         patch.Index,
		Action:        patch.Action,
		ToArea:        patch.ToArea,
		TokenName:     patch.TokenName,
		TokenValue:    patch.TokenValue,
		Extra:         patch.Extra,
	}
	if patch.Shapes != nil {
		out.Shapes = make([]diskShape, len(patch.Shapes))
		for i, shape := range patch.Shapes {
			out.Shapes[i] = diskShape{
				Kind:  shape.Kind,
				X:     shape.X,
				Y:     shape.Y,
				W:     shape.W,
				H:     shape.H,
				Cells: encodeCellSet(shape.Cells),
				Layer: shape.Layer,
				Key:   shape.Key,
				Value: shape.Value,
				Tiny:  shape.Tiny,
			}
		}
	}
	return out
}

func cpPatchOfDisk(_ string, patch diskPatch) Patch {
	when := cpWhenOfDisk(patch.When)
	out := Patch{
		Kind:          patch.Kind,
		Target:        intern(patch.Target),
		FromFile:      intern(patch.FromFile),
		Priority:      intern(patch.Priority),
		PatchMode:     intern(patch.PatchMode),
		When:          when,
		Spouse:        when.Spouse,
		Places:        when.Places,
		Image:         patch.Image,
		ImageDigest:   patch.ImageDigest,
		ImageFromArea: patch.ImageFromArea,
		Source:        intern(patch.Source),
		Index:         patch.Index,
		Action:        intern(patch.Action),
		ToArea:        patch.ToArea,
		TokenName:     patch.TokenName,
		TokenValue:    patch.TokenValue,
		Extra:         patch.Extra,
	}
	if patch.Shapes != nil {
		out.Shapes = make([]Shape, len(patch.Shapes))
		for i, shape := range patch.Shapes {
			out.Shapes[i] = Shape{
				Kind:  shape.Kind,
				X:     shape.X,
				Y:     shape.Y,
				W:     shape.W,
				H:     shape.H,
				Cells: decodeCellSet(shape.Cells),
				Layer: intern(shape.Layer),
				Key:   intern(shape.Key),
				Value: intern(shape.Value),
				Tiny:  shape.Tiny,
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

func diskWhenOf(when When) diskWhen {
	out := diskWhen{
		AnyOf:   when.AnyOf,
		NoneOf:  when.NoneOf,
		Spouse:  when.Spouse,
		Places:  when.Places,
		Assumed: when.Assumed,
		Cond:    when.Conditional,
	}
	if when.Config != nil {
		out.Config = make([]diskConfig, len(when.Config))
		for i, config := range when.Config {
			out.Config[i] = diskConfig(config)
		}
	}
	if when.Dynamic != nil {
		out.Dynamic = make([]diskDynamicCondition, len(when.Dynamic))
		for i, condition := range when.Dynamic {
			out.Dynamic[i] = diskDynamicCondition(condition)
		}
	}
	if when.Flags != nil {
		out.Flags = make([]diskFlagCondition, len(when.Flags))
		for i, flag := range when.Flags {
			out.Flags[i] = diskFlagCondition(flag)
		}
	}
	return out
}

func cpWhenOfDisk(when diskWhen) When {
	out := When{
		AnyOf:       when.AnyOf,
		NoneOf:      when.NoneOf,
		Spouse:      when.Spouse,
		Places:      when.Places,
		Assumed:     when.Assumed,
		Conditional: when.Cond,
	}
	if when.Config != nil {
		out.Config = make([]Config, len(when.Config))
		for i, config := range when.Config {
			out.Config[i] = Config(config)
		}
	}
	if when.Dynamic != nil {
		out.Dynamic = make([]DynamicCondition, len(when.Dynamic))
		for i, condition := range when.Dynamic {
			out.Dynamic[i] = DynamicCondition(condition)
		}
	}
	if when.Flags != nil {
		out.Flags = make([]FlagCondition, len(when.Flags))
		for i, flag := range when.Flags {
			out.Flags[i] = FlagCondition(flag)
		}
	}
	return out
}

var packCache sync.Map // folder path -> Pack

// packValidated holds the folders whose cached pack was checked against its files during the running Check, so a
// pack read many times in one Check is stat'ed once. Check clears it at both ends, never mid-check.
var packValidated sync.Map

func ReadPack(im Mod, skip bool) Pack {
	return readContentPackWithEnabled(im, true, skip)
}

func ReadPackForCleanup(im Mod, skip bool) Pack {
	return readContentPackWithEnabled(im, false, skip)
}

func readContentPackWithEnabled(im Mod, requireEnabled, skip bool) Pack {
	if (requireEnabled && !im.Enabled) || im.Folder == "" || !IsContentPatcherPack(im) {
		return Pack{}
	}
	root := filepath.Clean(im.Folder)
	if c, ok := packCache.Load(root); ok {
		if _, fresh := packValidated.Load(root); fresh {
			if got, ok := c.(Pack); ok && got.SkipImages == skip {
				return got
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "content.json")); err != nil {
		return Pack{}
	}
	if c, ok := packCache.Load(root); ok {
		got, ok := c.(Pack)
		if ok && got.SkipImages == skip && packFingerprintValid(root, got.Files, got.Fingerprint) {
			packValidated.Store(root, true)
			return got
		}
	}
	cachePath := loadPackDiskCache()
	if cachePath != "" {
		if entry, ok := diskPackEntryFor(root); ok && entry.SkipImages == skip && packFingerprintValid(root, entry.Files, entry.Fingerprint) {
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
	pack := Pack{Mentions: map[string]bool{}, SkipImages: skip}
	pack.recordPackFile(root, filepath.Join(root, "content.json"))
	pack.recordPackFile(root, filepath.Join(root, "config.json"))
	pack.Schema = readConfigSchema(root)
	pack.Values = packConfigValues(pack.Schema, root)
	scanContentFile(root, "content.json", map[string]bool{}, When{}, &pack)
	slices.SortFunc(pack.Files, func(a, b FileStamp) int {
		return strings.Compare(a.Path, b.Path)
	})
	// Appending left spare capacity, up to the slice's own size again.
	pack.Patches = slices.Clip(pack.Patches)
	pack.Fingerprint = packFilesFingerprint(pack.Files)
	if !packFingerprintValid(root, pack.Files, pack.Fingerprint) {
		noteUnstampable()
		dropPNGAlphaUnder(root)
		return pack
	}
	packCache.Store(root, pack)
	packValidated.Store(root, true)
	if cachePath != "" {
		if disk, err := encodeDiskPack(diskPackOf(pack)); err == nil {
			storeDiskPackEntry(root, diskPackEntry{Fingerprint: pack.Fingerprint, Files: slices.Clone(pack.Files), SkipImages: skip, Pack: disk})
		}
	}
	dropPNGAlphaUnder(root)
	return pack
}

func (p *Pack) recordPackFile(root, abs string) {
	if !safefs.UnderRoot(root, abs) {
		return
	}
	relative, err := filepath.Rel(root, abs)
	if err != nil {
		return
	}
	stamp := FileStamp{Path: filepath.ToSlash(relative), Size: absentSize}
	info, err := os.Stat(abs)
	switch {
	case err == nil:
		stamp.Size, stamp.ModTime = info.Size(), info.ModTime().UnixNano()
	case !errors.Is(err, fs.ErrNotExist):
		return
	}
	// The first stamp is the one taken before the file was read.
	if slices.ContainsFunc(p.Files, func(f FileStamp) bool { return f.Path == stamp.Path }) {
		return
	}
	p.Files = append(p.Files, stamp)
}

func packFilesFingerprint(files []FileStamp) string {
	raw, _ := json.Marshal(files)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func packFingerprintValid(root string, files []FileStamp, fingerprint string) bool {
	if len(files) == 0 || fingerprint == "" {
		return false
	}
	current := make([]FileStamp, len(files))
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

// writePackedCache writes v as gzipped JSON to path, reporting whether it did.
func writePackedCache(path string, v any) bool {
	raw, err := json.Marshal(v)
	if err != nil {
		return false
	}
	packed, err := zipPackCache(raw)
	return err == nil && os.MkdirAll(filepath.Dir(path), 0o700) == nil && safefs.WriteFileAtomic(path, packed, 0o600) == nil
}

func unzipPackCache(raw []byte) ([]byte, bool) {
	if len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		reader, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, false
		}
		payload, err := safefs.ReadCapped(reader, maxPackFileBytes)
		_ = reader.Close()
		return payload, err == nil
	}
	return raw, true
}

func loadPackDiskCache() string {
	if cacheRoot == "" {
		return ""
	}
	base := cacheRoot
	path := filepath.Join(base, "cache", "problems-content-packs.json")
	packDiskState.Lock()
	defer packDiskState.Unlock()
	if packDiskState.loaded && packDiskState.path == path {
		return path
	}
	packDiskState.loaded = true
	packDiskState.path = path
	packDiskState.base = base
	packDiskState.entries = map[string]diskPackEntry{}
	packDiskState.dirty = false
	f, err := safefs.Open(path)
	if err != nil {
		return path
	}
	defer func() { _ = f.Close() }()
	if cache, ok := readPackCache(bufio.NewReader(f)); ok {
		packDiskState.entries = cache.Packs
	}
	return path
}

type packCacheHeader struct {
	Version int
	Count   int
}

type packCacheRecord struct {
	Root  string
	Entry diskPackEntry
}

// The pack cache is a gob stream, not JSON: it holds every pack's cell sets, tens of MB that a start decodes in full,
// and gob reads them several times faster. One value per pack lets a save and a load hold a single pack at a time. It
// is left uncompressed for the same reason.
func writePackCache(w io.Writer, cache diskPackCache) error {
	enc := gob.NewEncoder(w)
	if err := enc.Encode(packCacheHeader{Version: cache.Version, Count: len(cache.Packs)}); err != nil {
		return err
	}
	for root, entry := range cache.Packs {
		if err := enc.Encode(packCacheRecord{Root: root, Entry: entry}); err != nil {
			return err
		}
	}
	return nil
}

// readPackCache reads a cache this parser version wrote; anything else reads as no cache.
func readPackCache(r io.Reader) (diskPackCache, bool) {
	dec := gob.NewDecoder(r)
	var head packCacheHeader
	if dec.Decode(&head) != nil || head.Version != contentPackParserVersion || head.Count < 0 {
		return diskPackCache{}, false
	}
	packs := make(map[string]diskPackEntry, min(head.Count, 4096))
	for range head.Count {
		var rec packCacheRecord
		if dec.Decode(&rec) != nil {
			return diskPackCache{}, false
		}
		packs[rec.Root] = rec.Entry
	}
	return diskPackCache{Version: head.Version, Packs: packs}, true
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
	entry, ok := packDiskState.entries[packKey(packDiskState.base, root)]
	return entry, ok
}

func storeDiskPackEntry(root string, entry diskPackEntry) {
	packDiskState.Lock()
	defer packDiskState.Unlock()
	if packDiskState.entries == nil {
		packDiskState.entries = map[string]diskPackEntry{}
	}
	packDiskState.entries[packKey(packDiskState.base, root)] = entry
	packDiskState.dirty = true
}

func FlushPackDiskCache(mods []Mod) {
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
	for key := range packDiskState.entries {
		if _, err := os.Stat(packKeyRoot(packDiskState.base, key)); err != nil {
			delete(packDiskState.entries, key)
			packDiskState.dirty = true
		}
	}
	if !packDiskState.dirty {
		packDiskState.Unlock()
		return
	}
	entries := maps.Clone(packDiskState.entries)
	packDiskState.dirty = false
	packDiskState.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		packDiskState.Lock()
		packDiskState.dirty = true
		packDiskState.Unlock()
		return
	}
	err := safefs.WriteStreamAtomic(path, 0o600, func(w io.Writer) error {
		bw := bufio.NewWriter(w)
		if err := writePackCache(bw, diskPackCache{Version: contentPackParserVersion, Packs: entries}); err != nil {
			return err
		}
		return bw.Flush()
	})
	if err != nil {
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

type Schema struct {
	Key           string
	DefaultValue  string
	AllowMultiple bool
	AllowValues   []string
	AllowBlank    bool
	Description   string
}

type configSchemaEntry struct {
	Default       json.RawMessage `json:"Default"`
	AllowValues   string          `json:"AllowValues"`
	AllowMultiple bool            `json:"AllowMultiple"`
	AllowBlank    bool            `json:"AllowBlank"`
	Description   string          `json:"Description"`
}

func configSchemaEntries(contentJSON []byte) (map[string]configSchemaEntry, error) {
	var doc struct {
		ConfigSchema map[string]json.RawMessage `json:"ConfigSchema"`
	}
	if err := json.Unmarshal(contentJSON, &doc); err != nil {
		return nil, fmt.Errorf("content.json: %w", err)
	}
	out := make(map[string]configSchemaEntry, len(doc.ConfigSchema))
	for name, raw := range doc.ConfigSchema {
		var e configSchemaEntry
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		out[name] = e
	}
	return out, nil
}

func readConfigSchema(root string) map[string]Schema {
	raw, err := safefs.ReadFile(filepath.Join(root, "content.json"))
	if err != nil {
		return nil
	}
	entries, err := configSchemaEntries(lenientjson.Clean(raw))
	if err != nil {
		return nil
	}
	out := make(map[string]Schema, len(entries))
	for key, entry := range entries {
		defaultValue, _ := ScalarValue(entry.Default)
		key = strings.TrimSpace(key)
		if key == "" || hasToken(key) {
			continue
		}
		out[strings.ToLower(key)] = Schema{
			Key:           key,
			DefaultValue:  defaultValue,
			AllowMultiple: entry.AllowMultiple,
			AllowValues:   SplitTargets(entry.AllowValues),
			AllowBlank:    entry.AllowBlank,
			Description:   entry.Description,
		}
	}
	return out
}

func IsContentPatcherPack(im Mod) bool {
	return Equal(im.ContentPackFor, contentPatcherID)
}

// intern shares one copy of a string a pack repeats on every change (its file names, targets and actions), so
// parsed changes do not each hold their own.
func intern(s string) string { return unique.Make(s).Value() }

type contentFileDoc struct {
	Changes       []json.RawMessage `json:"Changes"`
	DynamicTokens []dynamicTokenDoc `json:"DynamicTokens"`
}

type dynamicTokenDoc struct {
	Name  string                     `json:"Name"`
	Value json.RawMessage            `json:"Value"`
	When  map[string]json.RawMessage `json:"When"`
}

func scanContentFile(root, rel string, seen map[string]bool, outer When, pack *Pack) {
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
	raw, err := safefs.ReadFile(abs)
	if err != nil {
		return
	}
	var doc contentFileDoc
	if err := json.Unmarshal(lenientjson.Clean(raw), &doc); err != nil {
		return
	}
	// A dynamic token's value gates content like a change does, so its conditions count for compatibility
	// settings too. Content Patcher reads DynamicTokens only from content.json.
	if key == "content.json" {
		scanDynamicTokens(doc.DynamicTokens, outer, pack)
	}
	warmOverlayImages(root, rel, doc.Changes, pack.SkipImages)
	var dynamicValues map[string]string
	for i, rawChange := range doc.Changes {
		var ch contentChange
		if json.Unmarshal(rawChange, &ch) != nil {
			continue
		}
		action := strings.TrimSpace(ch.Action)
		withConfigValues(ch.When, pack.Values)
		when := outer.with(parseWhenWithTokens(ch.When, pack.Mentions, pack.Schema, pack.Tokens))
		when.Conditional = when.Conditional || len(ch.When) > 0
		noteAssetOwners(ch.Target, pack.Mentions)
		kind, action, shapes, included := classifyChange(root, rel, &ch, action, when, seen, pack)
		if included {
			continue
		}
		// Changes that cannot conflict still count for compatibility settings, whatever their target.
		if kind == "other" {
			pack.Patches = append(pack.Patches, Patch{Kind: kind, When: when, Source: intern(rel), Index: i, Action: intern(action)})
			continue
		}
		appendChangePatches(root, rel, i, rawChange, &ch, kind, action, shapes, when, &dynamicValues, pack)
	}
}

func scanDynamicTokens(tokens []dynamicTokenDoc, outer When, pack *Pack) {
	for _, tok := range tokens {
		withConfigValues(tok.When, pack.Values)
	}
	for _, tok := range tokens {
		value, ok := ScalarValue(tok.Value)
		if !ok {
			continue
		}
		pack.Tokens = append(pack.Tokens, TokenDefinition{Name: tokenName(tok.Name), Value: value, When: tok.When})
	}
	for _, tok := range tokens {
		value, ok := ScalarValue(tok.Value)
		if !ok {
			continue
		}
		pack.Patches = append(pack.Patches, Patch{Kind: "other", When: outer.with(parseWhen(tok.When, pack.Mentions, pack.Schema)), TokenName: strings.ToLower(strings.TrimSpace(tok.Name)), TokenValue: value})
	}
}

// classifyChange sorts one change into a patch kind and canonical action. An Include is followed
// recursively and reported as included, so the caller records nothing for it.
func classifyChange(root, rel string, ch *contentChange, action string, when When, seen map[string]bool, pack *Pack) (kind, canonAction string, shapes []Shape, included bool) {
	switch {
	case strings.EqualFold(action, "Include"):
		for _, from := range SplitTargets(ch.FromFile) {
			from = contentSourceReference(root, rel, from)
			if hasToken(from) {
				pack.Skips++
				continue
			}
			scanContentFile(root, from, seen, when, pack)
		}
		return "", action, nil, true
	case strings.EqualFold(action, kindLoad):
		ch.FromFile = contentSourceReference(root, rel, ch.FromFile)
		return "load", kindLoad, nil, false
	case strings.EqualFold(action, kindEditImage), strings.EqualFold(action, kindEditMap):
		canonAction = kindEditMap
		if strings.EqualFold(action, kindEditImage) {
			canonAction = kindEditImage
		}
		ch.FromFile = contentSourceReference(root, rel, ch.FromFile)
		recordReferencedPackFiles(root, ch.FromFile, canonAction == kindEditImage, pack)
		shapes = editShapes(root, *ch, canonAction == kindEditImage, pack.SkipImages)
		return editOrOther(shapes), canonAction, shapes, false
	case strings.EqualFold(action, kindEditData):
		if NormalizeTarget(ch.Target) == "data/additionalfarms" {
			pack.recordFarms(ch.Entries)
		}
		shapes = dataShapes(root, *ch, pack.Values)
		return editOrOther(shapes), kindEditData, shapes, false
	}
	return "other", action, nil, false
}

func editOrOther(shapes []Shape) string {
	if len(shapes) == 0 {
		return "other"
	}
	return "edit"
}

func appendChangePatches(root, rel string, i int, rawChange json.RawMessage, ch *contentChange, kind, action string, shapes []Shape, when When, dynamicValues *map[string]string, pack *Pack) {
	fromArea := string(lenientjson.Clean(ch.FromArea))
	toArea := string(lenientjson.Clean(ch.ToArea))
	priority, _ := ScalarValue(ch.Priority)
	extra := changeDoesMore(rawChange, *ch)
	for _, t := range SplitTargets(ch.Target) {
		targets := []tokenTarget{{Target: t, FromFile: ch.FromFile, When: when}}
		if hasToken(t) {
			if *dynamicValues == nil {
				*dynamicValues = singleDynamicValues(pack)
			}
			if targets = expandTargetTokens(t, ch.FromFile, when, pack.Values, *dynamicValues); targets == nil {
				pack.Skips++
				continue
			}
		}
		for _, tt := range targets {
			pack.Patches = append(pack.Patches, Patch{
				Kind: kind, Target: intern(NormalizeTarget(tt.Target)), FromFile: intern(tt.FromFile), Priority: intern(strings.TrimSpace(priority)),
				PatchMode: intern(strings.TrimSpace(ch.PatchMode)), When: tt.When,
				Shapes: shapes, Spouse: tt.When.Spouse, Places: tt.When.Places, Image: action == kindEditImage,
				ImageDigest:   imageFileDigest(root, tt.FromFile, action == kindEditImage),
				ImageFromArea: intern(fromArea),
				Source:        intern(rel), Index: i, Action: intern(action), ToArea: intern(toArea),
				Extra: extra,
			})
		}
	}
}

type tokenTarget struct {
	Target   string
	FromFile string
	When     When
}

var (
	plainToken  = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.]+)\s*\}\}`)
	seasonToken = regexp.MustCompile(`(?i)\{\{\s*season\s*\}\}`)
	seasons     = []string{"spring", "summer", "fall", "winter"}
)

// expandTargetTokens resolves a tokenized target with the pack's config values and its dynamic tokens that
// have one value under that config. {{Season}} becomes one target per season, each gated on that season.
// It returns nil when a token stays unknown.
func expandTargetTokens(target, fromFile string, when When, config, dynamic map[string]string) []tokenTarget {
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
		if limited := when.Places["season"]; len(limited) > 0 {
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
			w = when.with(When{})
			if w.Places == nil {
				w.Places = map[string][]string{}
			}
			w.Places["season"] = []string{season}
		}
		out = append(out, tokenTarget{Target: t, FromFile: from, When: w})
	}
	return out
}

// singleDynamicValues are the pack's dynamic tokens that hold exactly one value under its config. A
// definition whose When needs more than config fields may or may not apply, so its value stays possible.
func singleDynamicValues(pack *Pack) map[string]string {
	possible := map[string][]string{}
	unknown := map[string]bool{}
	for _, definition := range pack.Tokens {
		if hasToken(definition.Value) {
			unknown[definition.Name] = true
			continue
		}
		switch configOnlyState(definition.When, pack) {
		case cpConditionTrue:
			possible[definition.Name] = []string{definition.Value}
		case cpConditionUnknown:
			if !slices.Contains(possible[definition.Name], definition.Value) {
				possible[definition.Name] = append(possible[definition.Name], definition.Value)
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

func configOnlyState(raw map[string]json.RawMessage, pack *Pack) cpConditionState {
	if len(raw) == 0 {
		return cpConditionTrue
	}
	w := parseWhen(raw, map[string]bool{}, pack.Schema)
	if len(w.Config) != len(raw) || len(w.AnyOf) > 0 || len(w.NoneOf) > 0 || len(w.Dynamic) > 0 || len(w.Flags) > 0 {
		return cpConditionUnknown
	}
	if ConfigHolds(w.Config, pack.Schema, pack.Values) {
		return cpConditionTrue
	}
	return cpConditionFalse
}

func recordReferencedPackFiles(root, rel string, image bool, pack *Pack) {
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

func parseWhen(raw map[string]json.RawMessage, mentions map[string]bool, schema map[string]Schema) When {
	return parseWhenWithTokens(raw, mentions, schema, nil)
}

func parseWhenWithTokens(raw map[string]json.RawMessage, mentions map[string]bool, schema map[string]Schema, tokens []TokenDefinition) When {
	return parseWhenDepth(raw, mentions, schema, tokens, 0)
}

func parseWhenDepth(raw map[string]json.RawMessage, mentions map[string]bool, schema map[string]Schema, tokens []TokenDefinition, depth int) When {
	var w When
	for k, v := range raw {
		addWhenCondition(&w, k, v, mentions, schema, tokens, depth)
	}
	return w.with(When{Spouse: spouseOf(raw, tokens), Places: placesOf(raw)})
}

// addWhenCondition folds one When entry into w; entries the scan cannot read are skipped or assumed met.
func addWhenCondition(w *When, k string, v json.RawMessage, mentions map[string]bool, schema map[string]Schema, tokens []TokenDefinition, depth int) {
	if hasToken(k) {
		if merged, ok := dynamicTokenWhen(k, v, tokens, mentions, schema, depth); ok {
			*w = w.withDefinition(merged)
		} else {
			w.Assumed = appendAssumed(w.Assumed, k, v, tokens)
		}
		return
	}
	name, arg, _ := strings.Cut(k, "|")
	name = tokenName(name)
	if merged, ok := dynamicTokenWhen(k, v, tokens, mentions, schema, depth); ok {
		*w = w.withDefinition(merged)
		return
	}
	if strings.EqualFold(name, "hasmod") {
		addHasMod(w, strings.TrimSpace(arg), v, mentions)
		return
	}
	if flags, ok := flagConditions(k, v); ok {
		if len(flags) == 0 {
			w.Assumed = appendAssumed(w.Assumed, k, v, tokens)
		}
		w.Flags = append(w.Flags, flags...)
		return
	}
	field, ok := schema[strings.ToLower(name)]
	if !ok {
		w.Assumed = appendAssumed(w.Assumed, k, v, tokens)
		return
	}
	var values []string
	if !condValues(v, &values) {
		return
	}
	if arg = strings.TrimSpace(arg); arg != "" {
		var ok bool
		if values, ok = containsValues(field, arg, values); !ok {
			return
		}
	}
	w.Config = append(w.Config, Config{Field: field.Key, Values: values, AllowMultiple: field.AllowMultiple})
}

func addHasMod(w *When, arg string, v json.RawMessage, mentions map[string]bool) {
	if arg == "" {
		var list string
		if json.Unmarshal(v, &list) != nil || hasToken(list) {
			return
		}
		if ids := SplitTargets(list); len(ids) > 0 {
			w.AnyOf = append(w.AnyOf, note(ids, mentions))
		}
		return
	}
	param, list, ok := strings.Cut(arg, "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) {
		return
	}
	ids := note(SplitTargets(list), mentions)
	var flags []string
	if !condValues(v, &flags) || len(flags) != 1 {
		return
	}
	switch strings.ToLower(flags[0]) {
	case "true":
		w.AnyOf = append(w.AnyOf, ids)
	case "false":
		w.NoneOf = append(w.NoneOf, ids...)
	}
}

// appendAssumed records a condition this scan treats as met, unless it is a spouse or place condition the
// summary already names.
func appendAssumed(assumed []string, key string, raw json.RawMessage, tokens []TokenDefinition) []string {
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

func dynamicTokenWhen(key string, raw json.RawMessage, tokens []TokenDefinition, mentions map[string]bool, schema map[string]Schema, depth int) (When, bool) {
	condition, ok := dynamicTokenCondition(key, raw)
	if !ok || !slices.ContainsFunc(tokens, func(definition TokenDefinition) bool {
		return definition.Name == condition.Name
	}) {
		return When{}, false
	}
	out := When{Dynamic: []DynamicCondition{condition}}
	if condition.Contains != "" && !condition.Expected {
		return out, true
	}
	match := -1
	for i, definition := range tokens {
		if definition.Name != condition.Name || !dynamicConditionMatchesValue(condition, definition.Value) {
			continue
		}
		if match >= 0 {
			match = -1
			break
		}
		match = i
	}
	if match >= 0 && depth < 8 {
		out = out.withDefinition(parseWhenDepth(tokens[match].When, mentions, schema, tokens, depth+1))
	}
	return out, true
}

func dynamicTokenCondition(key string, raw json.RawMessage) (DynamicCondition, bool) {
	name, arg, ok := dynamicTokenConditionParts(key)
	if !ok {
		return DynamicCondition{}, false
	}
	if arg == "" {
		var values []string
		if !condValues(raw, &values) {
			return DynamicCondition{}, false
		}
		return DynamicCondition{Name: name, Values: values, Expected: true}, true
	}
	param, list, ok := strings.Cut(arg, "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) {
		return DynamicCondition{}, false
	}
	var flags []string
	if !condValues(raw, &flags) || len(flags) != 1 {
		return DynamicCondition{}, false
	}
	expected, err := strconv.ParseBool(strings.TrimSpace(flags[0]))
	if err != nil {
		return DynamicCondition{}, false
	}
	return DynamicCondition{Name: name, Contains: strings.TrimSpace(list), Expected: expected}, true
}

func dynamicConditionMatchesValue(condition DynamicCondition, value string) bool {
	if condition.Contains != "" {
		return condition.Expected && strings.EqualFold(strings.TrimSpace(condition.Contains), strings.TrimSpace(value))
	}
	return slices.ContainsFunc(condition.Values, func(want string) bool {
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
	Values map[string]bool
	Known  bool
}

func reachableDynamicTokens(tokens []TokenDefinition, present map[string]bool, schema map[string]Schema, config map[string]string, flags []FlagCondition) map[string]cpTokenReachability {
	reachable := map[string]cpTokenReachability{}
	for _, definition := range tokens {
		if definition.Name == "" {
			continue
		}
		state := dynamicWhenState(definition.When, tokens, reachable, present, schema, config, flags)
		current, exists := reachable[definition.Name]
		if !exists {
			current.Known = true
			current.Values = map[string]bool{}
		}
		switch state {
		case cpConditionTrue:
			current.Values = map[string]bool{}
			current.Known = true
		case cpConditionUnknown:
		case cpConditionFalse:
			reachable[definition.Name] = current
			continue
		}
		if hasToken(definition.Value) {
			current.Known = false
			reachable[definition.Name] = current
			continue
		}
		current.Values[strings.ToLower(strings.TrimSpace(definition.Value))] = true
		reachable[definition.Name] = current
	}
	return reachable
}

func dynamicWhenState(raw map[string]json.RawMessage, tokens []TokenDefinition, reachable map[string]cpTokenReachability, present map[string]bool, schema map[string]Schema, config map[string]string, flags []FlagCondition) cpConditionState {
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

func flagConditionState(flagValues, assumptions []FlagCondition) cpConditionState {
	state := cpConditionTrue
	for _, flag := range flagValues {
		current := cpConditionUnknown
		for _, assumption := range assumptions {
			if assumption.Name != flag.Name {
				continue
			}
			if assumption.Present != flag.Present {
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

func dynamicConditionState(key string, raw json.RawMessage, tokens []TokenDefinition, reachable map[string]cpTokenReachability, present map[string]bool, schema map[string]Schema, config map[string]string, flags []FlagCondition) cpConditionState {
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
		return flagConditionState(flagValues, flags)
	}
	if tokenDefinitionExists(tokens, name) {
		reachableToken, computed := reachable[name]
		if !computed || !reachableToken.Known {
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
		if ConfigHolds([]Config{condition}, schema, config) {
			return cpConditionTrue
		}
		return cpConditionFalse
	}
	return cpConditionUnknown
}

func dynamicConditionStateForReachable(condition DynamicCondition, reachable cpTokenReachability) cpConditionState {
	if !reachable.Known {
		return cpConditionUnknown
	}
	if len(reachable.Values) == 0 {
		return cpConditionFalse
	}
	matched := 0
	if condition.Contains != "" {
		has := reachable.Values[strings.ToLower(strings.TrimSpace(condition.Contains))]
		if condition.Expected {
			if !has {
				return cpConditionFalse
			}
			for value := range reachable.Values {
				if value != strings.ToLower(strings.TrimSpace(condition.Contains)) {
					return cpConditionUnknown
				}
			}
			return cpConditionTrue
		}
		if has {
			matched = len(reachable.Values) - 1
		} else {
			matched = len(reachable.Values)
		}
		if matched == 0 {
			return cpConditionFalse
		}
		if matched == len(reachable.Values) {
			return cpConditionTrue
		}
		return cpConditionUnknown
	}
	for value := range reachable.Values {
		if slices.ContainsFunc(condition.Values, func(want string) bool {
			return strings.EqualFold(strings.TrimSpace(want), value)
		}) {
			matched++
		}
	}
	if matched == 0 {
		return cpConditionFalse
	}
	if matched == len(reachable.Values) {
		return cpConditionTrue
	}
	return cpConditionUnknown
}

type reachableKey struct {
	Tokens  *TokenDefinition
	Present uintptr
	Flags   string
}

// reachableMemo holds reachableDynamicTokens per pack, present set and flag set for the running Check; a pack's
// config is fixed for the Check, so it is not part of the key.
var reachableMemo = struct {
	sync.Mutex
	byKey map[reachableKey]map[string]cpTokenReachability
}{byKey: map[reachableKey]map[string]cpTokenReachability{}}

func cachedReachableTokens(tokens []TokenDefinition, present map[string]bool, schema map[string]Schema, config map[string]string, flags []FlagCondition) map[string]cpTokenReachability {
	if len(tokens) == 0 {
		return reachableDynamicTokens(tokens, present, schema, config, flags)
	}
	key := reachableKey{Tokens: &tokens[0], Present: reflect.ValueOf(present).Pointer(), Flags: fmt.Sprint(flags)}
	reachableMemo.Lock()
	defer reachableMemo.Unlock()
	reachable, ok := reachableMemo.byKey[key]
	if !ok {
		reachable = reachableDynamicTokens(tokens, present, schema, config, flags)
		reachableMemo.byKey[key] = reachable
	}
	return reachable
}

func DynamicWhenHolds(when When, tokens []TokenDefinition, present map[string]bool, schema map[string]Schema, config map[string]string) bool {
	if len(when.Dynamic) == 0 {
		return true
	}
	reachable := cachedReachableTokens(tokens, present, schema, config, when.Flags)
	for _, condition := range when.Dynamic {
		state, ok := reachable[condition.Name]
		if !ok || !state.Known {
			continue
		}
		if len(state.Values) == 0 {
			return false
		}
		if condition.Contains != "" {
			has := state.Values[strings.ToLower(strings.TrimSpace(condition.Contains))]
			if condition.Expected && !has {
				return false
			}
			if !condition.Expected && has && len(state.Values) == 1 {
				return false
			}
			continue
		}
		if !slices.ContainsFunc(condition.Values, func(value string) bool {
			return state.Values[strings.ToLower(strings.TrimSpace(value))]
		}) {
			return false
		}
	}
	return true
}

func tokenDefinitionExists(tokens []TokenDefinition, name string) bool {
	return slices.ContainsFunc(tokens, func(definition TokenDefinition) bool {
		return definition.Name == name
	})
}

func hasModConditionState(arg string, raw json.RawMessage, present map[string]bool) cpConditionState {
	if strings.TrimSpace(arg) == "" {
		var list string
		if json.Unmarshal(raw, &list) != nil || hasToken(list) {
			return cpConditionUnknown
		}
		ids := SplitTargets(list)
		if len(ids) == 0 {
			return cpConditionUnknown
		}
		for _, id := range ids {
			if present[SMAPI(id).Fold()] {
				return cpConditionTrue
			}
		}
		return cpConditionFalse
	}
	param, list, ok := strings.Cut(arg, "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) {
		return cpConditionUnknown
	}
	ids := SplitTargets(list)
	var values []string
	if len(ids) == 0 || !condValues(raw, &values) || len(values) != 1 {
		return cpConditionUnknown
	}
	found := slices.ContainsFunc(ids, func(id string) bool { return present[SMAPI(id).Fold()] })
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

// containsValues applies a "Field |contains=A, B" argument to a condition's one true or false value: true accepts A or
// B, false every other value the field allows. It fails on any other argument or when no value is left.
func containsValues(field Schema, arg string, values []string) ([]string, bool) {
	param, list, ok := strings.Cut(arg, "=")
	if !ok || !strings.EqualFold(strings.TrimSpace(param), "contains") || hasToken(list) || len(values) != 1 {
		return nil, false
	}
	named := SplitTargets(list)
	switch strings.ToLower(values[0]) {
	case "true":
		values = named
	case "false":
		values = slices.DeleteFunc(slices.Clone(field.AllowValues), func(a string) bool {
			return slices.ContainsFunc(named, func(n string) bool { return strings.EqualFold(n, a) })
		})
	default:
		return nil, false
	}
	return values, len(values) > 0
}

func configCondition(field Schema, arg string, raw json.RawMessage) (Config, bool) {
	var values []string
	if !condValues(raw, &values) {
		return Config{}, false
	}
	if strings.TrimSpace(arg) == "" {
		return Config{Field: field.Key, Values: values, AllowMultiple: field.AllowMultiple}, true
	}
	values, ok := containsValues(field, arg, values)
	if !ok {
		return Config{}, false
	}
	return Config{Field: field.Key, Values: values, AllowMultiple: field.AllowMultiple}, true
}

func flagConditions(key string, raw json.RawMessage) ([]FlagCondition, bool) {
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
		out := make([]FlagCondition, 0, len(values))
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" || hasToken(value) {
				return nil, true
			}
			out = append(out, FlagCondition{Name: strings.ToLower(value), Present: true})
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
	return []FlagCondition{{Name: strings.ToLower(strings.TrimSpace(list)), Present: present}}, true
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
			value, ok := ScalarValue(item)
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
	*out = SplitTargets(str)
	return len(*out) > 0
}

func ScalarValue(v json.RawMessage) (string, bool) {
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

// noteAssetOwners records the mods whose own assets a change targets ("Author.Mod/Data"): changing a mod's asset uses
// that mod. Built-in assets ("Data/Objects") have no dot before the first slash.
func noteAssetOwners(target string, mentions map[string]bool) {
	for _, t := range SplitTargets(target) {
		owner, _, found := strings.Cut(strings.TrimSpace(t), "/")
		if found && strings.Contains(owner, ".") && !hasToken(owner) {
			note([]string{owner}, mentions)
		}
	}
}

func note(ids []string, mentions map[string]bool) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = SMAPI(id).Fold()
		mentions[out[i]] = true
	}
	return out
}

type contentChange struct {
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

func SplitTargets(s string) []string {
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

func NormalizeTarget(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\\", "/")
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}
	return strings.ToLower(s)
}

func inside(root, rel string) (string, bool) {
	local := filepath.FromSlash(rel)
	if !filepath.IsLocal(local) {
		return "", false
	}
	return filepath.Join(root, local), true
}

func preloadContentPacks(mods []Mod, skip bool) {
	PreloadPacks(mods, func(m Mod) Pack { return ReadPack(m, skip) })
}

// PreloadPacks parses packs on every core, so the serial passes after it find them cached.
func PreloadPacks(mods []Mod, read func(Mod) Pack) {
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
	pngAlphaCache.Range(func(k, _ any) bool {
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

// hitAt is one patch kind, target and id a pack touches.
type hitAt struct{ kind, target, id string }

// hitCollector gathers, per patch kind and target, the packs that touch it.
type hitCollector struct {
	At       map[string]map[string][]packHit
	Index    map[hitAt]int
	Authored []authoredPack
}

// assetConflictScan also returns the packs whose every change later packs overwrite (see shadowedPacks).
func assetConflictScan(mods []Mod, run *PartsRun, skip bool) ([]Conflict, []Hint, []Shadowed, map[string]ModRef) {
	defer dropCheckScratch()
	preloadContentPacks(mods, skip)
	present := map[string]bool{}
	for _, im := range mods {
		if im.Enabled {
			present[im.UniqueID.Fold()] = true
		}
	}
	hc := &hitCollector{At: map[string]map[string][]packHit{"load": {}, "edit": {}}, Index: map[hitAt]int{}}
	for _, im := range mods {
		hc.addMod(im, ReadPack(im, skip), present)
	}
	at := hc.At
	activateEdits(at["edit"])
	shadowed := shadowedPacks(mods, at, skip)
	bundled := bundles(hc.Authored)
	shadowed = addBundledRedundant(mods, shadowed, bundled)
	customFarms := map[string]string{}
	for _, im := range mods {
		for target, id := range ReadPack(im, skip).Farms {
			customFarms[target] = modIDToken.ReplaceAllLiteralString(id, im.UniqueID.Local())
		}
	}
	out, settings := targetConflicts(at, customFarms, bundled, run, skip)
	return out, settings, shadowed, bundled
}

func (hc *hitCollector) addMod(im Mod, pack Pack, present map[string]bool) {
	knows := maps.Clone(pack.Mentions)
	dependencies := map[string]bool{}
	loadAfter := map[string]bool{}
	for _, id := range im.LoadAfter {
		loadAfter[id.Fold()] = true
	}
	for _, d := range im.Dependencies {
		if knows == nil {
			knows = map[string]bool{}
		}
		id := d.UniqueID.Fold()
		if loadAfter[id] {
			continue
		}
		knows[id] = true
		dependencies[id] = true
	}
	config := map[string]string{}
	if len(pack.Schema) > 0 {
		config = ReadPackConfig(im.Folder)
	}
	seen := PresentFor(pack, present)
	sig, stable := "", false
	if len(pack.Patches) > 0 {
		sig, stable = PackSig(im, pack, seen)
	}
	id := im.UniqueID.Fold()
	own := authoredPack{Key: im.Key, ID: id, Name: im.Name, Author: normalAuthor(im.Author), Root: im.Folder, Knows: knows}
	if own.Author != "" {
		own.Patches = make([]Patch, 0, len(pack.Patches))
	}
	if stable {
		own.Sig = sig
	}
	for _, p := range pack.Patches {
		if p.Kind == "other" || !p.When.Holds(seen) || !DynamicWhenHolds(p.When, pack.Tokens, seen, pack.Schema, config) {
			continue
		}
		p = withTokenFarmTypes(p, pack.Tokens, seen, pack.Schema, config)
		i := hc.hitFor(hitAt{p.Kind, p.Target, id}, p, im, pack, knows, seen, config, dependencies, loadAfter, sig, stable)
		hits := hc.At[p.Kind][p.Target]
		if p.Kind == "edit" {
			hits[i].Eligible = append(hits[i].Eligible, p)
		}
		if !ConfigHolds(p.When.Config, pack.Schema, config) {
			if p.Kind == "edit" {
				hits[i].ConfigOff = append(hits[i].ConfigOff, len(hits[i].Eligible)-1)
			}
			continue
		}
		if p.Kind != "edit" {
			hits[i].Loads = append(hits[i].Loads, p)
		}
		if own.Author != "" {
			own.Patches = append(own.Patches, p)
		}
	}
	if own.Author != "" {
		hc.Authored = append(hc.Authored, own)
	}
}

// hitFor returns the index of im's hit for the patch's target, adding the hit on first sight or folding
// the patch into the existing one.
func (hc *hitCollector) hitFor(k hitAt, p Patch, im Mod, pack Pack, knows, seen map[string]bool, config map[string]string, dependencies, loadAfter map[string]bool, sig string, stable bool) int {
	hits := hc.At[p.Kind][p.Target]
	i, found := hc.Index[k]
	if !found {
		hits = append(hits, packHit{
			ID: im.UniqueID, Name: im.Name, Key: im.Key, Priority: p.Priority, Mentions: knows,
			Root: im.Folder, Tokens: pack.Patches, Present: seen, Schema: pack.Schema, Config: config,
			Dependencies: dependencies, LoadAfter: loadAfter, Sig: sig, Stable: stable, Author: normalAuthor(im.Author),
		})
		i = len(hits) - 1
		hc.Index[k] = i
		hc.At[p.Kind][p.Target] = hits
		return i
	}
	hits[i].Priority = strongerContentPatcherPriority(hits[i].Priority, p.Priority, p.Kind)
	// A second pack with the same id adds its patches to the first one's hit, so its share of the key too.
	if hits[i].Root != im.Folder && !strings.Contains(hits[i].Sig, sig) {
		hits[i].Sig += "\x01" + sig
		hits[i].Stable = hits[i].Stable && stable
	}
	return i
}

// activateEdits sets each hit's active edits: the eligible ones minus the config-off positions. Most hits have
// none off, and then share the slice.
func activateEdits(edits map[string][]packHit) {
	for _, hits := range edits {
		for i := range hits {
			h := &hits[i]
			if len(h.ConfigOff) == 0 {
				h.Edits = h.Eligible
				continue
			}
			h.Edits = make([]Patch, 0, len(h.Eligible)-len(h.ConfigOff))
			for j, e := range h.Eligible {
				if !slices.Contains(h.ConfigOff, j) {
					h.Edits = append(h.Edits, e)
				}
			}
			h.ConfigOff = nil
		}
	}
}

func addBundledRedundant(mods []Mod, shadowed []Shadowed, bundled map[string]ModRef) []Shadowed {
	for _, key := range slices.Sorted(maps.Keys(bundled)) {
		if slices.ContainsFunc(shadowed, func(r Shadowed) bool { return r.Key == key }) {
			continue
		}
		im := mods[slices.IndexFunc(mods, func(m Mod) bool { return m.Key == key })]
		shadowed = append(shadowed, Shadowed{Key: key, ID: im.UniqueID, Name: im.Name, By: []ModRef{bundled[key]}, Bundled: true})
	}
	// A Redundant row is offered for removal, and SMAPI skips every enabled pack that requires a removed one.
	return slices.DeleteFunc(shadowed, func(r Shadowed) bool { return requiredByEnabled(mods, r.ID) })
}

func targetConflicts(at map[string]map[string][]packHit, customFarms map[string]string, bundled map[string]ModRef, run *PartsRun, skip bool) ([]Conflict, []Hint) {
	out := []Conflict{}
	settings := []Hint{}
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
				fields = append(fields, h.Sig)
				stable = stable && h.Stable
			}
			e := run.Part(PartKey(skip, fields...), stable, func() PartEntry { return targetPart(kind, t, hits, farm, bundled) })
			if e.Conflict != nil {
				c := *e.Conflict
				c.Farms = farmIDs(c.Farms, customFarms)
				out = append(out, c)
			}
			settings = append(settings, e.Settings...)
		}
	}
	slices.SortFunc(out, func(a, b Conflict) int {
		if a.Kind != b.Kind {
			if a.Kind == "load" {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Target, b.Target)
	})
	return out, settings
}

// requiredByEnabled reports whether an enabled mod needs id: a required dependency, or the framework it is a
// content pack for.
func requiredByEnabled(mods []Mod, id ModID) bool {
	return slices.ContainsFunc(mods, func(m Mod) bool {
		if !m.Enabled {
			return false
		}
		for _, d := range m.Dependencies {
			if d.Required && Equal(d.UniqueID, id) {
				return true
			}
		}
		return false
	})
}

// targetPart is the outcome of one target that two or more packs touch: a conflict, a setting that settles it, or
// nothing.
// farm is the custom farm type whose map target is, or "".
func targetPart(kind, target string, hits []packHit, farm string, bundled map[string]ModRef) PartEntry {
	hits = withoutBundled(kind, hits, bundled)
	e := conflictPart(kind, target, hits, farm)
	if kind == "load" {
		e.Settings = append(e.Settings, deadFallbackSettings(target, hits, e.Conflict)...)
	}
	return e
}

// deadFallbackSettings are the settings that only turn on a fallback load another pack's load always beats: the
// pack is left out of the conflict, and its setting has no effect.
func deadFallbackSettings(target string, hits []packHit, conflict *Conflict) []Hint {
	var out []Hint
	for _, h := range hits {
		if conflict != nil && slices.Contains(conflict.Keys, h.Key) {
			continue
		}
	hit:
		for _, load := range h.Loads {
			for _, other := range hits {
				if other.Key == h.Key {
					continue
				}
				for _, otherLoad := range other.Loads {
					if !fallbackLoad(h, load, other, otherLoad) {
						continue
					}
					if hint := settingForDeadLoad(h, load, other, Conflict{Target: target}); hint != nil {
						out = append(out, *hint)
						break hit
					}
				}
			}
		}
	}
	return out
}

func conflictPart(kind, target string, hits []packHit, farm string) PartEntry {
	cosmetic := false
	var note *Note
	if kind == "edit" {
		hits, cosmetic, note = clashing(hits)
	} else {
		hits = clashingLoads(hits)
	}
	if len(hits) < 2 {
		return PartEntry{}
	}
	c := conflictOf(kind, target, hits)
	c.Evidence = ConflictEvidence(kind, hits)
	if kind == "load" {
		if allLoadFilesBlank(hits, target) || allLoadFilesIdentical(hits, target) {
			return PartEntry{}
		}
		var hint *Hint
		c.Cosmetic, hint = harmlessLoads(hits, c)
		if hint != nil {
			return PartEntry{Settings: []Hint{*hint}}
		}
	}
	if kind == "edit" {
		c.Cosmetic, c.Note = cosmetic, note
		markLoadAfterWinner(&c, hits)
	}
	c.Fixes = []Fix{}
	for _, h := range hits {
		if fix, ok := switchOff(h, hits); ok {
			c.Fixes = append(c.Fixes, fix)
		}
	}
	c.Farms = farmNeeds(kind, hits, farm)
	return PartEntry{Conflict: &c}
}

// farmNeeds is, per pack in conflict order, the farm types every one of its clashing patches is limited to,
// nil for a pack with one that applies on any farm; nil when no pack is limited.
func (h packHit) clashing(kind string) iter.Seq[Patch] {
	patches, clashes := h.Edits, h.Clashes
	if kind == "load" {
		patches, clashes = h.Loads, h.LoadClashes
	}
	return func(yield func(Patch) bool) {
		for i, p := range patches {
			if clashes != nil && !clashes[i] {
				continue
			}
			if !yield(p) {
				return
			}
		}
	}
}

func farmNeeds(kind string, hits []packHit, farm string) [][]string {
	out := make([][]string, len(hits))
	limited := false
	for i, h := range hits {
		var need []string
		for p := range h.clashing(kind) {
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

// farmIDs spells each custom farm in farms as its Data/AdditionalFarms id: a FarmType condition only gives it
// lower-cased.
func farmIDs(farms [][]string, custom map[string]string) [][]string {
	if farms == nil || len(custom) == 0 {
		return farms
	}
	ids := make(map[string]string, len(custom))
	for _, id := range custom {
		ids[strings.ToLower(id)] = id
	}
	out := make([][]string, len(farms))
	for i, need := range farms {
		if need == nil {
			continue
		}
		out[i] = make([]string, len(need))
		for j, t := range need {
			out[i][j] = t
			if id, ok := ids[t]; ok {
				out[i][j] = id
			}
		}
	}
	return out
}

// modIDToken is {{ModId}}, which Content Patcher fills with the pack's unique id, as a save records it.
var modIDToken = regexp.MustCompile(`(?i)\{\{\s*modid\s*\}\}`)

// recordFarms notes the map asset and id of each custom farm in Data/AdditionalFarms entries.
func (p *Pack) recordFarms(raw json.RawMessage) {
	var entries map[string]struct {
		ID      string `json:"ID"`
		MapName string `json:"MapName"`
	}
	if json.Unmarshal(lenientjson.Clean(raw), &entries) != nil {
		return
	}
	for key, entry := range entries {
		if entry.MapName == "" || hasToken(entry.MapName) {
			continue
		}
		id := cmp.Or(entry.ID, key)
		if p.Farms == nil {
			p.Farms = map[string]string{}
		}
		p.Farms["maps/"+strings.ToLower(entry.MapName)] = id
	}
}

func clashingLoads(hits []packHit) (out []packHit) {
	in := make([]bool, len(hits))
	for i := range hits {
		for j := i + 1; j < len(hits); j++ {
			for ai, a := range hits[i].Loads {
				for bj, b := range hits[j].Loads {
					if exclusive(a, b) || fallbackLoad(hits[i], a, hits[j], b) || fallbackLoad(hits[j], b, hits[i], a) {
						continue
					}
					in[i], in[j] = true, true
					if hits[i].LoadClashes == nil {
						hits[i].LoadClashes = map[int]bool{}
					}
					if hits[j].LoadClashes == nil {
						hits[j].LoadClashes = map[int]bool{}
					}
					hits[i].LoadClashes[ai], hits[j].LoadClashes[bj] = true, true
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
// fallback only when one of the two packs names the other (the stronger pack naming it means its author
// built the priority gap on purpose); one a blank load wipes stays reported.
func fallbackLoad(h packHit, load Patch, other packHit, otherLoad Patch) bool {
	rank := contentPatcherPriority("load", load.Priority)
	otherRank := contentPatcherPriority("load", otherLoad.Priority)
	if rank >= 0 || rank >= otherRank {
		return false
	}
	if h.Mentions[other.ID.Fold()] || other.Mentions[h.ID.Fold()] {
		return true
	}
	return otherRank >= 1000 && (loadFileBlank(h, load, load.Target) || !loadFileBlank(other, otherLoad, otherLoad.Target))
}

// clashing keeps the packs that share an overlapping edit of one target with a pack they were not built
// alongside. Packs in one entry, or where one names the other as a dependency or in a HasMod condition,
// were patched to work together, so their overlaps are intended.
func clashing(hits []packHit) (out []packHit, cosmetic bool, why *Note) {
	in := make([]bool, len(hits))
	cosmetic = true
	var note noteAgreement
	indexes := make([][]*shapeSet, len(hits))
	summaries := make([]shapeSummary, len(hits))
	for i := range hits {
		indexes[i] = indexesOf(hits[i].Edits)
		summaries[i] = summarize(indexes[i])
	}
	for i := range hits {
		for j := i + 1; j < len(hits); j++ {
			if aware(hits[i], hits[j]) || !summaries[i].mayOverlap(summaries[j]) {
				continue
			}
			clash, minor, pairNote := editsClashIndexed(hits[i].Edits, hits[j].Edits, indexes[j], sameAuthor(hits[i], hits[j]))
			if !clash {
				continue
			}
			in[i], in[j] = true, true
			markClashes(&hits[i], &hits[j])
			hits[i].Rivals = append(hits[i].Rivals, hits[j].ID)
			hits[j].Rivals = append(hits[j].Rivals, hits[i].ID)
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
	return a.Author != "" && a.Author == b.Author
}

func normalAuthor(author string) string {
	return strings.Join(strings.Fields(strings.ToLower(author)), " ")
}

// authoredPack is a pack's active Load and edit patches, kept while scanning so packs by one author can be compared.
type authoredPack struct {
	Key, ID, Name, Author, Root string
	Patches                     []Patch
	Knows                       map[string]bool // the packs it depends on or names in a HasMod condition
	// sig is the pack's PackSig when its stamps can be trusted, so its footprints can be reused by the next check.
	Sig string
}

// footprintMemo holds each pack's footprints from the last check by sig; a check keeps only the packs it saw.
var footprintMemo struct {
	sync.Mutex
	last map[string]map[string]bool
}

// bundles finds the packs whose every active patch writes where another pack by the same author also writes:
// the larger pack bundles the smaller one. Of two packs with the same footprint, the one listed later is bundled.
func bundles(packs []authoredPack) map[string]ModRef {
	byAuthor := map[string][]authoredPack{}
	for _, p := range packs {
		if p.Author != "" && len(p.Patches) > 0 {
			byAuthor[p.Author] = append(byAuthor[p.Author], p)
		}
	}
	footprintMemo.Lock()
	defer footprintMemo.Unlock()
	last, next := footprintMemo.last, map[string]map[string]bool{}
	defer func() { footprintMemo.last = next }()
	var out map[string]ModRef
	for _, group := range byAuthor {
		if len(group) < 2 {
			continue
		}
		out = bundleGroup(group, last, next, out)
	}
	return out
}

// groupFootprints lazily spells out each pack's footprint, reusing the previous scan's set under the same signature.
type groupFootprints struct {
	Group      []authoredPack
	Last, Next map[string]map[string]bool
	Sigs       []map[string]bool
}

func (g *groupFootprints) of(i int) map[string]bool {
	if g.Sigs[i] != nil {
		return g.Sigs[i]
	}
	if sig := g.Group[i].Sig; sig != "" {
		if set, ok := g.Last[sig]; ok {
			g.Sigs[i], g.Next[sig] = set, set
			return set
		}
	}
	g.Sigs[i] = make(map[string]bool, len(g.Group[i].Patches))
	for _, patch := range g.Group[i].Patches {
		g.Sigs[i][patchFootprint(patch)] = true
	}
	if g.Group[i].Sig != "" {
		g.Next[g.Group[i].Sig] = g.Sigs[i]
	}
	return g.Sigs[i]
}

func bundleGroup(group []authoredPack, last, next map[string]map[string]bool, out map[string]ModRef) map[string]ModRef {
	// Targets first: they rule out almost every pair before any footprint is spelled out.
	targets := make([]map[string]bool, len(group))
	for i, p := range group {
		targets[i] = map[string]bool{}
		for _, patch := range p.Patches {
			targets[i][patch.Target] = true
		}
	}
	fp := &groupFootprints{Group: group, Last: last, Next: next, Sigs: make([]map[string]bool, len(group))}
	for i, small := range group {
		for j, big := range group {
			if i == j || len(small.Patches) > len(big.Patches) && len(targets[i]) > len(targets[j]) {
				continue
			}
			// A pack built on the other layers over it on purpose, as an add-on does over its base.
			if small.Knows[big.ID] || big.Knows[small.ID] {
				continue
			}
			if _, bundledToo := out[big.Key]; bundledToo || !subsetOf(targets[i], targets[j]) {
				continue
			}
			a, b := fp.of(i), fp.of(j)
			if len(a) > len(b) || len(a) == len(b) && i < j || !subsetOf(a, b) || !sameLoads(small, big) {
				continue
			}
			if out == nil {
				out = map[string]ModRef{}
			}
			out[small.Key] = ModRef{Key: big.Key, Name: big.Name}
			break
		}
	}
	return out
}

// sameLoads reports whether big loads every file small loads, to the same target.
func sameLoads(small, big authoredPack) bool {
	for _, p := range small.Patches {
		if p.Kind != "load" {
			continue
		}
		digest := imageFileDigest(small.Root, p.FromFile, true)
		if digest == "" || !slices.ContainsFunc(big.Patches, func(o Patch) bool {
			return o.Kind == "load" && o.Target == p.Target && imageFileDigest(big.Root, o.FromFile, true) == digest
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
func withoutBundled(kind string, hits []packHit, bundled map[string]ModRef) []packHit {
	if len(bundled) == 0 || kind == "load" && len(hits) != 2 {
		return hits
	}
	return slices.DeleteFunc(slices.Clone(hits), func(h packHit) bool {
		by, ok := bundled[h.Key]
		return ok && slices.ContainsFunc(hits, func(o packHit) bool { return o.Key == by.Key })
	})
}

// patchFootprint is where an edit writes, not what: packs by one author that write the same places are one
// bundled in the other even when their sources are laid out differently. A load replaces the whole asset, so
// sameLoads compares its file too.
func patchFootprint(p Patch) string {
	b := make([]byte, 0, 64)
	b = append(b, p.Target...)
	b = append(b, '|')
	b = append(b, p.Action...)
	b = append(b, '|')
	b = append(b, strings.ToLower(strings.TrimSpace(p.PatchMode))...)
	if len(p.When.Places) > 0 {
		b = fmt.Appendf(b, "|%v", p.When.Places)
	}
	for _, shape := range p.Shapes {
		b = append(b, '|', shape.Kind)
		for _, n := range []int{shape.X, shape.Y, shape.W, shape.H} {
			b = strconv.AppendInt(append(b, ','), int64(n), 10)
		}
		b = append(append(append(append(append(b, ','), shape.Cells...), ','), shape.Layer...), ',')
		b = append(b, shape.Key...)
	}
	return string(b)
}

func aware(a, b packHit) bool {
	return (a.Key != "" && a.Key == b.Key) || a.Mentions[b.ID.Fold()] || b.Mentions[a.ID.Fold()]
}

func conflictOf(kind, target string, hits []packHit) Conflict {
	slices.SortFunc(hits, func(a, b packHit) int { return strings.Compare(a.ID.Fold(), b.ID.Fold()) })
	c := Conflict{Kind: kind, Target: target, Names: make([]string, len(hits)), Keys: make([]string, len(hits))}
	for i, h := range hits {
		c.PackIDs = append(c.PackIDs, h.ID)
		c.Names[i], c.Keys[i] = h.Name, h.Key
	}
	if kind == "edit" && len(hits) >= 3 && strings.Contains(strings.ToLower(target), "objects") {
		c.Cells = sharedCells(hits)
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
		c.WinnerKind = WinnerNeither
		return c
	}
	if best >= 0 && len(tied) == 1 {
		c.WinnerID, c.WinnerName, c.WinnerKind = hits[best].ID, hits[best].Name, WinnerTop
		for i, h := range hits {
			if i != best {
				c.Overridden = append(c.Overridden, h.Name)
			}
		}
		return c
	}
	if kind == "load" {
		if winner, ok := dependencyLoadWinner(hits, tied); ok {
			c.WinnerID, c.WinnerName, c.WinnerKind = hits[winner].ID, hits[winner].Name, WinnerLoadOrder
			for i, h := range hits {
				if i != winner {
					c.Overridden = append(c.Overridden, h.Name)
				}
			}
			return c
		}
	}
	c.WinnerKind = WinnerUnclear
	return c
}

// sharedCells are the cells every pack's clashing patches write, as "x,y", sorted.
func sharedCells(hits []packHit) []string {
	common := map[string]bool{}
	for _, patch := range hits[0].Edits {
		for _, shape := range patch.Shapes {
			for cell := range shapeCells(shape) {
				common[cell] = true
			}
		}
	}
	for _, hit := range hits[1:] {
		cells := map[string]bool{}
		for _, patch := range hit.Edits {
			for _, shape := range patch.Shapes {
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
	return slices.Sorted(maps.Keys(common))
}

func shapeCells(shape Shape) map[string]bool {
	out := map[string]bool{}
	if shape.Cells != "" {
		for cell := range strings.SplitSeq(shape.Cells, ";") {
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
	best, found := 0, false
	for patch := range hit.clashing(kind) {
		rank := contentPatcherPriority(kind, patch.Priority)
		if !found {
			best, found = rank, true
		} else {
			best = max(best, rank)
		}
	}
	if found {
		return best
	}
	return contentPatcherPriority(kind, hit.Priority)
}

func hasExclusiveLoad(hit packHit) bool {
	for patch := range hit.clashing("load") {
		if exclusiveLoadPriority(patch.Priority) {
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
	leftDepends := hits[left].Dependencies[hits[right].ID.Fold()]
	rightDepends := hits[right].Dependencies[hits[left].ID.Fold()]
	if leftDepends == rightDepends {
		return -1, false
	}
	if leftDepends {
		return left, true
	}
	return right, true
}

func harmlessLoads(hits []packHit, conflict Conflict) (bool, *Hint) {
	if len(hits) < 2 {
		return false, nil
	}
	if conflict.WinnerKind == WinnerNeither {
		return false, nil
	}
	if allLoadFilesIdentical(hits, conflict.Target) {
		return true, nil
	}
	winner := slices.IndexFunc(hits, func(h packHit) bool { return Equal(h.ID, conflict.WinnerID) })
	if winner < 0 {
		return allLoadsBlank(hits, conflict.Target), nil
	}

	winnerBlank := true
	for _, load := range hits[winner].Loads {
		winnerBlank = winnerBlank && loadFileBlank(hits[winner], load, conflict.Target)
	}
	var hint *Hint
	for i, hit := range hits {
		if i == winner {
			continue
		}
		h, ok := loserLoadsHint(hits, hit, hits[winner], winnerBlank, conflict)
		if !ok {
			return false, h
		}
		if h != nil {
			hint = h
		}
	}
	return true, hint
}

func allLoadsBlank(hits []packHit, target string) bool {
	for _, hit := range hits {
		for _, load := range hit.Loads {
			if !loadFileBlank(hit, load, target) {
				return false
			}
		}
	}
	return true
}

// loserLoadsHint checks a losing pack's loads against the winner. It reports false when a load makes the
// conflict real; the hint then is the setting that would settle it, if any.
func loserLoadsHint(hits []packHit, hit, winner packHit, winnerBlank bool, conflict Conflict) (*Hint, bool) {
	var hint *Hint
	for _, load := range hit.Loads {
		if loadFileBlank(hit, load, conflict.Target) || loadMatchesAny(hit, load, winner, conflict.Target) {
			continue
		}
		if winnerBlank {
			return settingForDeadLoad(hit, load, winner, conflict), false
		}
		if !loadPriorityDecided(hits, conflict) {
			return nil, false
		}
		if hitPriority("load", hit) > -1000 && !hit.Mentions[winner.ID.Fold()] {
			return nil, false
		}
		if candidate := settingForDeadLoad(hit, load, winner, conflict); candidate != nil {
			hint = candidate
		}
	}
	return hint, true
}

func loadMatchesAny(hit packHit, load Patch, winner packHit, target string) bool {
	return slices.ContainsFunc(winner.Loads, func(winningLoad Patch) bool {
		return loadFilesEqual(hit, load, winner, winningLoad, target)
	})
}

func settingForDeadLoad(hit packHit, load Patch, winner packHit, conflict Conflict) *Hint {
	if hitPriority("load", hit) > -1000 {
		return nil
	}
	for _, condition := range load.When.Config {
		schema, ok := hit.Schema[strings.ToLower(condition.Field)]
		if !ok || schema.DefaultValue == "" {
			continue
		}
		current := hit.Config[strings.ToLower(condition.Field)]
		current = cmp.Or(current, schema.DefaultValue)
		if current == schema.DefaultValue || !ConfigHolds([]Config{condition}, hit.Schema, hit.Config) {
			continue
		}
		return &Hint{
			Key: hit.Key, ID: hit.ID, Name: hit.Name, Field: schema.Key,
			Current: current, Suggested: []string{schema.DefaultValue},
			BecausePack: winner.Name, DeadTarget: conflict.Target,
		}
	}
	return nil
}

func allLoadFilesIdentical(hits []packHit, target string) bool {
	var reference any
	found := false
	for _, hit := range hits {
		for _, load := range hit.Loads {
			raw, ok := LoadFileBytes(hit, load, target)
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
		for _, load := range hit.Loads {
			found = true
			if !loadFileBlank(hit, load, target) {
				return false
			}
		}
	}
	return found
}

func loadPriorityDecided(hits []packHit, conflict Conflict) bool {
	if conflict.WinnerID == "" {
		return false
	}
	winner := slices.IndexFunc(hits, func(h packHit) bool { return Equal(h.ID, conflict.WinnerID) })
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

func loadFileBlank(hit packHit, load Patch, target string) bool {
	raw, ok := LoadFileBytes(hit, load, target)
	if !ok {
		return false
	}
	raw = bytes.TrimSpace(lenientjson.Clean(raw))
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) == nil && object != nil && len(object) == 0 {
		return true
	}
	var array []json.RawMessage
	return json.Unmarshal(raw, &array) == nil && array != nil && len(array) == 0
}

func loadFilesEqual(a packHit, ap Patch, b packHit, bp Patch, target string) bool {
	left, ok := LoadFileBytes(a, ap, target)
	if !ok {
		return false
	}
	right, ok := LoadFileBytes(b, bp, target)
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
	if json.Unmarshal(lenientjson.Clean(raw), &value) != nil {
		return nil, false
	}
	return value, true
}

func LoadFileBytes(hit packHit, load Patch, target string) ([]byte, bool) {
	path, ok := ResolveLoadFile(load.FromFile, target, hit.Tokens, hit.Present, hit.Schema, hit.Config)
	if !ok {
		return nil, false
	}
	return readPackPath(hit.Root, path)
}

func ResolveLoadFile(file, target string, tokens []Patch, present map[string]bool, schema map[string]Schema, config map[string]string) (string, bool) {
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
				if token.TokenName != name || !token.When.Holds(present) || !ConfigHolds(token.When.Config, schema, config) {
					continue
				}
				value, found = token.TokenValue, true
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
	relative, ok := foldedRel(root, rel)
	if !ok {
		if abs, inRoot := inside(root, rel); inRoot {
			noteRead(abs, nil)
		}
		return nil, false
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	info, err := os.Stat(path)
	if err != nil {
		noteUnstampable()
		return nil, false
	}
	noteRead(path, info)
	file, err := os.OpenInRoot(root, filepath.FromSlash(relative))
	if err != nil {
		return nil, false
	}
	defer func() { _ = file.Close() }()
	raw, err := safefs.ReadCapped(file, maxPackFileBytes)
	return raw, err == nil
}

// maxPackFileBytes bounds one file read from a content pack.
const maxPackFileBytes = 256 << 20

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

// foldedRel resolves rel below root as a slash path, matching each element without regard to case: content packs are
// written for Windows. A blank, rooted or absolute rel, or any ".." element, finds nothing.
func foldedRel(root, rel string) (string, bool) {
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" || strings.TrimSpace(rel) == "" ||
		os.IsPathSeparator(rel[0]) {
		return "", false
	}
	var kept []string
	for part := range strings.SplitSeq(filepath.ToSlash(rel), "/") {
		if part == ".." {
			return "", false
		}
		if part != "" && part != "." {
			kept = append(kept, part)
		}
	}
	got, err := safefs.ResolveFold(os.DirFS(root), cmp.Or(strings.Join(kept, "/"), "."))
	return got, err == nil
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
