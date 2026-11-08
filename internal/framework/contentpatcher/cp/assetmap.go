package cp

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// AssetTouch is one mod or patch that writes a target (or a data key on that target).
type AssetTouch struct {
	ModID     ModID  `json:"modId"`
	ModName   string `json:"modName"`
	ModKey    string `json:"modKey"`
	Action    string `json:"action"`
	Priority  string `json:"priority"`
	LoadOrder int    `json:"loadOrder"`
	Winner    bool   `json:"winner"`
	DataKey   string `json:"dataKey,omitempty"`
	Source    string `json:"source,omitempty"`
	Index     int    `json:"index"`
	// CanWin is set on a losing mod that loading later would make win: it ties the winner's priority and no
	// other mod in the group already loads after it.
	CanWin bool `json:"canWin,omitempty"`
}

// AssetTarget is every recorded touch of one normalised asset (and optional data key).
type AssetTarget struct {
	Target string       `json:"target"`
	Key    string       `json:"key,omitempty"`
	Mods   []AssetTouch `json:"mods"`
	Winner ModID        `json:"winner"`
}

// WhoChangesPage is the fuzzy search result for WhoChanges.
type WhoChangesPage struct {
	Query   string        `json:"query"`
	Targets []AssetTarget `json:"targets"`
}

// AssetMapPage is a sorted, paged slice of touched assets. All and Shared count the assets the search matches,
// and those of them more than one mod changes; Total is the count the page slices.
type AssetMapPage struct {
	Targets []AssetTarget `json:"targets"`
	Total   int           `json:"total"`
	Offset  int           `json:"offset"`
	All     int           `json:"all"`
	Shared  int           `json:"shared"`
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
	Target string
	Key    string
	Kind   string
}

type indexedTouch struct {
	Touch AssetTouch
	Kind  string
	Rank  int
}

func WhoChangesOf(index []AssetTarget, query string) WhoChangesPage {
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

func AssetMapOf(index []AssetTarget, filter string, shared bool, offset int) AssetMapPage {
	needles := queryNeedles(filter)
	page := AssetMapPage{Offset: max(offset, 0), Targets: []AssetTarget{}}
	matched := make([]AssetTarget, 0, len(index))
	for _, target := range index {
		if len(needles) > 0 && !assetMatches(target, needles) {
			continue
		}
		page.All++
		many := sharedTarget(target)
		if many {
			page.Shared++
		}
		if many || !shared {
			matched = append(matched, target)
		}
	}
	offset = page.Offset
	page.Total = len(matched)
	if offset >= len(matched) {
		return page
	}
	end := min(offset+assetMapPageSize, len(matched))
	page.Targets = matched[offset:end]
	return page
}

func sharedTarget(target AssetTarget) bool {
	for _, m := range target.Mods {
		if !Equal(m.ModID, target.Mods[0].ModID) {
			return true
		}
	}
	return false
}

// smapiOrder is each enabled mod's position in SMAPI's load order and the folded mod ids it loads after.
func smapiOrder(mods []Mod, order func([]Mod) []string) (rank map[string]int, after map[string][]string) {
	after = map[string][]string{}
	for _, im := range mods {
		if !im.Enabled {
			continue
		}
		id := im.UniqueID.Fold()
		for _, d := range im.Dependencies {
			after[id] = append(after[id], d.UniqueID.Fold())
		}
		if im.ContentPackFor != "" {
			after[id] = append(after[id], im.ContentPackFor.Fold())
		}
	}
	rank = map[string]int{}
	if order != nil {
		for i, id := range order(mods) {
			rank[id] = i + 1
		}
	}
	return rank, after
}

// loadsAfter reports whether mod a already loads after mod b through its dependencies.
func loadsAfter(after map[string][]string, a, b ModID) bool {
	seen := map[string]bool{}
	stack := []string{a.Fold()}
	target := b.Fold()
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, dep := range after[id] {
			if dep == target {
				return true
			}
			if !seen[dep] {
				seen[dep] = true
				stack = append(stack, dep)
			}
		}
	}
	return false
}

