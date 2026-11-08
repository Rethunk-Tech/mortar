package contentpatcher

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	cp "github.com/Rethunk-Tech/go-contentpatcher"
)

type settingGroup struct {
	pack      cp.Mod
	schema    cp.Schema
	current   string
	required  []cp.Mod
	suggested []string
	active    bool
}

func compatibilitySettings(mods []cp.Mod, run *cp.PartsRun, skip bool) []cp.Hint {
	present := map[string]bool{}
	byID := map[string]cp.Mod{}
	for _, im := range mods {
		if !im.Enabled {
			continue
		}
		id := im.UniqueID.Fold()
		if id == "" {
			continue
		}
		present[id] = true
		if _, exists := byID[id]; !exists {
			byID[id] = im
		}
	}
	// Only a mod that is some recolour can be one of a pack's enabled recolours.
	recolours := slices.DeleteFunc(slices.Clone(mods), func(m cp.Mod) bool { return len(enabledRecolours([]cp.Mod{m}, "")) == 0 })

	var out []cp.Hint
	for _, packMod := range mods {
		if !packMod.Enabled {
			continue
		}
		pack := cp.ReadPack(packMod, skip)
		if len(pack.Schema) == 0 {
			continue
		}
		seen := cp.PresentFor(pack, present)
		enabled := enabledRecolours(recolours, packMod.UniqueID)
		sig, stable := cp.PackSig(packMod, pack, seen)
		fields := []string{"settings", sig}
		for _, id := range slices.Sorted(maps.Keys(seen)) {
			fields = append(fields, fmt.Sprintf("%q %q", byID[id].UniqueID, byID[id].Name))
		}
		for i := range recolourFamilies {
			if im, ok := enabled[i]; ok {
				fields = append(fields, fmt.Sprintf("r%d %q %q", i, im.UniqueID, im.Name))
			}
		}
		e := run.Part(cp.PartKey(skip, fields...), stable, func() cp.PartEntry {
			return cp.PartEntry{Settings: packSettings(packMod, pack, seen, byID, enabled)}
		})
		out = append(out, e.Settings...)
	}
	slices.SortStableFunc(out, func(a, b cp.Hint) int {
		if c := strings.Compare(strings.ToLower(a.Key), strings.ToLower(b.Key)); c != 0 {
			return c
		}
		if c := strings.Compare(a.ID.Fold(), b.ID.Fold()); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.Field), strings.ToLower(b.Field))
	})
	return out
}

