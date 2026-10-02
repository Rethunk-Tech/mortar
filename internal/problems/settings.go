package problems

import (
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
				if !ok || len(condition.values) == 0 {
					continue
				}
				current, present := config[strings.ToLower(schema.key)]
				if !present {
					current = schema.defaultValue
				}
				groupKey := strings.ToLower(schema.key) + "\x00" + strings.Join(ids, "\x00")
				group := groups[groupKey]
				if group == nil {
					group = &settingGroup{
						pack:     packMod,
						schema:   schema,
						current:  current,
						required: required,
					}
					groups[groupKey] = group
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