// BuildAssetIndex maps every asset the profile's mods write. order is the profile's load order as folded ids; nil
// leaves every mod at the same rank.
func BuildAssetIndex(mods []Mod, order func([]Mod) []string) []AssetTarget {
	defer HoldPacks()()
	defer FlushPackDiskCache(mods)
	preloadContentPacks(mods, false)
	present := map[string]bool{}
	for _, im := range mods {
		if im.Enabled {
			present[im.UniqueID.Fold()] = true
		}
	}
	rank, after := smapiOrder(mods, order)
	grouped := map[assetIndexKey][]indexedTouch{}
	for _, im := range mods {
		if !im.Enabled {
			continue
		}
		loadOrder := rank[im.UniqueID.Fold()]
		if IsContentPatcherPack(im) {
			indexContentPack(im, present, loadOrder, grouped)
			continue
		}
		indexReplacedFiles(im, loadOrder, grouped)
	}
	keys := make([]assetIndexKey, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b assetIndexKey) int {
		if a.Target != b.Target {
			return strings.Compare(a.Target, b.Target)
		}
		if a.Key != b.Key {
			return strings.Compare(a.Key, b.Key)
		}
		return strings.Compare(a.Kind, b.Kind)
	})
	out := make([]AssetTarget, 0, len(keys))
	for _, key := range keys {
		touches := grouped[key]
		slices.SortFunc(touches, func(a, b indexedTouch) int {
			if a.Touch.LoadOrder != b.Touch.LoadOrder {
				return a.Touch.LoadOrder - b.Touch.LoadOrder
			}
			return strings.Compare(a.Touch.ModID.Fold(), b.Touch.ModID.Fold())
		})
		winner := indexWinner(touches)
		modsOut := make([]AssetTouch, len(touches))
		for i, t := range touches {
			t.Touch.Winner = winner != "" && Equal(t.Touch.ModID, winner)
			t.Touch.CanWin = canWin(touches, t, winner, after)
			modsOut[i] = t.Touch
		}
		out = append(out, AssetTarget{Target: key.Target, Key: key.Key, Mods: modsOut, Winner: winner})
	}
	return out
}

func indexContentPack(im Mod, present map[string]bool, loadOrder int, grouped map[assetIndexKey][]indexedTouch) {
	pack := ReadPack(im, false)
	config := map[string]string{}
	if len(pack.Schema) > 0 {
		config = ReadPackConfig(im.Folder)
	}
	for _, p := range pack.Patches {
		if p.Kind == "other" || p.Target == "" {
			continue
		}
		if !p.When.Holds(present) || !DynamicWhenHolds(p.When, pack.Tokens, present, pack.Schema, config) {
			continue
		}
		if !ConfigHolds(p.When.Config, pack.Schema, config) {
			continue
		}
		kind := p.Kind
		if kind == "" {
			if strings.EqualFold(p.Action, kindLoad) {
				kind = "load"
			} else {
				kind = "edit"
			}
		}
		base := indexedTouch{
			Touch: AssetTouch{
				ModID:     im.UniqueID,
				ModName:   im.Name,
				ModKey:    im.Key,
				Action:    p.Action,
				Priority:  p.Priority,
				LoadOrder: loadOrder,
				Source:    p.Source,
				Index:     p.Index,
			},
			Kind: kind,
			Rank: contentPatcherPriority(kind, p.Priority),
		}
		dataKeys := dataKeysOf(p)
		if len(dataKeys) == 0 {
			key := assetIndexKey{Target: p.Target, Kind: kind}
			grouped[key] = append(grouped[key], base)
			continue
		}
		for _, dataKey := range dataKeys {
			item := base
			item.Touch.DataKey = dataKey
			key := assetIndexKey{Target: p.Target, Key: dataKey, Kind: kind}
			grouped[key] = append(grouped[key], item)
		}
	}
}

