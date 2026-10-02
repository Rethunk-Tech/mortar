package problems

import (
	"cmp"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

type settingGroup struct {
	pack      Installed
	schema    cpSchema
	current   string
	required  []Installed
	suggested []string
	active    bool
}

func compatibilitySettings(mods []Installed) []SettingHint {
	present := map[string]bool{}
	byID := map[string]Installed{}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		id := strings.ToLower(strings.TrimSpace(mod.UniqueID))
		if id == "" {
			continue
		}
		present[id] = true
		if _, exists := byID[id]; !exists {
			byID[id] = mod
		}
	}

	var out []SettingHint
	for _, packMod := range mods {
		if !packMod.Enabled {
			continue
		}
		pack := readContentPack(packMod)
		if len(pack.schema) == 0 {
			continue
		}
		config := readPackConfig(packMod.Folder)
		out = append(out, variantSettings(packMod, pack, config, present, byID)...)
		groups := map[string]*settingGroup{}
		for _, patch := range pack.patches {
			if !patch.when.holds(present) {
				continue
			}
			ids := enabledRequirements(patch.when, present, packMod.UniqueID)
			if len(ids) == 0 {
				continue
			}
			required := make([]Installed, 0, len(ids))
			for _, id := range ids {
				required = append(required, byID[id])
			}
			for _, condition := range patch.when.config {
				schema, ok := pack.schema[strings.ToLower(condition.field)]
				if !ok || len(condition.values) == 0 || !schema.booleanToggle() {
					continue
				}
				current, present := config[strings.ToLower(schema.key)]
				if !present {
					current = schema.defaultValue
				}
				// One group per field: a value that already runs a patch for any installed mod is the player's
				// choice, even when other patches for other mod combinations need a different value.
				groupKey := strings.ToLower(schema.key)
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
					for _, mod := range required {
						if !slices.ContainsFunc(group.required, func(m Installed) bool { return sameID(m.UniqueID, mod.UniqueID) }) {
							group.required = append(group.required, mod)
						}
					}
				}
				for _, value := range condition.values {
					addSettingValue(&group.suggested, value)
				}
				if settingValueMatches(current, condition) {
					group.active = true
				}
			}
		}
		for _, group := range groups {
			if group.active || len(group.suggested) == 0 {
				continue
			}
			hint := SettingHint{
				Key:         group.pack.Key,
				UniqueID:    group.pack.UniqueID,
				Name:        group.pack.Name,
				Field:       group.schema.key,
				Current:     group.current,
				Suggested:   group.suggested,
				Description: group.schema.description,
			}
			for _, mod := range group.required {
				hint.For = append(hint.For, mod.UniqueID)
				name := mod.Name
				if name == "" {
					name = mod.UniqueID
				}
				hint.ForNames = append(hint.ForNames, name)
			}
			out = append(out, hint)
		}
	}
	slices.SortFunc(out, func(a, b SettingHint) int {
		if c := strings.Compare(strings.ToLower(a.Key), strings.ToLower(b.Key)); c != 0 {
			return c
		}
		if c := strings.Compare(strings.ToLower(a.UniqueID), strings.ToLower(b.UniqueID)); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.Field), strings.ToLower(b.Field))
	})
	return out
}

func enabledRequirements(when cpWhen, present map[string]bool, own string) []string {
	ids := map[string]bool{}
	for _, group := range when.anyOf {
		for _, id := range group {
			id = strings.ToLower(strings.TrimSpace(id))
			if id != "" && !sameID(id, own) && present[id] {
				ids[id] = true
			}
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func readPackConfig(folder string) map[string]string {
	raw, err := fsx.ReadFile(filepath.Join(folder, "config.json"))
	if err != nil {
		return map[string]string{}
	}
	var values map[string]json.RawMessage
	if json.Unmarshal(stripJSONNoise(raw), &values) != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		if text, ok := configValue(value); ok {
			out[strings.ToLower(key)] = text
		}
	}
	return out
}

func configValue(raw json.RawMessage) (string, bool) {
	if value, ok := scalarValue(raw); ok {
		return value, true
	}
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return "", false
	}
	var out []string
	for _, value := range values {
		text, ok := scalarValue(value)
		if !ok {
			return "", false
		}
		out = append(out, text)
	}
	return strings.Join(out, ", "), true
}

func settingValueMatches(current string, condition cpConfig) bool {
	if condition.allowMultiple {
		for _, value := range splitTargets(current) {
			if slices.ContainsFunc(condition.values, func(want string) bool {
				return strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(want))
			}) {
				return true
			}
		}
		return false
	}
	return slices.ContainsFunc(condition.values, func(value string) bool {
		return strings.EqualFold(strings.TrimSpace(current), strings.TrimSpace(value))
	})
}

func addSettingValue(values *[]string, value string) {
	value = strings.TrimSpace(value)
	if value == "" || slices.ContainsFunc(*values, func(existing string) bool {
		return strings.EqualFold(existing, value)
	}) {
		return
	}
	*values = append(*values, value)
}

// toggle reports a single-choice field whose values can select which compatibility patch runs.
func (s cpSchema) toggle() bool {
	values := s.allowValues
	if len(values) == 0 {
		values = []string{s.defaultValue}
	}
	if s.allowMultiple || len(values) == 0 {
		return false
	}
	if len(s.allowValues) > 0 {
		return true
	}
	for _, v := range values {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "false", "on", "off", "enabled", "disabled", "yes", "no":
		default:
			return false
		}
	}
	return true
}

