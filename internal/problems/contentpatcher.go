package problems

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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
	Kind    string   `json:"kind"` // "load" (hard) or "edit" (soft)
	Target  string   `json:"target"`
	PackIDs []string `json:"packIds"`
	Names   []string `json:"names"`
	Keys    []string `json:"keys"`
}

type packHit struct {
	id   string
	name string
	key  string
}

type cachedPack struct {
	mtime time.Time
	load  []string
	edit  []string
	skips int
}

var packCache sync.Map // folder path -> cachedPack

func contentPackTargets(mod Installed) (load, edit []string, skips int) {
	if !mod.Enabled || mod.Folder == "" || !isContentPatcherPack(mod.Folder) {
		return nil, nil, 0
	}
	root := filepath.Clean(mod.Folder)
	info, err := os.Stat(filepath.Join(root, "content.json"))
	if err != nil {
		return nil, nil, 0
	}
	if c, ok := packCache.Load(root); ok {
		got, ok := c.(cachedPack)
		if ok && got.mtime.Equal(info.ModTime()) {
			return got.load, got.edit, got.skips
		}
	}
	load, edit, skips = scanContentFile(root, "content.json", map[string]bool{})
	packCache.Store(root, cachedPack{mtime: info.ModTime(), load: load, edit: edit, skips: skips})
	return load, edit, skips
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

func scanContentFile(root, rel string, seen map[string]bool) (load, edit []string, skips int) {
	rel = filepath.ToSlash(rel)
	key := strings.ToLower(rel)
	if rel == "" || seen[key] {
		return nil, nil, 0
	}
	seen[key] = true
	abs, ok := inside(root, rel)
	if !ok {
		return nil, nil, 0
	}
	raw, err := fsx.ReadFile(abs)
	if err != nil {
		return nil, nil, 0
	}
	var doc struct {
		Changes []cpChange `json:"Changes"`
	}
	if err := json.Unmarshal(stripJSONNoise(raw), &doc); err != nil {
		return nil, nil, 0
	}
	for _, ch := range doc.Changes {
		action := strings.TrimSpace(ch.Action)
		if strings.EqualFold(action, "Include") {
			for _, from := range splitTargets(ch.FromFile) {
				if hasToken(from) {
					skips++
					continue
				}
				l, e, s := scanContentFile(root, from, seen)
				load = append(load, l...)
				edit = append(edit, e...)
				skips += s
			}
			continue
		}
		targets := splitTargets(ch.Target)
		switch {
		case strings.EqualFold(action, kindLoad):
			for _, t := range targets {
				if hasToken(t) {
					skips++
					continue
				}
				load = append(load, normalizeTarget(t))
			}
		case strings.EqualFold(action, kindEditImage), strings.EqualFold(action, kindEditMap):
			for _, t := range targets {
				if hasToken(t) {
					skips++
					continue
				}
				edit = append(edit, normalizeTarget(t))
			}
		}
	}
	return load, edit, skips
}

type cpChange struct {
	Action   string `json:"Action"`
	Target   string `json:"Target"`
	FromFile string `json:"FromFile"`
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
	loadAt := map[string][]packHit{}
	editAt := map[string][]packHit{}
	for _, mod := range mods {
		load, edit, _ := contentPackTargets(mod)
		hit := packHit{id: mod.UniqueID, name: mod.Name, key: mod.Key}
		for _, t := range load {
			if !hasPack(loadAt[t], hit.id) {
				loadAt[t] = append(loadAt[t], hit)
			}
		}
		for _, t := range edit {
			if !hasPack(editAt[t], hit.id) {
				editAt[t] = append(editAt[t], hit)
			}
		}
	}
	out := []AssetConflict{}
	for t, hits := range loadAt {
		if len(hits) >= 2 {
			out = append(out, conflictOf("load", t, hits))
		}
	}
	for t, hits := range editAt {
		if len(hits) >= 2 {
			out = append(out, conflictOf("edit", t, hits))
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

func hasPack(hits []packHit, id string) bool {
	return slices.ContainsFunc(hits, func(h packHit) bool { return sameID(h.id, id) })
}

func conflictOf(kind, target string, hits []packHit) AssetConflict {
	slices.SortFunc(hits, func(a, b packHit) int { return strings.Compare(strings.ToLower(a.id), strings.ToLower(b.id)) })
	c := AssetConflict{Kind: kind, Target: target, PackIDs: make([]string, len(hits)), Names: make([]string, len(hits)), Keys: make([]string, len(hits))}
	for i, h := range hits {
		c.PackIDs[i], c.Names[i], c.Keys[i] = h.id, h.name, h.key
	}
	return c
}

func dismissBucket(gameID, profileID string) string {
	return "~mortar/cp/" + gameID + "/" + profileID
}

func dismissToken(kind, target string) string {
	return kind + "\t" + target
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
