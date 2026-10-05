package problems

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// AssetTouch is one mod or patch that writes a target (or a data key on that target).
type AssetTouch struct {
	ModID     mod.ID `json:"modId"`
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
	Winner mod.ID       `json:"winner"`
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

type cachedIndex struct {
	fingerprint string
	targets     []AssetTarget
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

//wails:ignore
func (s *Service) WhoChanges(_ context.Context, gameID, id, query string) (WhoChangesPage, error) {
	index, err := s.assetIndex(gameID, id)
	if err != nil {
		return WhoChangesPage{}, err
	}
	return WhoChangesOf(index, query), nil
}

// AssetMap pages the profile's touched assets; shared keeps only those more than one mod changes.
func (s *Service) AssetMap(_ context.Context, gameID, id, filter string, shared bool, offset int) (AssetMapPage, error) {
	index, err := s.assetIndex(gameID, id)
	if err != nil {
		return AssetMapPage{}, err
	}
	return AssetMapOf(index, filter, shared, offset), nil
}

// assetIndex builds the profile's index on first use and keeps it until the problems fingerprint changes, so
// searching and paging do not re-read every content pack.
func (s *Service) assetIndex(gameID, id string) ([]AssetTarget, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return nil, err
	}
	key := gameID + "/" + id
	fp := fingerprint(Environment{}, mods, "")
	s.mu.Lock()
	c, ok := s.assets[key]
	s.mu.Unlock()
	if ok && c.fingerprint == fp {
		return c.targets, nil
	}
	index := buildAssetIndex(mods)
	s.mu.Lock()
	if s.assets == nil {
		s.assets = map[string]cachedIndex{}
	}
	s.assets[key] = cachedIndex{fingerprint: fp, targets: index}
	s.mu.Unlock()
	return index, nil
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
		if !mod.Equal(m.ModID, target.Mods[0].ModID) {
			return true
		}
	}
	return false
}

// smapiOrder is each enabled mod's position in SMAPI's load order and the folded mod ids it loads after.
func smapiOrder(mods []Installed) (order map[string]int, after map[string][]string) {
	in := make([]loadorder.Mod, 0, len(mods))
	after = map[string][]string{}
	for _, im := range mods {
		if !im.Enabled {
			continue
		}
		m := loadorder.Mod{ID: im.ModID(), Name: im.Name, ContentPackFor: im.ContentPackForID()}
		id := im.ModID().Fold()
		for _, d := range im.Dependencies {
			if d.Required {
				m.Needs = append(m.Needs, d.ModID())
			} else {
				m.Optional = append(m.Optional, d.ModID())
			}
			after[id] = append(after[id], d.ModID().Fold())
		}
		if im.ContentPackFor != "" {
			after[id] = append(after[id], im.ContentPackForID().Fold())
		}
		in = append(in, m)
	}
	order = map[string]int{}
	for _, row := range loadorder.Resolve(in) {
		order[row.ID.Fold()] = row.Position
	}
	return order, after
}

// loadsAfter reports whether mod a already loads after mod b through its dependencies.
func loadsAfter(after map[string][]string, a, b mod.ID) bool {
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

func buildAssetIndex(mods []Installed) []AssetTarget {
	defer flushPackDiskCache(mods)
	preloadContentPacks(mods)
	present := map[string]bool{}
	for _, im := range mods {
		if im.Enabled {
			present[im.ModID().Fold()] = true
		}
	}
	order, after := smapiOrder(mods)
	grouped := map[assetIndexKey][]indexedTouch{}
	for _, im := range mods {
		if !im.Enabled {
			continue
		}
		loadOrder := order[im.ModID().Fold()]
		if isContentPatcherPack(im) {
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
			return strings.Compare(a.touch.ModID.Fold(), b.touch.ModID.Fold())
		})
		winner := indexWinner(touches)
		modsOut := make([]AssetTouch, len(touches))
		for i, t := range touches {
			t.touch.Winner = winner != "" && mod.Equal(t.touch.ModID, winner)
			t.touch.CanWin = canWin(touches, t, winner, after)
			modsOut[i] = t.touch
		}
		out = append(out, AssetTarget{Target: key.target, Key: key.key, Mods: modsOut, Winner: winner})
	}
	return out
}

func indexContentPack(im Installed, present map[string]bool, loadOrder int, grouped map[assetIndexKey][]indexedTouch) {
	pack := readContentPack(im)
	config := map[string]string{}
	if len(pack.schema) > 0 {
		config = readPackConfig(im.Folder)
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
				ModID:     im.ModID(),
				ModName:   im.Name,
				ModKey:    im.Key,
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

func indexReplacedFiles(im Installed, loadOrder int, grouped map[assetIndexKey][]indexedTouch) {
	for _, rel := range replacedAssetFiles(im.Folder) {
		target := normalizeTarget(rel)
		if target == "" {
			continue
		}
		key := assetIndexKey{target: target, kind: "load"}
		grouped[key] = append(grouped[key], indexedTouch{
			touch: AssetTouch{
				ModID:     im.ModID(),
				ModName:   im.Name,
				ModKey:    im.Key,
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

// indexWinner follows Content Patcher's order (touches are sorted by SMAPI load order). Edits apply by
// priority, then load order, then patch order (Content Patcher's EditData author guide, Priority), so the
// highest priority wins and a tie goes to the mod loaded last. Loads keep the highest priority; on a tie
// PatchManager.ApplyPatchesToAsset replaces its pick with each later candidate, so the last loaded wins too
// (the Load docs say "first", but the code only skips a strictly higher priority). Two Exclusive loads
// apply neither.
func indexWinner(touches []indexedTouch) mod.ID {
	if len(touches) == 0 {
		return ""
	}
	if len(touches) == 1 {
		return touches[0].touch.ModID
	}
	type hit struct {
		id        mod.ID
		rank      int
		exclusive bool
		order     int
		kind      string
	}
	byMod := map[string]*hit{}
	var order []string
	for _, t := range touches {
		id := t.touch.ModID
		cur, ok := byMod[id.Fold()]
		if !ok {
			cur = &hit{id: id, rank: t.rank, order: t.touch.LoadOrder, kind: t.kind}
			byMod[id.Fold()] = cur
			order = append(order, id.Fold())
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
	if best < 0 {
		return ""
	}
	return hits[tied[len(tied)-1]].id
}

// canWin reports whether making t's mod load after every other mod in the group would make it the winner.
// Order only breaks a priority tie, and a mod that another already loads after cannot be moved past it
// without a dependency cycle.
func canWin(touches []indexedTouch, t indexedTouch, winner mod.ID, after map[string][]string) bool {
	if winner == "" || mod.Equal(t.touch.ModID, winner) {
		return false
	}
	top, own := t.rank, t.rank
	for _, o := range touches {
		top = max(top, o.rank)
		if mod.Equal(o.touch.ModID, t.touch.ModID) {
			own = max(own, o.rank)
		} else if loadsAfter(after, o.touch.ModID, t.touch.ModID) {
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
