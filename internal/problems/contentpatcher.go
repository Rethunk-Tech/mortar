package problems

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const contentPatcherID = "Pathoschild.ContentPatcher"

const (
	kindLoad      = "Load"
	kindEditImage = "EditImage"
	kindEditMap   = "EditMap"
)

// AssetConflict is two or more enabled Content Patcher packs that Load the same target (hard)
// or EditImage/EditMap the same target (soft). EditData on the same target is not a conflict.
type AssetConflict struct {
	Kind       string   `json:"kind"` // "load" (hard) or "edit" (soft)
	Target     string   `json:"target"`
	PackIDs    []string `json:"packIds"`
	Names      []string `json:"names"`
	Keys       []string `json:"keys"`
	WinnerID   string   `json:"winnerId"`
	WinnerName string   `json:"winnerName"`
	Overridden []string `json:"overridden"`
	// Cosmetic marks an edit conflict whose every overlap is harmless (see harmless): shown, never counted.
	Cosmetic bool `json:"cosmetic"`
	// Fixes are settings that switch off every clashing edit of one pack.
	Fixes []ConflictFix `json:"fixes"`
}

// ConflictFix sets one on/off field of a pack (Key, UniqueID) to Value, which turns off all of that pack's edits
// in the conflict; Current is the field's value now.
type ConflictFix struct {
	Key      string `json:"key"`
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
	Field    string `json:"field"`
	Current  string `json:"current"`
	Value    string `json:"value"`
}

type packHit struct {
	id           string
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
	schema       map[string]cpSchema
	config       map[string]string
	clashes      map[int]bool // indices into edits that overlap an edit of a pack it was not built with
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
	imageSource   []byte
	imageFromArea string
	// tokenValue is a dynamic token's value; a token that yields a picker value only when a mod is
	// installed is how a pack says which mod that value is for.
	tokenName  string
	tokenValue string
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
		anyOf:   append(slices.Clone(w.anyOf), o.anyOf...),
		noneOf:  append(slices.Clone(w.noneOf), o.noneOf...),
		config:  append(slices.Clone(w.config), o.config...),
		dynamic: append(slices.Clone(w.dynamic), o.dynamic...),
		flags:   append(slices.Clone(w.flags), o.flags...),
		spouse:  spouse,
		places:  places,
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
	fingerprint string
	files       []packFileStamp
	patches     []cpPatch
	tokens      []cpTokenDefinition
	mentions    map[string]bool
	schema      map[string]cpSchema
	skips       int
}

const contentPackParserVersion = 2

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
	Pack        diskCachedPack  `json:"pack"`
}

type diskCachedPack struct {
	Patches  []diskPatch           `json:"patches"`
	Tokens   []diskTokenDefinition `json:"tokens"`
	Mentions map[string]bool       `json:"mentions"`
	Schema   map[string]diskSchema `json:"schema"`
	Skips    int                   `json:"skips"`
}

type diskPatch struct {
	Kind          string              `json:"kind"`
	Target        string              `json:"target"`
	FromFile      string              `json:"fromFile"`
	Priority      string              `json:"priority"`
	PatchMode     string              `json:"patchMode"`
	When          diskWhen            `json:"when"`
	Shapes        []diskShape         `json:"shapes"`
	Spouse        string              `json:"spouse"`
	Places        map[string][]string `json:"places"`
	Image         bool                `json:"image"`
	ImageSource   []byte              `json:"imageSource"`
	ImageFromArea string              `json:"imageFromArea"`
	TokenName     string              `json:"tokenName"`
	TokenValue    string              `json:"tokenValue"`
}

type diskShape struct {
	Kind  byte   `json:"kind"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Cells string `json:"cells"`
	Layer string `json:"layer"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Tiny  bool   `json:"tiny"`
}

type diskWhen struct {
	AnyOf   [][]string             `json:"anyOf"`
	NoneOf  []string               `json:"noneOf"`
	Config  []diskConfig           `json:"config"`
	Dynamic []diskDynamicCondition `json:"dynamic"`
	Flags   []diskFlagCondition    `json:"flags"`
	Spouse  string                 `json:"spouse"`
	Places  map[string][]string    `json:"places"`
}

