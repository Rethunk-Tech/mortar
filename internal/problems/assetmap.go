package problems

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

// AssetTouch is one mod or patch that writes a target (or a data key on that target).
type AssetTouch struct {
	ModID     string `json:"modId"`
	ModName   string `json:"modName"`
	ModKey    string `json:"modKey"`
	Action    string `json:"action"`
	Priority  string `json:"priority"`
	LoadOrder int    `json:"loadOrder"`
	Winner    bool   `json:"winner"`
	DataKey   string `json:"dataKey,omitempty"`
	Source    string `json:"source,omitempty"`
	Index     int    `json:"index"`
}

// AssetTarget is every recorded touch of one normalised asset (and optional data key).
type AssetTarget struct {
	Target string       `json:"target"`
	Key    string       `json:"key,omitempty"`
	Mods   []AssetTouch `json:"mods"`
	Winner string       `json:"winner"`
}

// WhoChangesPage is the fuzzy search result for WhoChanges.
type WhoChangesPage struct {
	Query   string        `json:"query"`
	Targets []AssetTarget `json:"targets"`
}

// AssetMapPage is a sorted, paged slice of touched assets.
type AssetMapPage struct {
	Targets []AssetTarget `json:"targets"`
	Total   int           `json:"total"`
	Offset  int           `json:"offset"`
}

const assetMapPageSize = 100

var assetAliases = map[string][]string{
	"abigail":    {"characters/abigail", "portraits/abigail"},
	"alex":       {"characters/alex", "portraits/alex"},
	"elliott":    {"characters/elliott", "portraits/elliott"},
	"emily":      {"characters/emily", "portraits/emily"},
	"haley":      {"characters/haley", "portraits/haley"},
	"harvey":     {"characters/harvey", "portraits/harvey"},
	"leah":       {"characters/leah", "portraits/leah"},
	"maru":       {"characters/maru", "portraits/maru"},
	"penny":      {"characters/penny", "portraits/penny"},
	"sam":        {"characters/sam", "portraits/sam"},
	"sebastian":  {"characters/sebastian", "portraits/sebastian"},
	"shane":      {"characters/shane", "portraits/shane"},
	"farmhouse":  {"maps/farmhouse"},
	"farm house": {"maps/farmhouse"},
	"town":       {"maps/town"},
	"objects":    {"data/objects"},
}

type assetIndexKey struct {
	target string
	key    string
	kind   string
}

type indexedTouch struct {
	touch AssetTouch
	kind  string
	rank  int
}

func (s *Service) WhoChanges(_ context.Context, gameID, id, query string) (WhoChangesPage, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return WhoChangesPage{}, err
	}
	return WhoChangesOf(mods, query), nil
}

func (s *Service) AssetMap(_ context.Context, gameID, id, filter string, offset int) (AssetMapPage, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return AssetMapPage{}, err
	}
	return AssetMapOf(mods, filter, offset), nil
}

func WhoChangesOf(mods []Installed, query string) WhoChangesPage {
	index := buildAssetIndex(mods)
	needles := queryNeedles(query)
	out := WhoChangesPage{Query: strings.TrimSpace(query), Targets: []AssetTarget{}}
	if len(needles) == 0 {
		return out
	}
	for _, target := range index {
		if assetMatches(target, needles) {
			out.Targets = append(out.Targets, target)
		}
	}
	return out
}

func AssetMapOf(mods []Installed, filter string, offset int) AssetMapPage {
	index := buildAssetIndex(mods)
	needles := queryNeedles(filter)
	matched := index
	if len(needles) > 0 {
		filtered := make([]AssetTarget, 0, len(index))
		for _, target := range index {
			if assetMatches(target, needles) {
				filtered = append(filtered, target)
			}
		}
		matched = filtered
	}
	if offset < 0 {
		offset = 0
	}
	page := AssetMapPage{Total: len(matched), Offset: offset, Targets: []AssetTarget{}}
	if offset >= len(matched) {
		return page
	}
	end := min(offset+assetMapPageSize, len(matched))
	page.Targets = matched[offset:end]
	return page
}

