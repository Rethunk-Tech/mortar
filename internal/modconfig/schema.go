// Package modconfig parses Content Patcher ConfigSchema and validates field values.
package modconfig

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
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

// Entry is one ConfigSchema entry as written, before a caller applies its own reading of the values.
type Entry struct {
	Default       json.RawMessage `json:"Default"`
	AllowValues   string          `json:"AllowValues"`
	AllowMultiple bool            `json:"AllowMultiple"`
	AllowBlank    bool            `json:"AllowBlank"`
	Description   string          `json:"Description"`
	Section       json.RawMessage `json:"Section"`
}

// Entries reads ConfigSchema from a content.json document, keyed by the raw field name. An entry that is not an
// object of the expected shape is left out.
func Entries(contentJSON []byte) (map[string]Entry, error) {
	var doc struct {
		ConfigSchema map[string]json.RawMessage `json:"ConfigSchema"`
	}
	if err := json.Unmarshal(contentJSON, &doc); err != nil {
		return nil, fmt.Errorf("content.json: %w", err)
	}
	out := make(map[string]Entry, len(doc.ConfigSchema))
	for name, raw := range doc.ConfigSchema {
		var e Entry
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		out[name] = e
	}
	return out, nil
}

// Parse reads ConfigSchema from a content.json document.
func Parse(contentJSON []byte) (Schema, error) {
	entries, err := Entries(contentJSON)
	if err != nil {
		return nil, err
	}
	out := Schema{}
	for name, e := range entries {
		var section string
		if len(e.Section) > 0 && json.Unmarshal(e.Section, &section) != nil {
			continue
		}
		out[name] = Field{
			Name:          name,
			AllowValues:   splitCSV(e.AllowValues),
			AllowMultiple: e.AllowMultiple,
			AllowBlank:    e.AllowBlank,
			Default:       rawString(e.Default),
			Description:   e.Description,
			Section:       section,
		}
	}
	return out, nil
}

// IsContentPack reports a Pathoschild.ContentPatcher content pack.
func IsContentPack(manifestJSON []byte) bool {
	m, err := smapimanifest.Parse(manifestJSON)
	return err == nil && strings.EqualFold(m.ContentPackFor, ContentPatcherID)
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
	if _, after, ok := strings.CutLast(path, "."); ok {
		leaf = after
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