func (s cpSchema) booleanToggle() bool {
	values := s.allowValues
	if len(values) == 0 {
		values = []string{s.defaultValue}
	}
	if s.allowMultiple || len(values) > 2 {
		return false
	}
	for _, v := range values {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "false", "on", "off", "enabled", "disabled", "yes", "no":
		default:
			return false
		}
	}
	return true
}

// variantSettings suggests a picker value from the pack's own mapping of values to mods: a dynamic token that
// yields one of the field's allowed values only when a mod is installed (Value "Earthy" When HasMod
// DaisyNiko.EarthyRecolour). A field with two or more such values is a picker. Its automatic choice is a
// blank value when allowed, or the value the mapping tokens themselves require of the field ("Off"), since
// with it the pack detects the installed mod on its own.
func variantSettings(packMod Installed, pack cachedPack, config map[string]string, present map[string]bool, byID map[string]Installed) []SettingHint {
	var out []SettingHint
	for _, schema := range pack.schema {
		if schema.allowMultiple || len(schema.allowValues) < 2 {
			continue
		}
		// Only the field's own tokens count: those that read it ({{Field}}) or test it. Other tokens can yield
		// the same words ("Default") for unrelated choices.
		own := map[string]bool{}
		for _, p := range pack.patches {
			if p.tokenName == "" {
				continue
			}
			reads := strings.Contains(strings.ToLower(p.tokenValue), "{{"+strings.ToLower(schema.key)+"}}")
			if reads || slices.ContainsFunc(p.when.config, func(c cpConfig) bool { return strings.EqualFold(c.field, schema.key) }) {
				own[p.tokenName] = true
			}
		}
		forValue := map[string][]string{} // lower-cased allowed value -> mod ids that select it
		var auto []string
		for _, p := range pack.patches {
			usesField := slices.ContainsFunc(p.when.config, func(c cpConfig) bool {
				return strings.EqualFold(c.field, schema.key)
			})
			if !own[p.tokenName] && !usesField {
				continue
			}
			ids := make([]string, 0)
			for _, group := range p.when.anyOf {
				for _, id := range group {
					if !sameID(id, packMod.UniqueID) {
						ids = append(ids, strings.ToLower(id))
					}
				}
			}
			if len(ids) == 0 {
				continue
			}
			if own[p.tokenName] {
				if value, ok := allowedValue(schema, p.tokenValue); ok {
					forValue[strings.ToLower(value)] = append(forValue[strings.ToLower(value)], ids...)
				}
			}
			for _, c := range p.when.config {
				if !strings.EqualFold(c.field, schema.key) {
					continue
				}
				for _, conditionValue := range c.values {
					if value, ok := allowedValue(schema, conditionValue); ok {
						forValue[strings.ToLower(value)] = append(forValue[strings.ToLower(value)], ids...)
					}
					for _, v := range c.values {
						addSettingValue(&auto, v)
					}
				}
			}
		}
		if len(forValue) < 2 {
			continue
		}
		current, set := config[strings.ToLower(schema.key)]
		if !set {
			current = schema.defaultValue
		}
		current = strings.TrimSpace(current)
		if (current == "" && schema.allowBlank) || slices.ContainsFunc(auto, func(a string) bool { return strings.EqualFold(a, current) }) {
			continue
		}
		var enabledFor []Installed
		var enabledValues []string
		for _, value := range schema.allowValues {
			for _, id := range forValue[strings.ToLower(value)] {
				if present[id] {
					enabledFor = append(enabledFor, byID[id])
					addSettingValue(&enabledValues, value)
				}
			}
		}
		currentFor := ""
		for _, id := range forValue[strings.ToLower(current)] {
			if present[id] {
				currentFor = ""
				break
			}
			currentFor = id
		}
		if forValue[strings.ToLower(current)] != nil && currentFor == "" {
			continue // the current value is for a mod the profile has
		}
		if currentFor == "" && len(enabledFor) == 0 {
			continue // a generic choice and none of the mapped mods: nothing to match
		}
		var suggested []string
		switch {
		case schema.allowBlank:
			suggested = []string{""}
		case len(auto) > 0:
			suggested = auto
		default:
			suggested = enabledValues
		}
		if len(suggested) == 0 {
			continue
		}
		hint := SettingHint{
			Key: packMod.Key, UniqueID: packMod.UniqueID, Name: packMod.Name, Field: schema.key, Current: current,
			Suggested: suggested, Description: schema.description, Variant: true, CurrentFor: currentFor,
		}
		for _, mod := range enabledFor {
			hint.For = append(hint.For, mod.UniqueID)
			hint.ForNames = append(hint.ForNames, cmp.Or(mod.Name, mod.UniqueID))
		}
		out = append(out, hint)
	}
	return out
}

func allowedValue(schema cpSchema, value string) (string, bool) {
	i := slices.IndexFunc(schema.allowValues, func(v string) bool { return strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(value)) })
	if i < 0 {
		return "", false
	}
	return schema.allowValues[i], true
}

// configHolds reports whether the pack's current settings (config.json, else the schema Default) meet every
// config condition, so a patch the player switched off is not counted as a conflict.
func configHolds(conditions []cpConfig, schema map[string]cpSchema, config map[string]string) bool {
	for _, c := range conditions {
		field, ok := schema[strings.ToLower(c.field)]
		if !ok {
			continue
		}
		current, set := config[strings.ToLower(field.key)]
		if !set {
			current = field.defaultValue
		}
		if !settingValueMatches(current, c) {
			return false
		}
	}
	return true
}