func buildAssetIndex(mods []Installed) []AssetTarget {
	defer flushPackDiskCache(mods)
	preloadContentPacks(mods)
	present := map[string]bool{}
	order := map[string]int{}
	for i, mod := range mods {
		if mod.Enabled {
			present[strings.ToLower(mod.UniqueID)] = true
		}
		order[strings.ToLower(mod.UniqueID)] = i
	}
	grouped := map[assetIndexKey][]indexedTouch{}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		loadOrder := order[strings.ToLower(mod.UniqueID)]
		if isContentPatcherPack(mod) {
			indexContentPack(mod, present, loadOrder, grouped)
			continue
		}
		indexReplacedFiles(mod, loadOrder, grouped)
	}
	keys := make([]assetIndexKey, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b assetIndexKey) int {
		if a.target != b.target {
			return strings.Compare(a.target, b.target)
		}
		if a.key != b.key {
			return strings.Compare(a.key, b.key)
		}
		return strings.Compare(a.kind, b.kind)
	})
	out := make([]AssetTarget, 0, len(keys))
	for _, key := range keys {
		touches := grouped[key]
		slices.SortFunc(touches, func(a, b indexedTouch) int {
			if a.touch.LoadOrder != b.touch.LoadOrder {
				return a.touch.LoadOrder - b.touch.LoadOrder
			}
			return strings.Compare(strings.ToLower(a.touch.ModID), strings.ToLower(b.touch.ModID))
		})
		winner := indexWinner(touches)
		modsOut := make([]AssetTouch, len(touches))
		for i, t := range touches {
			t.touch.Winner = winner != "" && profile.SameID(t.touch.ModID, winner)
			modsOut[i] = t.touch
		}
		out = append(out, AssetTarget{Target: key.target, Key: key.key, Mods: modsOut, Winner: winner})
	}
	return out
}

func indexContentPack(mod Installed, present map[string]bool, loadOrder int, grouped map[assetIndexKey][]indexedTouch) {
	pack := readContentPack(mod)
	config := map[string]string{}
	if len(pack.schema) > 0 {
		config = readPackConfig(mod.Folder)
	}
	for _, p := range pack.patches {
		if p.kind == "other" || p.target == "" {
			continue
		}
		if !p.when.holds(present) || !dynamicWhenHolds(p.when, pack.tokens, present, pack.schema, config) {
			continue
		}
		if !configHolds(p.when.config, pack.schema, config) {
			continue
		}
		kind := p.kind
		if kind == "" {
			if strings.EqualFold(p.action, kindLoad) {
				kind = "load"
			} else {
				kind = "edit"
			}
		}
		base := indexedTouch{
			touch: AssetTouch{
				ModID:     mod.UniqueID,
				ModName:   mod.Name,
				ModKey:    mod.Key,
				Action:    p.action,
				Priority:  p.priority,
				LoadOrder: loadOrder,
				Source:    p.source,
				Index:     p.index,
			},
			kind: kind,
			rank: contentPatcherPriority(kind, p.priority),
		}
		dataKeys := dataKeysOf(p)
		if len(dataKeys) == 0 {
			key := assetIndexKey{target: p.target, kind: kind}
			grouped[key] = append(grouped[key], base)
			continue
		}
		for _, dataKey := range dataKeys {
			item := base
			item.touch.DataKey = dataKey
			key := assetIndexKey{target: p.target, key: dataKey, kind: kind}
			grouped[key] = append(grouped[key], item)
		}
	}
}

