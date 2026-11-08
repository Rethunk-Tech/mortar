package cp

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-lenientjson"
	"github.com/Rethunk-Tech/go-safefs"
)

// ReadPackConfig reads a pack's config.json as field values by lower-case key; a missing or unreadable file is no
// values, so every field keeps its schema default.
func ReadPackConfig(folder string) map[string]string {
	raw, err := safefs.ReadFile(filepath.Join(folder, "config.json"))
	if err != nil {
		return map[string]string{}
	}
	var values map[string]json.RawMessage
	if json.Unmarshal(lenientjson.Clean(raw), &values) != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		if text, ok := ConfigValue(value); ok {
			out[strings.ToLower(key)] = text
		}
	}
	return out
}

// ConfigValue is a config.json value as text: a scalar, or a list joined by ", ".
func ConfigValue(raw json.RawMessage) (string, bool) {
	if value, ok := ScalarValue(raw); ok {
		return value, true
	}
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return "", false
	}
	var out []string
	for _, value := range values {
		text, ok := ScalarValue(value)
		if !ok {
			return "", false
		}
		out = append(out, text)
	}
	return strings.Join(out, ", "), true
}

// SettingValueMatches reports whether a field's current value meets a config condition.
func SettingValueMatches(current string, condition Config) bool {
	if condition.AllowMultiple {
		for _, value := range SplitTargets(current) {
			if slices.ContainsFunc(condition.Values, func(want string) bool {
				return strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(want))
			}) {
				return true
			}
		}
		return false
	}
	return slices.ContainsFunc(condition.Values, func(value string) bool {
		return strings.EqualFold(strings.TrimSpace(current), strings.TrimSpace(value))
	})
}

// AddSettingValue appends a suggested value unless it is blank or already listed, ignoring case.
func AddSettingValue(values *[]string, value string) {
	value = strings.TrimSpace(value)
	if value == "" || slices.ContainsFunc(*values, func(existing string) bool {
		return strings.EqualFold(existing, value)
	}) {
		return
	}
	*values = append(*values, value)
}

// ConfigHolds reports whether the pack's current settings (config.json, else the schema Default) meet every
// config condition, so a patch the player switched off is not counted as a conflict.
func ConfigHolds(conditions []Config, schema map[string]Schema, config map[string]string) bool {
	for _, c := range conditions {
		field, ok := schema[strings.ToLower(c.Field)]
		if !ok {
			continue
		}
		current, set := config[strings.ToLower(field.Key)]
		if !set {
			current = field.DefaultValue
		}
		if !SettingValueMatches(current, c) {
			return false
		}
	}
	return true
}

// Toggle reports a single-choice field whose values can select which compatibility patch runs.
func (s Schema) Toggle() bool {
	values := s.AllowValues
	if len(values) == 0 {
		values = []string{s.DefaultValue}
	}
	if s.AllowMultiple || len(values) == 0 {
		return false
	}
	if len(s.AllowValues) > 0 {
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

// BooleanToggle reports a two-valued on/off field.
func (s Schema) BooleanToggle() bool {
	values := s.AllowValues
	if len(values) == 0 {
		values = []string{s.DefaultValue}
	}
	if s.AllowMultiple || len(values) > 2 {
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
