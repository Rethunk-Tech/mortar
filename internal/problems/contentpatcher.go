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
}

type packHit struct {
	id       string
	name     string
	key      string
	priority string
	mentions map[string]bool
}

// cpPatch is one Load or EditImage/EditMap change with the HasMod conditions that gate it.
type cpPatch struct {
	kind     string // "load" or "edit"
	target   string
	priority string
	when     cpWhen
}

// cpWhen holds a change's HasMod conditions: each anyOf group needs one of its mods
// present, and no noneOf mod may be present. Other conditions are treated as met.
type cpWhen struct {
	anyOf  [][]string
	noneOf []string
}

func (w cpWhen) with(o cpWhen) cpWhen {
	return cpWhen{anyOf: append(slices.Clone(w.anyOf), o.anyOf...), noneOf: append(slices.Clone(w.noneOf), o.noneOf...)}
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
	skips    int
}

var packCache sync.Map // folder path -> cachedPack

func contentPackTargets(mod Installed) (load, edit []string, skips int) {
	pack := readContentPack(mod)
	for _, p := range pack.patches {
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
	pack := cachedPack{mtime: info.ModTime(), mentions: map[string]bool{}}
	scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
	packCache.Store(root, pack)
	return pack
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
		Changes []cpChange `json:"Changes"`
	}
	if err := json.Unmarshal(stripJSONNoise(raw), &doc); err != nil {
		return
	}
	for _, ch := range doc.Changes {
		action := strings.TrimSpace(ch.Action)
		when := outer.with(parseWhen(ch.When, pack.mentions))
		var kind string
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
		default:
			continue
		}
		for _, t := range splitTargets(ch.Target) {
			if hasToken(t) {
				pack.skips++
				continue
			}
			pack.patches = append(pack.patches, cpPatch{kind: kind, target: normalizeTarget(t), priority: strings.TrimSpace(ch.Priority), when: when})
		}
	}
}

// parseWhen reads the HasMod conditions of a When block and records every mod they name.
// It understands "HasMod": "A, B" and "HasMod |contains=A, B": true/false.
func parseWhen(raw map[string]json.RawMessage, mentions map[string]bool) cpWhen {
	var w cpWhen
	for k, v := range raw {
		name, arg, _ := strings.Cut(k, "|")
		if !strings.EqualFold(strings.TrimSpace(name), "hasmod") {
			continue
		}
		arg = strings.TrimSpace(arg)
		if arg == "" {
			var ids []string
			if !condValues(v, &ids) {
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
	}
	return w
}

// condValues reads a condition value given as a string, comma list, bool or array; false when it holds a token.
func condValues(v json.RawMessage, out *[]string) bool {
	var str string
	var b bool
	var arr []string
	switch {
	case json.Unmarshal(v, &str) == nil:
	case json.Unmarshal(v, &b) == nil:
		str = strconv.FormatBool(b)
	case json.Unmarshal(v, &arr) == nil:
		str = strings.Join(arr, ",")
	default:
		return false
	}
	if hasToken(str) {
		return false
	}
	*out = splitTargets(str)
	return len(*out) > 0
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
	Action   string                     `json:"Action"`
	Target   string                     `json:"Target"`
	FromFile string                     `json:"FromFile"`
	Priority string                     `json:"Priority"`
	When     map[string]json.RawMessage `json:"When"`
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
		for _, p := range pack.patches {
			if !p.when.holds(present) {
				continue
			}
			hits := at[p.kind][p.target]
			i := slices.IndexFunc(hits, func(h packHit) bool { return sameID(h.id, mod.UniqueID) })
			if i < 0 {
				at[p.kind][p.target] = append(hits, packHit{id: mod.UniqueID, name: mod.Name, key: mod.Key, priority: p.priority, mentions: knows})
				continue
			}
			hits[i].priority = strongerContentPatcherPriority(hits[i].priority, p.priority, p.kind)
		}
	}
	out := []AssetConflict{}
	for kind, targets := range at {
		for t, hits := range targets {
			if len(hits) >= 2 && (kind == "load" || !allAware(hits)) {
				out = append(out, conflictOf(kind, t, hits))
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

// allAware reports whether every pair of packs editing one target was built to work together:
// they ship in one entry, or one names the other as a dependency or in a HasMod condition.
func allAware(hits []packHit) bool {
	for i := range hits {
		for j := i + 1; j < len(hits); j++ {
			if hits[i].key != hits[j].key && !hits[i].mentions[strings.ToLower(hits[j].id)] && !hits[j].mentions[strings.ToLower(hits[i].id)] {
				return false
			}
		}
	}
	return true
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