func indexReplacedFiles(mod Installed, loadOrder int, grouped map[assetIndexKey][]indexedTouch) {
	for _, rel := range replacedAssetFiles(mod.Folder) {
		target := normalizeTarget(rel)
		if target == "" {
			continue
		}
		key := assetIndexKey{target: target, kind: "load"}
		grouped[key] = append(grouped[key], indexedTouch{
			touch: AssetTouch{
				ModID:     mod.UniqueID,
				ModName:   mod.Name,
				ModKey:    mod.Key,
				Action:    kindLoad,
				LoadOrder: loadOrder,
			},
			kind: "load",
			rank: contentPatcherPriority("load", ""),
		})
	}
}

func replacedAssetFiles(root string) []string {
	if root == "" {
		return nil
	}
	var out []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			switch strings.ToLower(name) {
			case "i18n", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(name), ".xnb") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		rel = strings.TrimSuffix(rel, filepath.Ext(rel))
		if rel != "" {
			out = append(out, rel)
		}
		return nil
	})
	return out
}

func dataKeysOf(p cpPatch) []string {
	if p.action != kindEditData {
		return nil
	}
	var keys []string
	seen := map[string]bool{}
	for _, s := range p.shapes {
		if s.kind != 'p' {
			continue
		}
		label := packScope.ReplaceAllString(s.key, "")
		label = strings.TrimPrefix(label, "entry:")
		label = strings.TrimPrefix(label, "field:")
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		keys = append(keys, label)
	}
	return keys
}

func indexWinner(touches []indexedTouch) string {
	if len(touches) == 0 {
		return ""
	}
	if len(touches) == 1 {
		return touches[0].touch.ModID
	}
	type hit struct {
		id        string
		rank      int
		exclusive bool
		order     int
		kind      string
	}
	byMod := map[string]*hit{}
	var order []string
	for _, t := range touches {
		id := t.touch.ModID
		cur, ok := byMod[strings.ToLower(id)]
		if !ok {
			cur = &hit{id: id, rank: t.rank, order: t.touch.LoadOrder, kind: t.kind}
			byMod[strings.ToLower(id)] = cur
			order = append(order, strings.ToLower(id))
		}
		if t.rank > cur.rank {
			cur.rank = t.rank
		}
		if t.kind == "load" && exclusiveLoadPriority(t.touch.Priority) {
			cur.exclusive = true
		}
	}
	hits := make([]hit, len(order))
	for i, id := range order {
		hits[i] = *byMod[id]
	}
	kind := hits[0].kind
	best := -1
	bestRank := -1 << 31
	var tied []int
	exclusiveCount := 0
	for i, h := range hits {
		if h.exclusive {
			exclusiveCount++
		}
		if h.rank > bestRank {
			best, bestRank, tied = i, h.rank, []int{i}
		} else if h.rank == bestRank {
			tied = append(tied, i)
		}
	}
	if kind == "load" && exclusiveCount >= 2 {
		return ""
	}
	if best >= 0 && len(tied) == 1 {
		return hits[best].id
	}
	if kind == "load" && len(tied) == 2 {
		a, b := tied[0], tied[1]
		if hits[a].order != hits[b].order {
			if hits[a].order > hits[b].order {
				return hits[a].id
			}
			return hits[b].id
		}
	}
	return ""
}

func queryNeedles(query string) []string {
	q := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(query, "\\", "/")))
	if q == "" {
		return nil
	}
	needles := []string{q}
	if aliases, ok := assetAliases[q]; ok {
		needles = append(needles, aliases...)
	}
	return needles
}

func assetMatches(target AssetTarget, needles []string) bool {
	hay := target.Target
	if target.Key != "" {
		hay += " " + strings.ToLower(target.Key)
	}
	for _, n := range needles {
		if n == "" {
			continue
		}
		if strings.Contains(hay, n) || strings.Contains(strings.ToLower(target.Key), n) {
			return true
		}
		for _, m := range target.Mods {
			if strings.Contains(strings.ToLower(m.ModName), n) || strings.Contains(strings.ToLower(m.DataKey), n) {
				return true
			}
		}
	}
	return false
}
