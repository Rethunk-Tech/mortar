package problems

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

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
	id       string
	name     string
	key      string
	priority string
	mentions map[string]bool
	edits    []cpPatch // the pack's active edits of this target
	schema   map[string]cpSchema
	config   map[string]string
	clashes  map[int]bool // indices into edits that overlap an edit of a pack it was not built with
}

// cpPatch is one Load or EditImage/EditMap change with the HasMod conditions that gate it.
type cpPatch struct {
	kind     string // "load" or "edit"
	target   string
	priority string
	when     cpWhen
	shapes   []cpShape           // what an edit writes; see editShapes
	spouse   string              // the spouse the change requires, or ""
	places   map[string][]string // literal values the change requires of placeTokens
	image    bool                // an EditImage change, which only changes how something looks
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
// present, and no noneOf mod may be present. Other conditions are treated as met.
type cpWhen struct {
	anyOf  [][]string
	noneOf []string
	config []cpConfig
}

func (w cpWhen) with(o cpWhen) cpWhen {
	return cpWhen{
		anyOf:  append(slices.Clone(w.anyOf), o.anyOf...),
		noneOf: append(slices.Clone(w.noneOf), o.noneOf...),
		config: append(slices.Clone(w.config), o.config...),
	}
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
	mtime    time.Time
	patches  []cpPatch
	mentions map[string]bool
	schema   map[string]cpSchema
	skips    int
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
	if !mod.Enabled || mod.Folder == "" || !isContentPatcherPack(mod.Folder) {
		return cachedPack{}
	}
	root := filepath.Clean(mod.Folder)
	info, err := os.Stat(filepath.Join(root, "content.json"))
	if err != nil {
		return cachedPack{}
	}
	if c, ok := packCache.Load(root); ok {
		got, ok := c.(cachedPack)
		if ok && got.mtime.Equal(info.ModTime()) {
			return got
		}
	}
	pack := cachedPack{mtime: info.ModTime(), mentions: map[string]bool{}, schema: readConfigSchema(root)}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	packCache.Store(root, pack)
	return pack
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
	var doc struct {
		Changes       []cpChange `json:"Changes"`
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
			value, _ := scalarValue(tok.Value)
			pack.patches = append(pack.patches, cpPatch{kind: "other", when: outer.with(parseWhen(tok.When, pack.mentions, pack.schema)), tokenName: strings.ToLower(strings.TrimSpace(tok.Name)), tokenValue: value})
		}
	}
	for _, ch := range doc.Changes {
		action := strings.TrimSpace(ch.Action)
		when := outer.with(parseWhen(ch.When, pack.mentions, pack.schema))
		var kind string
		var shapes []cpShape
		switch {
		case strings.EqualFold(action, "Include"):
			for _, from := range splitTargets(ch.FromFile) {
				if hasToken(from) {
					pack.skips++
					continue
				}
				scanContentFile(root, from, seen, when, pack)
			}
			continue
		case strings.EqualFold(action, kindLoad):
			kind = "load"
		case strings.EqualFold(action, kindEditImage), strings.EqualFold(action, kindEditMap):
			kind = "edit"
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
			pack.patches = append(pack.patches, cpPatch{
				kind: kind, target: normalizeTarget(t), priority: strings.TrimSpace(ch.Priority), when: when,
				shapes: shapes, spouse: spouseOf(ch.When), places: placesOf(ch.When), image: strings.EqualFold(action, kindEditImage),
			})
		}
	}
}