type diskConfig struct {
	Field         string   `json:"field"`
	Values        []string `json:"values"`
	AllowMultiple bool     `json:"allowMultiple"`
}

type diskDynamicCondition struct {
	Name     string   `json:"name"`
	Values   []string `json:"values"`
	Contains string   `json:"contains"`
	Expected bool     `json:"expected"`
}

type diskFlagCondition struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
}

type diskTokenDefinition struct {
	Name  string                     `json:"name"`
	Value string                     `json:"value"`
	When  map[string]json.RawMessage `json:"when"`
}

type diskSchema struct {
	Key           string   `json:"key"`
	DefaultValue  string   `json:"defaultValue"`
	AllowMultiple bool     `json:"allowMultiple"`
	AllowValues   []string `json:"allowValues"`
	AllowBlank    bool     `json:"allowBlank"`
	Description   string   `json:"description"`
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

func cachedPackOfDisk(disk diskCachedPack, entry diskPackEntry) cachedPack {
	out := cachedPack{
		fingerprint: entry.Fingerprint,
		files:       entry.Files,
		mentions:    disk.Mentions,
		schema:      make(map[string]cpSchema, len(disk.Schema)),
		skips:       disk.Skips,
	}
	if disk.Patches != nil {
		out.patches = make([]cpPatch, len(disk.Patches))
		for i, patch := range disk.Patches {
			out.patches[i] = cpPatchOfDisk(patch)
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
		Spouse:        patch.spouse,
		Places:        patch.places,
		Image:         patch.image,
		ImageSource:   patch.imageSource,
		ImageFromArea: patch.imageFromArea,
		TokenName:     patch.tokenName,
		TokenValue:    patch.tokenValue,
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
				Cells: shape.cells,
				Layer: shape.layer,
				Key:   shape.key,
				Value: shape.value,
				Tiny:  shape.tiny,
			}
		}
	}
	return out
}

func cpPatchOfDisk(patch diskPatch) cpPatch {
	out := cpPatch{
		kind:          patch.Kind,
		target:        patch.Target,
		fromFile:      patch.FromFile,
		priority:      patch.Priority,
		patchMode:     patch.PatchMode,
		when:          cpWhenOfDisk(patch.When),
		spouse:        patch.Spouse,
		places:        patch.Places,
		image:         patch.Image,
		imageSource:   patch.ImageSource,
		imageFromArea: patch.ImageFromArea,
		tokenName:     patch.TokenName,
		tokenValue:    patch.TokenValue,
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
				cells: shape.Cells,
				layer: shape.Layer,
				key:   shape.Key,
				value: shape.Value,
				tiny:  shape.Tiny,
			}
		}
	}
	return out
}

