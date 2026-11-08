package contentpatcher

import (
	cp "github.com/Rethunk-Tech/go-contentpatcher"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func frameworkID(id cp.ModID) mod.ID { return mod.ID(id) }

// cpRequest is the library's view of the profile: the fields its analysis reads, in its own types.
func cpRequest(in framework.Input) cp.Request {
	return cp.Request{
		Enabled:          cpMods(in.Enabled),
		All:              cpMods(in.All),
		SkipImageOverlap: in.SkipImageOverlap,
		Order:            smapiLoadOrder,
	}
}

func cpMods(mods []framework.Mod) []cp.Mod {
	out := make([]cp.Mod, len(mods))
	for i, im := range mods {
		out[i] = cpMod(im)
	}
	return out
}

func cpMod(im framework.Mod) cp.Mod {
	deps := make([]cp.Dependency, len(im.Dependencies))
	for i, d := range im.Dependencies {
		deps[i] = cp.Dependency{UniqueID: cp.ModID(d.ModID()), Required: d.Required}
	}
	loadAfter := make([]cp.ModID, len(im.LoadAfter))
	for i, id := range im.LoadAfter {
		loadAfter[i] = cp.ModID(id)
	}
	return cp.Mod{
		Key:            im.Key,
		Name:           im.Name,
		UniqueID:       cp.ModID(im.ModID()),
		Author:         im.Author,
		Version:        im.Version,
		Folder:         im.Folder,
		Enabled:        im.Enabled,
		Dependencies:   deps,
		LoadAfter:      loadAfter,
		ContentPackFor: cp.ModID(im.ContentPackForID()),
	}
}

// smapiLoadOrder is the profile's folded ids in SMAPI's order.
func smapiLoadOrder(mods []cp.Mod) []string {
	in := make([]smapimanifest.Mod, 0, len(mods))
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		row := smapimanifest.Mod{Folder: m.Folder}
		row.UniqueID, row.ContentPackFor = m.UniqueID.Local(), m.ContentPackFor.Local()
		in = append(in, row)
	}
	out := make([]string, 0, len(in))
	for _, row := range smapimanifest.Order(in) {
		out = append(out, row.ID.Fold())
	}
	return out
}

// toFrameworkFindings is the library's findings in Mortar's terms.
func toFrameworkFindings(found cp.Findings) framework.Findings {
	out := framework.Findings{
		AssetConflicts: make([]framework.AssetConflict, 0, len(found.Conflicts)),
		Settings:       make([]framework.SettingHint, 0, len(found.Hints)),
		Redundant:      make([]framework.Redundant, 0, len(found.Shadowed)+len(found.Bundled)),
		Supporting:     found.Supporting,
		Mentioned:      found.Mentioned,
	}
	for _, c := range found.Conflicts {
		out.AssetConflicts = append(out.AssetConflicts, frameworkConflict(c))
	}
	for _, h := range found.Hints {
		out.Settings = append(out.Settings, frameworkHint(h))
	}
	for _, s := range found.Shadowed {
		out.Redundant = append(out.Redundant, frameworkShadowed(s))
	}
	return out
}

func frameworkConflict(c cp.Conflict) framework.AssetConflict {
	out := framework.AssetConflict{
		Kind:       c.Kind,
		Target:     c.Target,
		Names:      c.Names,
		Keys:       c.Keys,
		WinnerName: c.WinnerName,
		WinnerKind: winnerKind(c.WinnerKind),
		Overridden: c.Overridden,
		Cosmetic:   c.Cosmetic,
		Farms:      c.Farms,
	}
	if c.WinnerID != "" {
		out.WinnerID = frameworkID(c.WinnerID)
	}
	for _, id := range c.PackIDs {
		out.PackIDs = append(out.PackIDs, frameworkID(id))
	}
	for _, e := range c.Evidence {
		out.Evidence = append(out.Evidence, frameworkEvidence(e))
	}
	for _, fix := range c.Fixes {
		out.Fixes = append(out.Fixes, frameworkFix(fix))
	}
	if c.Note != nil {
		out.Note = &framework.ConflictNote{Kind: c.Note.Kind, Value: c.Note.Value}
	}
	if len(c.Cells) > 0 {
		out.Info = itemConflictInfo(c.Cells)
	}
	return out
}

func frameworkEvidence(e cp.Evidence) framework.ConflictEvidence {
	return framework.ConflictEvidence{
		PackName: e.PackName,
		PackID:   frameworkID(e.PackID),
		Source:   e.Source,
		Index:    e.Index,
		Action:   e.Action,
		Target:   e.Target,
		ToArea:   e.ToArea,
		FromArea: e.FromArea,
		When:     whenSummary(e.When),
		Priority: e.Priority,
		FromFile: e.FromFile,
		CropX:    e.CropX,
		CropY:    e.CropY,
		CropW:    e.CropW,
		CropH:    e.CropH,
		Keys:     evidenceKeyLabels(e.Keys),
	}
}

func frameworkFix(f cp.Fix) framework.ConflictFix {
	return framework.ConflictFix{
		Key: f.Key, ID: frameworkID(f.ID), Name: f.Name, Field: f.Field, Current: f.Current, Value: f.Value,
	}
}

func frameworkHint(h cp.Hint) framework.SettingHint {
	out := framework.SettingHint{
		Key:         h.Key,
		ID:          frameworkID(h.ID),
		Name:        h.Name,
		Field:       h.Field,
		Current:     h.Current,
		Suggested:   h.Suggested,
		ForNames:    h.ForNames,
		Description: h.Description,
		Variant:     h.Variant,
		CurrentFor:  h.CurrentFor,
	}
	for _, id := range h.For {
		out.For = append(out.For, frameworkID(id))
	}
	return out
}

func frameworkShadowed(s cp.Shadowed) framework.Redundant {
	out := framework.Redundant{
		Kind: "shadowed",
		Key:  s.Key,
		ID:   frameworkID(s.ID),
		Name: s.Name,
	}
	if s.Bundled {
		out.Kind = "bundled"
	}
	for _, ref := range s.By {
		out.By = append(out.By, framework.ModRef{Key: ref.Key, Name: ref.Name})
	}
	return out
}

func winnerKind(kind string) string {
	switch kind {
	case cp.WinnerTop:
		return framework.WinnerTop
	case cp.WinnerChosen:
		return framework.WinnerChosen
	case cp.WinnerLoadOrder:
		return framework.WinnerLoadOrder
	case cp.WinnerNeither:
		return framework.WinnerNeither
	case cp.WinnerPerEntry:
		return framework.WinnerPerEntry
	case cp.WinnerUnclear:
		return framework.WinnerUnclear
	}
	return ""
}

// evidenceKeyLabels words each data key of a patch as "Entry 301" or "Field 301.Price".
func evidenceKeyLabels(keys []string) []string {
	if keys == nil {
		return nil
	}
	out := make([]string, len(keys))
	for i, key := range keys {
		out[i] = keyLabel(key)
	}
	return out
}

// BuildAssetIndex is the profile's asset map, ranked by the loader's order.
func BuildAssetIndex(mods []framework.Mod) []cp.AssetTarget {
	return cp.BuildAssetIndex(cpMods(mods), smapiLoadOrder)
}
