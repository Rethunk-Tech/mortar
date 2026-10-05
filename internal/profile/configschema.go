package profile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/jsonc"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/modconfig"
)

// ConfigField is one printed or settable config.json path.
type ConfigField struct {
	Path          string   `json:"path"`
	Value         string   `json:"value"`
	AllowValues   []string `json:"allowValues,omitempty"`
	AllowMultiple bool     `json:"allowMultiple,omitempty"`
	AllowBlank    bool     `json:"allowBlank,omitempty"`
	Default       string   `json:"default,omitempty"`
	Description   string   `json:"description,omitempty"`
	Section       string   `json:"section,omitempty"`
}

// ReadContentSchema returns the Content Patcher ConfigSchema object, or {}.
func (s *Service) ReadContentSchema(game, id, key string, uniqueID mod.ID) (string, error) {
	schema, err := s.contentSchema(game, id, key, uniqueID)
	if err != nil {
		return "", err
	}
	if len(schema) == 0 {
		return "{}", nil
	}
	raw := map[string]any{}
	for name, f := range schema {
		entry := map[string]any{}
		if len(f.AllowValues) > 0 {
			entry["AllowValues"] = strings.Join(f.AllowValues, ", ")
		}
		if f.AllowMultiple {
			entry["AllowMultiple"] = true
		}
		if f.AllowBlank {
			entry["AllowBlank"] = true
		}
		if f.Default != "" {
			entry["Default"] = f.Default
		}
		if f.Description != "" {
			entry["Description"] = f.Description
		}
		if f.Section != "" {
			entry["Section"] = f.Section
		}
		raw[name] = entry
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ListConfigFields lists config.json values and Content Patcher allowed values.
//
//wails:ignore
func (s *Service) ListConfigFields(game, id, key string, uniqueID mod.ID) ([]ConfigField, error) {
	schema, err := s.contentSchema(game, id, key, uniqueID)
	if err != nil {
		return nil, err
	}
	raw, err := s.ReadConfig(game, id, key, uniqueID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || strings.Contains(err.Error(), "missing") {
			raw = "{}"
		} else {
			return nil, err
		}
	}
	node, err := decodeConfigBytes([]byte(raw))
	if err != nil {
		return nil, err
	}
	var fields []ConfigField
	listFields(node, "", schema, &fields)
	seen := map[string]bool{}
	for _, f := range fields {
		leaf := f.Path
		if i := strings.LastIndex(leaf, "."); i >= 0 {
			leaf = leaf[i+1:]
		}
		seen[strings.ToLower(leaf)] = true
	}
	for name, spec := range schema {
		if seen[strings.ToLower(name)] {
			continue
		}
		fields = append(fields, schemaField(name, spec.Default, spec))
	}
	return fields, nil
}

// SetConfigValue changes one setting while retaining the config's other raw JSON values.
func (s *Service) SetConfigValue(game, id, key string, uniqueID mod.ID, field, value string) error {
	field = strings.TrimSpace(field)
	if field == "" {
		return errors.New("missing config field")
	}
	folder, err := s.store.ModFolder(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	path := filepath.Join(folder, configFile)
	raw, err := fsx.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		raw = []byte("{}")
	} else if err != nil {
		return err
	}
	schema, err := s.contentSchema(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	if err := schema.Validate(field, value); err != nil {
		return err
	}
	types := schemaFromDir(folder)
	next, err := applyConfigSet(raw, field, value, types)
	if err != nil {
		return err
	}
	return s.store.WriteConfig(game, id, key, uniqueID, string(next))
}

func (s *Service) contentSchema(game, id, key string, uniqueID mod.ID) (modconfig.Schema, error) {
	folder, err := s.store.ModFolder(game, id, key, uniqueID)
	if err != nil {
		return nil, err
	}
	man, err := fsx.ReadFile(filepath.Join(folder, manifest.FileName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return modconfig.Schema{}, nil
		}
		return nil, err
	}
	if !modconfig.IsContentPack(man) {
		return modconfig.Schema{}, nil
	}
	return schemaFromDir(folder), nil
}

func schemaFromDir(folder string) modconfig.Schema {
	raw, err := fsx.ReadFile(filepath.Join(folder, "content.json"))
	if err != nil {
		return modconfig.Schema{}
	}
	schema, err := modconfig.Parse(jsonc.Clean(raw))
	if err != nil {
		return modconfig.Schema{}
	}
	return schema
}

func decodeConfigBytes(raw []byte) (jsonNode, error) {
	rewritten, err := rewriteConfigJSON(raw)
	if err != nil {
		return jsonNode{}, fmt.Errorf("config.json is not valid JSON: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(rewritten))
	dec.UseNumber()
	return decodeJSON(dec)
}

func applyConfigSet(raw []byte, field, value string, schema modconfig.Schema) ([]byte, error) {
	node, err := decodeConfigBytes(raw)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(field, ".")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, fmt.Errorf("invalid config field %s", field)
		}
	}
	next, err := setJSONPath(node, parts, value, schema)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := encodeJSON(&buf, next, 0); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func setJSONPath(n jsonNode, parts []string, raw string, schema modconfig.Schema) (jsonNode, error) {
	if len(parts) == 0 {
		return coerceJSON(n, raw, schema, "")
	}
	if n.kind != jsonObj {
		return n, fmt.Errorf("%s is not an object", parts[0])
	}
	head, rest := parts[0], parts[1:]
	for i, p := range n.obj {
		if !strings.EqualFold(p.Key, head) {
			continue
		}
		var (
			child jsonNode
			err   error
		)
		if len(rest) == 0 {
			child, err = coerceJSON(p.Value, raw, schema, p.Key)
		} else {
			child, err = setJSONPath(p.Value, rest, raw, schema)
		}
		if err != nil {
			return n, err
		}
		n.obj[i].Value = child
		return n, nil
	}
	if len(rest) > 0 {
		return n, fmt.Errorf("unknown config field %s", head)
	}
	name := head
	if f, ok := schema.Lookup(head); ok {
		name = f.Name
	}
	child, err := coerceJSON(jsonNode{kind: jsonNull}, raw, schema, name)
	if err != nil {
		return n, err
	}
	n.obj = append(n.obj, jsonPair{Key: name, Value: child})
	return n, nil
}

func coerceJSON(existing jsonNode, raw string, schema modconfig.Schema, field string) (jsonNode, error) {
	f, _ := schema.Lookup(field)
	if existing.kind == jsonBool || modconfig.IsBooleanField(f) {
		parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return jsonNode{}, fmt.Errorf("config field %s expects a boolean", field)
		}
		return jsonNode{kind: jsonBool, flag: parsed}, nil
	}
	if existing.kind == jsonNum {
		if _, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err != nil {
			return jsonNode{}, fmt.Errorf("config field %s expects a number", field)
		}
		return jsonNode{kind: jsonNum, num: json.Number(strings.TrimSpace(raw))}, nil
	}
	return jsonNode{kind: jsonStr, str: raw}, nil
}

func listFields(n jsonNode, prefix string, schema modconfig.Schema, out *[]ConfigField) {
	if n.kind != jsonObj {
		if prefix != "" {
			*out = append(*out, schemaField(prefix, nodeText(n), fieldForPath(schema, prefix)))
		}
		return
	}
	for _, p := range n.obj {
		path := p.Key
		if prefix != "" {
			path = prefix + "." + p.Key
		}
		if p.Value.kind == jsonObj {
			listFields(p.Value, path, schema, out)
			continue
		}
		*out = append(*out, schemaField(path, nodeText(p.Value), fieldForPath(schema, path)))
	}
}

func fieldForPath(schema modconfig.Schema, path string) modconfig.Field {
	leaf := path
	if _, after, ok := strings.CutLast(path, "."); ok {
		leaf = after
	}
	f, _ := schema.Lookup(leaf)
	return f
}

func schemaField(path, value string, spec modconfig.Field) ConfigField {
	return ConfigField{
		Path:          path,
		Value:         value,
		AllowValues:   spec.AllowValues,
		AllowMultiple: spec.AllowMultiple,
		AllowBlank:    spec.AllowBlank,
		Default:       spec.Default,
		Description:   spec.Description,
		Section:       spec.Section,
	}
}

func nodeText(n jsonNode) string {
	switch n.kind {
	case jsonBool:
		return strconv.FormatBool(n.flag)
	case jsonNum:
		return n.num.String()
	case jsonStr:
		return n.str
	case jsonNull:
		return "null"
	default:
		var buf bytes.Buffer
		_ = encodeJSON(&buf, n, 0)
		return buf.String()
	}
}