// packSettings are one pack's compatibility settings; present holds the enabled mods its conditions name.
func packSettings(packMod cp.Mod, pack cp.Pack, present map[string]bool, byID map[string]cp.Mod, recolours map[int]cp.Mod) []cp.Hint {
	config := cp.ReadPackConfig(packMod.Folder)
	variants := variantSettings(packMod, pack, config, present, byID)
	covered := map[string]bool{}
	for _, hint := range variants {
		covered[strings.ToLower(hint.Field)] = true
	}
	out := slices.Clone(variants)
	out = append(out, recolourSettings(packMod, pack, config, recolours, covered)...)
	groups := map[string]*settingGroup{}
	for _, patch := range pack.Patches {
		if !patch.When.Holds(present) || !cp.DynamicWhenHolds(patch.When, pack.Tokens, present, pack.Schema, config) {
			continue
		}
		ids := enabledRequirements(patch.When, present, packMod.UniqueID)
		if len(ids) == 0 {
			continue
		}
		required := make([]cp.Mod, 0, len(ids))
		for _, id := range ids {
			required = append(required, byID[id])
		}
		for _, condition := range patch.When.Config {
			addSettingCondition(groups, packMod, pack, config, condition, required)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(groups)) {
		group := groups[key]
		if group.active || len(group.suggested) == 0 {
			continue
		}
		out = append(out, group.hint())
	}
	return out
}

// addSettingCondition folds one boolean-toggle condition into its field's group. One group per field: a value
// that already runs a patch for any installed mod is the player's choice, even when other patches for other
// mod combinations need a different value.
func addSettingCondition(groups map[string]*settingGroup, packMod cp.Mod, pack cp.Pack, config map[string]string, condition cp.Config, required []cp.Mod) {
	schema, ok := pack.Schema[strings.ToLower(condition.Field)]
	if !ok || len(condition.Values) == 0 || !schema.BooleanToggle() {
		return
	}
	current, set := config[strings.ToLower(schema.Key)]
	if !set {
		current = schema.DefaultValue
	}
	groupKey := strings.ToLower(schema.Key)
	group := groups[groupKey]
	if group == nil {
		group = &settingGroup{
			pack:     packMod,
			schema:   schema,
			current:  current,
			required: required,
		}
		groups[groupKey] = group
	} else {
		for _, im := range required {
			if !slices.ContainsFunc(group.required, func(m cp.Mod) bool { return cp.Equal(m.UniqueID, im.UniqueID) }) {
				group.required = append(group.required, im)
			}
		}
	}
	for _, value := range condition.Values {
		cp.AddSettingValue(&group.suggested, value)
	}
	if cp.SettingValueMatches(current, condition) {
		group.active = true
	}
}

func (group *settingGroup) hint() cp.Hint {
	hint := cp.Hint{
		Key:         group.pack.Key,
		ID:          group.pack.UniqueID,
		Name:        group.pack.Name,
		Field:       group.schema.Key,
		Current:     group.current,
		Suggested:   group.suggested,
		Description: group.schema.Description,
	}
	for _, im := range group.required {
		hint.For = append(hint.For, im.UniqueID)
		hint.ForNames = append(hint.ForNames, cmp.Or(im.Name, im.UniqueID.Local()))
	}
	return hint
}

func enabledRequirements(when cp.When, present map[string]bool, own cp.ModID) []string {
	ids := map[string]bool{}
	for _, group := range when.AnyOf {
		for _, id := range group {
			if id != "" && id != own.Fold() && present[id] {
				ids[id] = true
			}
		}
	}
	out := slices.Sorted(maps.Keys(ids))
	return out
}

// variantSettings suggests a picker value from the pack's own mapping of values to mods: a dynamic token that
// yields one of the field's allowed values only when a mod is installed (Value "Earthy" When HasMod
// DaisyNiko.EarthyRecolour). A field with two or more such values is a picker. Its automatic choice is a
// blank value when allowed, or the value the mapping tokens themselves require of the field ("Off"), since
// with it the pack detects the installed mod on its own.
func variantSettings(packMod cp.Mod, pack cp.Pack, config map[string]string, present map[string]bool, byID map[string]cp.Mod) []cp.Hint {
	var out []cp.Hint
	for _, schema := range pack.Schema {
		if schema.AllowMultiple || len(schema.AllowValues) < 2 {
			continue
		}
		forValue, auto := variantChoices(packMod, pack, schema)
		if len(forValue) < 2 {
			continue
		}
		if hint, ok := variantHint(packMod, schema, config, present, byID, forValue, auto); ok {
			out = append(out, hint)
		}
	}
	return out
}

// ownTokens are the field's own tokens: those that read it ({{Field}}) or test it. Other tokens can yield
// the same words ("Default") for unrelated choices.
func ownTokens(pack cp.Pack, schema cp.Schema) map[string]bool {
	own := map[string]bool{}
	for _, p := range pack.Patches {
		if p.TokenName == "" {
			continue
		}
		reads := strings.Contains(strings.ToLower(p.TokenValue), "{{"+strings.ToLower(schema.Key)+"}}")
		if reads || slices.ContainsFunc(p.When.Config, func(c cp.Config) bool { return strings.EqualFold(c.Field, schema.Key) }) {
			own[p.TokenName] = true
		}
	}
	return own
}

// variantChoices maps each lower-cased allowed value to the mod ids that select it, and lists the values the
// pack's own conditions name.
func variantChoices(packMod cp.Mod, pack cp.Pack, schema cp.Schema) (forValue map[string][]string, auto []string) {
	own := ownTokens(pack, schema)
	forValue = map[string][]string{}
	for _, p := range pack.Patches {
		usesField := slices.ContainsFunc(p.When.Config, func(c cp.Config) bool {
			return strings.EqualFold(c.Field, schema.Key)
		})
		if !own[p.TokenName] && !usesField {
			continue
		}
		ids := make([]string, 0)
		for _, group := range p.When.AnyOf {
			for _, id := range group {
				if id != packMod.UniqueID.Fold() {
					ids = append(ids, id)
				}
			}
		}
		if len(ids) == 0 {
			continue
		}
		if own[p.TokenName] {
			if value, ok := allowedValue(schema, p.TokenValue); ok {
				forValue[strings.ToLower(value)] = append(forValue[strings.ToLower(value)], ids...)
			}
		}
		for _, c := range p.When.Config {
			if strings.EqualFold(c.Field, schema.Key) {
				noteConditionValues(schema, c, ids, forValue, &auto)
			}
		}
	}
	return forValue, auto
}

func noteConditionValues(schema cp.Schema, c cp.Config, ids []string, forValue map[string][]string, auto *[]string) {
	for _, conditionValue := range c.Values {
		if value, ok := allowedValue(schema, conditionValue); ok {
			forValue[strings.ToLower(value)] = append(forValue[strings.ToLower(value)], ids...)
		}
		for _, v := range c.Values {
			cp.AddSettingValue(auto, v)
		}
	}
}

func variantHint(packMod cp.Mod, schema cp.Schema, config map[string]string, present map[string]bool, byID map[string]cp.Mod, forValue map[string][]string, auto []string) (cp.Hint, bool) {
	current, set := config[strings.ToLower(schema.Key)]
	if !set {
		current = schema.DefaultValue
	}
	current = strings.TrimSpace(current)
	if (current == "" && schema.AllowBlank) || slices.ContainsFunc(auto, func(a string) bool { return strings.EqualFold(a, current) }) {
		return cp.Hint{}, false
	}
	var enabledFor []cp.Mod
	var enabledValues []string
	for _, value := range schema.AllowValues {
		for _, id := range forValue[strings.ToLower(value)] {
			if present[id] {
				enabledFor = append(enabledFor, byID[id])
				cp.AddSettingValue(&enabledValues, value)
			}
		}
	}
	currentFor := ""
	for _, id := range forValue[strings.ToLower(current)] {
		if present[id] {
			currentFor = ""
			break
		}
		currentFor = cp.ModID(id).Local()
	}
	if forValue[strings.ToLower(current)] != nil && currentFor == "" {
		return cp.Hint{}, false // the current value is for a mod the profile has
	}
	if currentFor == "" && len(enabledFor) == 0 {
		return cp.Hint{}, false // a generic choice and none of the mapped mods: nothing to match
	}
	var suggested []string
	switch {
	case schema.AllowBlank:
		suggested = []string{""}
	case len(auto) > 0:
		suggested = auto
	default:
		suggested = enabledValues
	}
	if len(suggested) == 0 {
		return cp.Hint{}, false
	}
	hint := cp.Hint{
		Key: packMod.Key, ID: packMod.UniqueID, Name: packMod.Name, Field: schema.Key, Current: current,
		Suggested: suggested, Description: schema.Description, Variant: true, CurrentFor: currentFor,
	}
	for _, im := range enabledFor {
		hint.For = append(hint.For, im.UniqueID)
		hint.ForNames = append(hint.ForNames, cmp.Or(im.Name, im.UniqueID.Local()))
	}
	return hint, true
}

func allowedValue(schema cp.Schema, value string) (string, bool) {
	i := slices.IndexFunc(schema.AllowValues, func(v string) bool { return strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(value)) })
	if i < 0 {
		return "", false
	}
	return schema.AllowValues[i], true
}

// configHolds reports whether the pack's current settings (config.json, else the schema Default) meet every
// config condition, so a patch the player switched off is not counted as a conflict.