func indexReplacedFiles(im Mod, loadOrder int, grouped map[assetIndexKey][]indexedTouch) {
	for _, rel := range replacedAssetFiles(im.Folder) {
		target := NormalizeTarget(rel)
		if target == "" {
			continue
		}
		key := assetIndexKey{Target: target, Kind: "load"}
		grouped[key] = append(grouped[key], indexedTouch{
			Touch: AssetTouch{
				ModID:     im.UniqueID,
				ModName:   im.Name,
				ModKey:    im.Key,
				Action:    kindLoad,
				LoadOrder: loadOrder,
			},
			Kind: "load",
			Rank: contentPatcherPriority("load", ""),
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

func dataKeysOf(p Patch) []string {
	if p.Action != kindEditData {
		return nil
	}
	var keys []string
	seen := map[string]bool{}
	for _, s := range p.Shapes {
		if s.Kind != 'p' {
			continue
		}
		label := strings.TrimPrefix(s.Key, "entry:")
		label = strings.TrimPrefix(label, "field:")
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		keys = append(keys, label)
	}
	return keys
}

// indexWinner follows Content Patcher's order (touches are sorted by SMAPI load order). Edits apply by
// priority, then load order, then patch order (Content Patcher's EditData author guide, Priority), so the
// highest priority wins and a tie goes to the mod loaded last. Loads keep the highest priority; on a tie
// PatchManager.ApplyPatchesToAsset replaces its pick with each later candidate, so the last loaded wins too
// (the Load docs say "first", but the code only skips a strictly higher priority). Two Exclusive loads
// apply neither.
func indexWinner(touches []indexedTouch) ModID {
	if len(touches) == 0 {
		return ""
	}
	if len(touches) == 1 {
		return touches[0].Touch.ModID
	}
	type hit struct {
		ID        ModID
		Rank      int
		Exclusive bool
		Order     int
		Kind      string
	}
	byMod := map[string]*hit{}
	var order []string
	for _, t := range touches {
		id := t.Touch.ModID
		cur, ok := byMod[id.Fold()]
		if !ok {
			cur = &hit{ID: id, Rank: t.Rank, Order: t.Touch.LoadOrder, Kind: t.Kind}
			byMod[id.Fold()] = cur
			order = append(order, id.Fold())
		}
		if t.Rank > cur.Rank {
			cur.Rank = t.Rank
		}
		if t.Kind == "load" && exclusiveLoadPriority(t.Touch.Priority) {
			cur.Exclusive = true
		}
	}
	hits := make([]hit, len(order))
	for i, id := range order {
		hits[i] = *byMod[id]
	}
	kind := hits[0].Kind
	best := -1
	bestRank := -1 << 31
	var tied []int
	exclusiveCount := 0
	for i, h := range hits {
		if h.Exclusive {
			exclusiveCount++
		}
		if h.Rank > bestRank {
			best, bestRank, tied = i, h.Rank, []int{i}
		} else if h.Rank == bestRank {
			tied = append(tied, i)
		}
	}
	if kind == "load" && exclusiveCount >= 2 {
		return ""
	}
	if best < 0 {
		return ""
	}
	return hits[tied[len(tied)-1]].ID
}

// canWin reports whether making t's mod load after every other mod in the group would make it the winner.
// Order only breaks a priority tie, and a mod that another already loads after cannot be moved past it
// without a dependency cycle.
func canWin(touches []indexedTouch, t indexedTouch, winner ModID, after map[string][]string) bool {
	if winner == "" || Equal(t.Touch.ModID, winner) {
		return false
	}
	top, own := t.Rank, t.Rank
	for _, o := range touches {
		top = max(top, o.Rank)
		if Equal(o.Touch.ModID, t.Touch.ModID) {
			own = max(own, o.Rank)
		} else if loadsAfter(after, o.Touch.ModID, t.Touch.ModID) {
			return false
		}
	}
	return own == top
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