func diskWhenOf(when cpWhen) diskWhen {
	out := diskWhen{
		AnyOf:  when.anyOf,
		NoneOf: when.noneOf,
		Spouse: when.spouse,
		Places: when.places,
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
		anyOf:  when.AnyOf,
		noneOf: when.NoneOf,
		spouse: when.Spouse,
		places: when.Places,
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

func contentPackTargets(mod Installed) (load, edit []string, skips int) {
	pack := readContentPack(mod)
	for _, p := range pack.patches {
		if p.kind == "other" {
			continue
		}
		if p.kind == "load" {
			load = append(load, p.target)
		} else {
			edit = append(edit, p.target)
		}
	}
	return load, edit, pack.skips
}

func readContentPack(mod Installed) cachedPack {
	return readContentPackWithEnabled(mod, true)
}

func readContentPackForCleanup(mod Installed) cachedPack {
	return readContentPackWithEnabled(mod, false)
}

func readContentPackWithEnabled(mod Installed, requireEnabled bool) cachedPack {
	if (requireEnabled && !mod.Enabled) || mod.Folder == "" || !isContentPatcherPack(mod.Folder) {
		return cachedPack{}
	}
	root := filepath.Clean(mod.Folder)
	if _, err := os.Stat(filepath.Join(root, "content.json")); err != nil {
		return cachedPack{}
	}
	if c, ok := packCache.Load(root); ok {
		got, ok := c.(cachedPack)
		if ok && packFingerprintValid(root, got.files, got.fingerprint) {
			return got
		}
	}
	cachePath := loadPackDiskCache()
	if cachePath != "" {
		if entry, ok := diskPackEntryFor(root); ok && packFingerprintValid(root, entry.Files, entry.Fingerprint) {
			pack := cachedPackOfDisk(entry.Pack, entry)
			packCache.Store(root, pack)
			clearDiskPackPayload(root)
			return pack
		}
	}
	pack := cachedPack{mentions: map[string]bool{}, schema: readConfigSchema(root)}
	pack.recordPackFile(root, filepath.Join(root, "content.json"))
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	slices.SortFunc(pack.files, func(a, b packFileStamp) int {
		return strings.Compare(a.Path, b.Path)
	})
	pack.fingerprint = packFilesFingerprint(pack.files)
	packCache.Store(root, pack)
	if cachePath != "" {
		storeDiskPackEntry(root, diskPackEntry{
			Fingerprint: pack.fingerprint,
			Files:       slices.Clone(pack.files),
			Pack:        diskPackOf(pack),
		})
	}
	return pack
}

func (p *cachedPack) recordPackFile(root, abs string) {
	info, err := os.Stat(abs)
	if err != nil {
		return
	}
	relative, err := filepath.Rel(root, abs)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return
	}
	stamp := packFileStamp{
		Path:    filepath.ToSlash(relative),
		Size:    info.Size(),
		ModTime: info.ModTime().UnixNano(),
	}
	for i, existing := range p.files {
		if existing.Path == stamp.Path {
			p.files[i] = stamp
			return
		}
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
		if err != nil || info.Size() != expected.Size || info.ModTime().UnixNano() != expected.ModTime {
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
	var cache diskPackCache
	if json.Unmarshal(raw, &cache) == nil && cache.Version == contentPackParserVersion && cache.Packs != nil {
		packDiskState.entries = cache.Packs
	}
	return path
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
	entry.Pack = diskCachedPack{}
	packDiskState.entries[root] = entry
	packDiskState.dirty = true
}

func clearDiskPackPayload(root string) {
	packDiskState.Lock()
	defer packDiskState.Unlock()
	entry, ok := packDiskState.entries[root]
	if ok {
		entry.Pack = diskCachedPack{}
		packDiskState.entries[root] = entry
	}
}

func flushPackDiskCache(mods []Installed) {
	path := loadPackDiskCache()
	if path == "" {
		return
	}
	present := map[string]bool{}
	for _, mod := range mods {
		if mod.Folder != "" {
			present[filepath.Clean(mod.Folder)] = true
		}
	}
	packDiskState.Lock()
	for root := range packDiskState.entries {
		if !present[root] {
			delete(packDiskState.entries, root)
			packDiskState.dirty = true
		}
	}
	if !packDiskState.dirty {
		packDiskState.Unlock()
		return
	}
	entries := make(map[string]diskPackEntry, len(packDiskState.entries))
	for root, entry := range packDiskState.entries {
		if cached, ok := packCache.Load(root); ok {
			if pack, ok := cached.(cachedPack); ok {
				entry.Pack = diskPackOf(pack)
			}
		}
		entries[root] = entry
	}
	raw, err := json.Marshal(diskPackCache{
		Version: contentPackParserVersion,
		Packs:   entries,
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
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		packDiskState.Lock()
		packDiskState.dirty = true
		packDiskState.Unlock()
		return
	}
	packDiskState.Lock()
	for root, entry := range packDiskState.entries {
		if _, ok := packCache.Load(root); ok {
			entry.Pack = diskCachedPack{}
			packDiskState.entries[root] = entry
		}
	}
	packDiskState.Unlock()
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
	if json.Unmarshal(stripJSONNoise(raw), &doc) != nil {
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

func isContentPatcherPack(folder string) bool {
	raw, err := fsx.ReadFile(filepath.Join(folder, "manifest.json"))
	if err != nil {
		return false
	}
	var doc map[string]json.RawMessage
	if json.Unmarshal(stripJSONNoise(raw), &doc) != nil {
		return false
	}
	var pack map[string]json.RawMessage
	for k, v := range doc {
		if strings.EqualFold(k, "contentpackfor") {
			if json.Unmarshal(v, &pack) != nil {
				return false
			}
			break
		}
	}
	for k, v := range pack {
		if !strings.EqualFold(k, "uniqueid") {
			continue
		}
		var id string
		if json.Unmarshal(v, &id) != nil {
			return false
		}
		return sameID(strings.TrimSpace(id), contentPatcherID)
	}
	return false
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
	raw, err := fsx.ReadFile(abs)
	if err != nil {
		return
	}
	pack.recordPackFile(root, abs)
	var doc struct {
		Changes       []json.RawMessage `json:"Changes"`
		DynamicTokens []struct {
			Name  string                     `json:"Name"`
			Value json.RawMessage            `json:"Value"`
			When  map[string]json.RawMessage `json:"When"`
		} `json:"DynamicTokens"`
	}
	if err := json.Unmarshal(stripJSONNoise(raw), &doc); err != nil {
		return
	}
	// A dynamic token's value gates content like a change does, so its conditions count for compatibility
	// settings too. Content Patcher reads DynamicTokens only from content.json.
	if key == "content.json" {
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
	for _, rawChange := range doc.Changes {
		var ch cpChange
		if json.Unmarshal(rawChange, &ch) != nil {
			continue
		}
		action := strings.TrimSpace(ch.Action)
		when := outer.with(parseWhenWithTokens(ch.When, pack.mentions, pack.schema, pack.tokens))
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
			ch.FromFile = contentSourceReference(root, rel, ch.FromFile)
		case strings.EqualFold(action, kindEditImage), strings.EqualFold(action, kindEditMap):
			kind = "edit"
			ch.FromFile = contentSourceReference(root, rel, ch.FromFile)
			recordReferencedPackFiles(root, ch.FromFile, strings.EqualFold(action, kindEditImage), pack)
			shapes = editShapes(root, ch, strings.EqualFold(action, kindEditImage))
			if len(shapes) == 0 {
				kind = "other"
			}
		default:
			kind = "other"
		}
		// Changes that cannot conflict still count for compatibility settings, whatever their target.
		if kind == "other" {
			pack.patches = append(pack.patches, cpPatch{kind: kind, when: when})
			continue
		}
		for _, t := range splitTargets(ch.Target) {
			if hasToken(t) {
				pack.skips++
				continue
			}
			priority, _ := scalarValue(ch.Priority)
			pack.patches = append(pack.patches, cpPatch{
				kind: kind, target: normalizeTarget(t), fromFile: ch.FromFile, priority: strings.TrimSpace(priority),
				patchMode: strings.TrimSpace(ch.PatchMode), when: when,
				shapes: shapes, spouse: when.spouse, places: when.places, image: strings.EqualFold(action, kindEditImage),
				imageSource:   imageSource(root, ch.FromFile, strings.EqualFold(action, kindEditImage)),
				imageFromArea: string(stripJSONNoise(ch.FromArea)),
			})
		}
	}
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
			w.flags = append(w.flags, flags...)
			continue
		}
		if field, ok := schema[strings.ToLower(name)]; ok {
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
	}
	return w.with(cpWhen{spouse: spouseOf(raw), places: placesOf(raw)})
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
		return cpDynamicCondition{name: name, values: values}, true
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

func dynamicWhenHolds(when cpWhen, tokens []cpTokenDefinition, present map[string]bool, schema map[string]cpSchema, config map[string]string) bool {
	if len(when.dynamic) == 0 {
		return true
	}
	reachable := reachableDynamicTokens(tokens, present, schema, config, when.flags)
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
			if present[strings.ToLower(id)] {
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
	found := slices.ContainsFunc(ids, func(id string) bool { return present[strings.ToLower(id)] })
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
		out[i] = strings.ToLower(id)
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

func contentReference(_, rel string) string {
	rel = strings.ReplaceAll(strings.TrimSpace(rel), "\\", "/")
	if rel == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
}

func contentSourceReference(_, _, rel string) string {
	return contentReference("content.json", rel)
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
	relToRoot, err := filepath.Rel(root, joined)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return joined, true
}

// stripJSONNoise removes // and /* */ comments and trailing commas so Content Patcher's JSON parses.
func stripJSONNoise(b []byte) []byte {
	out := make([]byte, 0, len(b))
	inStr := false
	esc := false
	i := 0
	for i < len(b) {
		c := b[i]
		if inStr {
			out = append(out, c)
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				inStr = false
			}
			i++
			continue
		}
		if c == '"' {
			inStr = true
			out = append(out, c)
			i++
			continue
		}
		if c == '/' && i+1 < len(b) {
			if b[i+1] == '/' {
				i += 2
				for i < len(b) && b[i] != '\n' {
					i++
				}
				continue
			}
			if b[i+1] == '*' {
				i += 2
				for i+1 < len(b) && (b[i] != '*' || b[i+1] != '/') {
					i++
				}
				if i+1 < len(b) {
					i += 2
				}
				continue
			}
		}
		if c == ',' {
			j := i + 1
			for j < len(b) && unicode.IsSpace(rune(b[j])) {
				j++
			}
			for j < len(b) && b[j] == '/' && j+1 < len(b) && (b[j+1] == '/' || b[j+1] == '*') {
				if b[j+1] == '/' {
					for j < len(b) && b[j] != '\n' {
						j++
					}
				} else {
					j += 2
					for j+1 < len(b) && (b[j] != '*' || b[j+1] != '/') {
						j++
					}
					if j+1 < len(b) {
						j += 2
					}
				}
				for j < len(b) && unicode.IsSpace(rune(b[j])) {
					j++
				}
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				i++
				continue
			}
		}
		out = append(out, c)
		i++
	}
	return out
}

func assetConflicts(mods []Installed) []AssetConflict {
	conflicts, _ := assetConflictResults(mods)
	return conflicts
}

func assetConflictResults(mods []Installed) ([]AssetConflict, []SettingHint) {
	defer flushPackDiskCache(mods)
	present := map[string]bool{}
	for _, mod := range mods {
		if mod.Enabled {
			present[strings.ToLower(mod.UniqueID)] = true
		}
	}
	at := map[string]map[string][]packHit{"load": {}, "edit": {}}
	for _, mod := range mods {
		pack := readContentPack(mod)
		knows := maps.Clone(pack.mentions)
		dependencies := map[string]bool{}
		for _, d := range mod.Dependencies {
			if knows == nil {
				knows = map[string]bool{}
			}
			id := strings.ToLower(d.UniqueID)
			knows[id] = true
			dependencies[id] = true
		}
		config := map[string]string{}
		if len(pack.schema) > 0 {
			config = readPackConfig(mod.Folder)
		}
		for _, p := range pack.patches {
			if p.kind == "other" || !p.when.holds(present) || !dynamicWhenHolds(p.when, pack.tokens, present, pack.schema, config) {
				continue
			}
			hits := at[p.kind][p.target]
			i := slices.IndexFunc(hits, func(h packHit) bool { return sameID(h.id, mod.UniqueID) })
			if i < 0 {
				hits = append(hits, packHit{
					id: mod.UniqueID, name: mod.Name, key: mod.Key, priority: p.priority, mentions: knows,
					root: mod.Folder, tokens: pack.patches, present: present, schema: pack.schema, config: config,
					dependencies: dependencies,
				})
				i = len(hits) - 1
				at[p.kind][p.target] = hits
			} else {
				hits[i].priority = strongerContentPatcherPriority(hits[i].priority, p.priority, p.kind)
			}
			if p.kind == "edit" {
				hits[i].eligible = append(hits[i].eligible, p)
				if !configHolds(p.when.config, pack.schema, config) {
					continue
				}
				hits[i].edits = append(hits[i].edits, p)
			} else {
				if !configHolds(p.when.config, pack.schema, config) {
					continue
				}
				hits[i].loads = append(hits[i].loads, p)
			}
		}
	}
	out := []AssetConflict{}
	settings := []SettingHint{}
	for kind, targets := range at {
		for t, hits := range targets {
			cosmetic := false
			if kind == "edit" {
				hits, cosmetic = clashing(hits)
			} else {
				hits = clashingLoads(hits)
			}
			if len(hits) >= 2 {
				c := conflictOf(kind, t, hits)
				if kind == "load" {
					if allLoadFilesBlank(hits, t) || allLoadFilesIdentical(hits, t) {
						continue
					}
					var hint *SettingHint
					c.Cosmetic, hint = harmlessLoads(hits, c)
					if hint != nil {
						settings = append(settings, *hint)
						continue
					}
				}
				if kind == "edit" {
					c.Cosmetic = cosmetic
				}
				c.Fixes = []ConflictFix{}
				for _, h := range hits {
					if fix, ok := switchOff(h, hits); ok {
						c.Fixes = append(c.Fixes, fix)
					}
				}
				out = append(out, c)
			}
		}
	}
	slices.SortFunc(out, func(a, b AssetConflict) int {
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

func clashingLoads(hits []packHit) (out []packHit) {
	in := make([]bool, len(hits))
	for i := range hits {
		for j := i + 1; j < len(hits); j++ {
			for ai, a := range hits[i].loads {
				for bj, b := range hits[j].loads {
					if exclusive(a, b) {
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

// clashing keeps the packs that share an overlapping edit of one target with a pack they were not built
// alongside. Packs in one entry, or where one names the other as a dependency or in a HasMod condition,
// were patched to work together, so their overlaps are intended.
func clashing(hits []packHit) (out []packHit, cosmetic bool) {
	in := make([]bool, len(hits))
	cosmetic = true
	for i := range hits {
		for j := i + 1; j < len(hits); j++ {
			if aware(hits[i], hits[j]) {
				continue
			}
			clash, minor := editsClash(hits[i].edits, hits[j].edits)
			if !clash {
				continue
			}
			in[i], in[j] = true, true
			markClashes(&hits[i], &hits[j])
			cosmetic = cosmetic && minor
		}
	}
	for i, h := range hits {
		if in[i] {
			out = append(out, h)
		}
	}
	return out, cosmetic
}

func aware(a, b packHit) bool {
	return (a.key != "" && a.key == b.key) || a.mentions[strings.ToLower(b.id)] || b.mentions[strings.ToLower(a.id)]
}

func conflictOf(kind, target string, hits []packHit) AssetConflict {
	slices.SortFunc(hits, func(a, b packHit) int { return strings.Compare(strings.ToLower(a.id), strings.ToLower(b.id)) })
	c := AssetConflict{Kind: kind, Target: target, PackIDs: make([]string, len(hits)), Names: make([]string, len(hits)), Keys: make([]string, len(hits))}
	for i, h := range hits {
		c.PackIDs[i], c.Names[i], c.Keys[i] = h.id, h.name, h.key
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
		if strings.EqualFold(strings.TrimSpace(patch.priority), "exclusive") {
			return true
		}
	}
	return false
}

func dependencyLoadWinner(hits []packHit, tied []int) (int, bool) {
	if len(tied) != 2 {
		return -1, false
	}
	left, right := tied[0], tied[1]
	leftDepends := hits[left].dependencies[strings.ToLower(hits[right].id)]
	rightDepends := hits[right].dependencies[strings.ToLower(hits[left].id)]
	if leftDepends == rightDepends {
		return -1, false
	}
	if leftDepends {
		return left, true
	}
	return right, true
}

func harmlessLoads(hits []packHit, conflict AssetConflict) (bool, *SettingHint) {
	if len(hits) < 2 {
		return false, nil
	}
	if conflict.WinnerName == "CP applies neither" {
		return false, nil
	}
	if allLoadFilesIdentical(hits, conflict.Target) {
		return true, nil
	}
	winner := slices.IndexFunc(hits, func(h packHit) bool { return sameID(h.id, conflict.WinnerID) })
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
	var hint *SettingHint
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
			if hitPriority("load", hit) > -1000 && !hit.mentions[strings.ToLower(hits[winner].id)] {
				return false, nil
			}
			if candidate := settingForDeadLoad(hit, load, hits[winner], conflict); candidate != nil {
				hint = candidate
			}
		}
	}
	return true, hint
}

func settingForDeadLoad(hit packHit, load cpPatch, winner packHit, conflict AssetConflict) *SettingHint {
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
		return &SettingHint{
			Key: hit.key, UniqueID: hit.id, Name: hit.name, Field: schema.key,
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

func loadPriorityDecided(hits []packHit, conflict AssetConflict) bool {
	if conflict.WinnerID == "" {
		return false
	}
	winner := slices.IndexFunc(hits, func(h packHit) bool { return sameID(h.id, conflict.WinnerID) })
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
	raw = bytes.TrimSpace(stripJSONNoise(raw))
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
	if json.Unmarshal(stripJSONNoise(raw), &value) != nil {
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
		return nil, false
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(relative) {
		return nil, false
	}
	file, err := os.OpenInRoot(root, relative)
	if err != nil {
		return nil, false
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(file)
	return raw, err == nil
}

func imageSource(root, rel string, image bool) []byte {
	if !image {
		return nil
	}
	raw, ok := readPackPath(root, rel)
	if !ok {
		return nil
	}
	return raw
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

func dismissBucket(gameID, profileID string) string {
	return "~mortar/cp/" + gameID + "/" + profileID
}

func dismissToken(kind, target string) string {
	return kind + "\t" + target
}

func settingChoiceToken(uniqueID, field, value string) string {
	target := strings.ToLower(strings.TrimSpace(uniqueID)) + "\t" +
		strings.ToLower(strings.TrimSpace(field)) + "\t" +
		strings.ToLower(strings.TrimSpace(value))
	return dismissToken("setting-choice", target)
}

func hideDismissedBroken(broken []Broken, tokens []string) ([]Broken, []DismissedProblem) {
	if len(tokens) == 0 {
		return broken, nil
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []Broken{}
	dismissed := []DismissedProblem{}
	for _, b := range broken {
		token := dismissToken("abandoned", strings.ToLower(b.UniqueID))
		if b.Status == "abandoned" && skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, Broken: &b})
			continue
		}
		out = append(out, b)
	}
	return out, dismissed
}

func hideDismissedListed(missing []Missing, tokens []string) ([]Missing, []DismissedProblem) {
	if len(tokens) == 0 {
		return missing, nil
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []Missing{}
	dismissed := []DismissedProblem{}
	for _, m := range missing {
		token := dismissToken("listed", strings.ToLower(m.UniqueID))
		if m.Listed && skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, Missing: &m})
			continue
		}
		out = append(out, m)
	}
	return out, dismissed
}

func hideDismissedSettings(settings []SettingHint, tokens []string) ([]SettingHint, []DismissedProblem) {
	if len(tokens) == 0 {
		return settings, nil
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []SettingHint{}
	dismissed := []DismissedProblem{}
	for _, setting := range settings {
		target := strings.ToLower(setting.UniqueID) + "\t" + strings.ToLower(setting.Field)
		token := dismissToken("setting", target)
		if skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, Setting: &setting})
			continue
		}
		token = settingChoiceToken(setting.UniqueID, setting.Field, setting.Current)
		if skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, Setting: &setting})
			continue
		}
		out = append(out, setting)
	}
	return out, dismissed
}

func hideDismissed(conflicts []AssetConflict, tokens []string) ([]AssetConflict, []DismissedProblem) {
	if len(tokens) == 0 {
		return conflicts, nil
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []AssetConflict{}
	dismissed := []DismissedProblem{}
	for _, c := range conflicts {
		token := dismissToken(c.Kind, c.Target)
		if skip[token] {
			dismissed = append(dismissed, DismissedProblem{Token: token, AssetConflict: &c})
			continue
		}
		out = append(out, c)
	}
	return out, dismissed
}