// parseWhen reads the HasMod and schema-backed config conditions of a When block.
// It understands "HasMod": "A, B" and "HasMod |contains=A, B": true/false.
func parseWhen(raw map[string]json.RawMessage, mentions map[string]bool, schema map[string]cpSchema) cpWhen {
	var w cpWhen
	for k, v := range raw {
		name, arg, _ := strings.Cut(k, "|")
		name = tokenName(name)
		if hasToken(k) {
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
	return w
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
	Priority      string                     `json:"Priority"`
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
		for _, d := range mod.Dependencies {
			if knows == nil {
				knows = map[string]bool{}
			}
			knows[strings.ToLower(d.UniqueID)] = true
		}
		config := map[string]string{}
		if len(pack.schema) > 0 {
			config = readPackConfig(mod.Folder)
		}
		for _, p := range pack.patches {
			if p.kind == "other" || !p.when.holds(present) || !configHolds(p.when.config, pack.schema, config) {
				continue
			}
			hits := at[p.kind][p.target]
			i := slices.IndexFunc(hits, func(h packHit) bool { return sameID(h.id, mod.UniqueID) })
			if i < 0 {
				hits = append(hits, packHit{id: mod.UniqueID, name: mod.Name, key: mod.Key, priority: p.priority, mentions: knows, schema: pack.schema, config: config})
				i = len(hits) - 1
				at[p.kind][p.target] = hits
			} else {
				hits[i].priority = strongerContentPatcherPriority(hits[i].priority, p.priority, p.kind)
			}
			if p.kind == "edit" {
				hits[i].edits = append(hits[i].edits, p)
			}
		}
	}
	out := []AssetConflict{}
	for kind, targets := range at {
		for t, hits := range targets {
			cosmetic := false
			if kind == "edit" {
				hits, cosmetic = clashing(hits)
			}
			if len(hits) >= 2 {
				c := conflictOf(kind, t, hits)
				c.Cosmetic = cosmetic
				c.Fixes = []ConflictFix{}
				for _, h := range hits {
					if fix, ok := switchOff(h); ok {
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
	return a.key == b.key || a.mentions[strings.ToLower(b.id)] || b.mentions[strings.ToLower(a.id)]
}

func conflictOf(kind, target string, hits []packHit) AssetConflict {
	slices.SortFunc(hits, func(a, b packHit) int { return strings.Compare(strings.ToLower(a.id), strings.ToLower(b.id)) })
	c := AssetConflict{Kind: kind, Target: target, PackIDs: make([]string, len(hits)), Names: make([]string, len(hits)), Keys: make([]string, len(hits))}
	for i, h := range hits {
		c.PackIDs[i], c.Names[i], c.Keys[i] = h.id, h.name, h.key
	}
	best := -1
	bestRank := -1
	tied := false
	exclusive := 0
	for i, hit := range hits {
		rank := contentPatcherPriority(kind, hit.priority)
		if kind == "load" && strings.EqualFold(strings.TrimSpace(hit.priority), "exclusive") {
			exclusive++
		}
		if rank > bestRank {
			best, bestRank, tied = i, rank, false
		} else if rank == bestRank {
			tied = true
		}
	}
	if best >= 0 && !tied && exclusive < 2 {
		c.WinnerID, c.WinnerName = hits[best].id, hits[best].name
		for i, h := range hits {
			if i != best {
				c.Overridden = append(c.Overridden, h.name)
			}
		}
	} else {
		c.WinnerName = "unclear"
	}
	return c
}

func contentPatcherPriority(kind, priority string) int {
	priority = strings.ToLower(strings.TrimSpace(priority))
	if kind == "load" {
		if priority == "" {
			priority = "exclusive"
		}
		switch priority {
		case "low":
			return 0
		case "medium":
			return 1000
		case "high":
			return 2000
		case "exclusive":
			return 3000
		}
		return 1000
	}
	if priority == "" {
		priority = "default"
	}
	switch priority {
	case "early":
		return 0
	case "default":
		return 1000
	case "late":
		return 2000
	}
	return 1000
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

func hideDismissedBroken(broken []Broken, tokens []string) []Broken {
	if len(tokens) == 0 {
		return broken
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []Broken{}
	for _, b := range broken {
		if b.Status == "abandoned" && skip[dismissToken("abandoned", strings.ToLower(b.UniqueID))] {
			continue
		}
		out = append(out, b)
	}
	return out
}

func hideDismissedListed(missing []Missing, tokens []string) []Missing {
	if len(tokens) == 0 {
		return missing
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []Missing{}
	for _, m := range missing {
		if m.Listed && skip[dismissToken("listed", strings.ToLower(m.UniqueID))] {
			continue
		}
		out = append(out, m)
	}
	return out
}

func hideDismissedSettings(settings []SettingHint, tokens []string) []SettingHint {
	if len(tokens) == 0 {
		return settings
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []SettingHint{}
	for _, setting := range settings {
		target := strings.ToLower(setting.UniqueID) + "\t" + strings.ToLower(setting.Field)
		if skip[dismissToken("setting", target)] {
			continue
		}
		out = append(out, setting)
	}
	return out
}

func hideDismissed(conflicts []AssetConflict, tokens []string) []AssetConflict {
	if len(tokens) == 0 {
		return conflicts
	}
	skip := map[string]bool{}
	for _, t := range tokens {
		skip[t] = true
	}
	out := []AssetConflict{}
	for _, c := range conflicts {
		if c.Kind == "edit" && skip[dismissToken(c.Kind, c.Target)] {
			continue
		}
		out = append(out, c)
	}
	return out
}
