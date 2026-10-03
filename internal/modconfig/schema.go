// Package modconfig parses Content Patcher ConfigSchema and validates field values.
package modconfig

import (
	"encoding/json"
	"fmt"
	"strings"
)

const ContentPatcherID = "Pathoschild.ContentPatcher"

// Field is one Content Patcher ConfigSchema entry.
type Field struct {
	Name          string
	AllowValues   []string
	AllowMultiple bool
	AllowBlank    bool
	Default       string
	Description   string
	Section       string
}

// Schema is ConfigSchema keyed by field name.
type Schema map[string]Field

// Parse reads ConfigSchema from a content.json document.
func Parse(contentJSON []byte) (Schema, error) {
	var doc struct {
		ConfigSchema map[string]json.RawMessage `json:"ConfigSchema"`
	}
	if err := json.Unmarshal(contentJSON, &doc); err != nil {
		return nil, fmt.Errorf("content.json: %w", err)
	}
	out := Schema{}
	for name, raw := range doc.ConfigSchema {
		var spec struct {
			AllowValues   string          `json:"AllowValues"`
			AllowMultiple bool            `json:"AllowMultiple"`
			AllowBlank    bool            `json:"AllowBlank"`
			Default       json.RawMessage `json:"Default"`
			Description   string          `json:"Description"`
			Section       string          `json:"Section"`
		}
		if json.Unmarshal(raw, &spec) != nil {
			continue
		}
		out[name] = Field{
			Name:          name,
			AllowValues:   splitCSV(spec.AllowValues),
			AllowMultiple: spec.AllowMultiple,
			AllowBlank:    spec.AllowBlank,
			Default:       rawString(spec.Default),
			Description:   spec.Description,
			Section:       spec.Section,
		}
	}
	return out, nil
}

// IsContentPack reports a Pathoschild.ContentPatcher content pack.
func IsContentPack(manifestJSON []byte) bool {
	var doc struct {
		ContentPackFor *struct {
			UniqueID string `json:"UniqueID"`
		} `json:"ContentPackFor"`
	}
	if json.Unmarshal(manifestJSON, &doc) != nil || doc.ContentPackFor == nil {
		return false
	}
	return strings.EqualFold(doc.ContentPackFor.UniqueID, ContentPatcherID)
}

// Lookup finds a field by name, ignoring case.
func (s Schema) Lookup(name string) (Field, bool) {
	if s == nil {
		return Field{}, false
	}
	if f, ok := s[name]; ok {
		return f, true
	}
	for k, f := range s {
		if strings.EqualFold(k, name) {
			return f, true
		}
	}
	return Field{}, false
}

// Validate checks a set value against AllowValues when the field has them.
func (s Schema) Validate(path, value string) error {
	leaf := path
	if i := strings.LastIndex(path, "."); i >= 0 {
		leaf = path[i+1:]
	}
	f, ok := s.Lookup(leaf)
	if !ok || len(f.AllowValues) == 0 {
		return nil
	}
	if strings.TrimSpace(value) == "" {
		if f.AllowBlank {
			return nil
		}
		return fmt.Errorf("config field %s cannot be blank", path)
	}
	parts := []string{strings.TrimSpace(value)}
	if f.AllowMultiple {
		parts = splitCSV(value)
	}
	allowed := map[string]string{}
	for _, v := range f.AllowValues {
		allowed[strings.ToLower(v)] = v
	}
	for _, p := range parts {
		if _, ok := allowed[strings.ToLower(p)]; !ok {
			return fmt.Errorf("config field %s does not allow %q (allowed: %s)", path, p, strings.Join(f.AllowValues, ", "))
		}
	}
	return nil
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return fmt.Sprint(b)
	}
	return strings.TrimSpace(string(raw))
}

// IsBooleanField reports a schema entry whose default or allowed values are booleans.
func IsBooleanField(f Field) bool {
	if f.Default == "true" || f.Default == "false" {
		return true
	}
	if len(f.AllowValues) == 0 {
		return false
	}
	for _, v := range f.AllowValues {
		low := strings.ToLower(v)
		if low != "true" && low != "false" {
			return false
		}
	}
	return true
}
