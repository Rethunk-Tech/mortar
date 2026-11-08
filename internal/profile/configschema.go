package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-cfgedit/jsonfile"
	"github.com/Rethunk-Tech/go-lenientjson"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
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
		if _, tail, ok := strings.CutLast(leaf, "."); ok {
			leaf = tail
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
	return s.SetConfigValues(game, id, key, uniqueID, []ConfigSet{{Field: field, Value: value}})
}

// ConfigSet is one field of a config.json and the value to give it.
type ConfigSet struct{ Field, Value string }

// SetConfigValues changes several settings of one config.json in a single write and a single history event; nothing
// is written when any value is refused.
func (s *Service) SetConfigValues(game, id, key string, uniqueID mod.ID, sets []ConfigSet) error {
	folder, err := s.store.ModFolder(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	path := filepath.Join(folder, configFile)
	raw, found, err := safefs.ReadFileIfExists(path)
	if err != nil {
		return err
	}
	if !found {
		raw = []byte("{}")
	}
	schema, err := s.contentSchema(game, id, key, uniqueID)
	if err != nil {
		return err
	}
	types := schemaFromDir(folder)
	shipped, _ := s.store.ShippedConfig(game, id, key, uniqueID)
	for _, c := range sets {
		field := strings.TrimSpace(c.Field)
		if field == "" {
			return errors.New("missing config field")
		}
		if err := schema.Validate(field, c.Value); err != nil {
			return err
		}
		if raw, err = applyConfigSet(raw, field, c.Value, types, shipped); err != nil {
			return err
		}
	}
	return s.store.WriteConfig(game, id, key, uniqueID, string(raw))
}

func (s *Service) contentSchema(game, id, key string, uniqueID mod.ID) (modconfig.Schema, error) {
	return s.store.ContentSchema(game, id, key, uniqueID)
}

// ContentSchema is the Content Patcher ConfigSchema of a content pack, or an empty schema for any other mod.
func (s *Store) ContentSchema(game, id, key string, uniqueID mod.ID) (modconfig.Schema, error) {
	folder, err := s.ModFolder(game, id, key, uniqueID)
	if err != nil {
		return nil, err
	}
	man, found, err := safefs.ReadFileIfExists(filepath.Join(folder, smapimanifest.FileName))
	if err != nil {
		return nil, err
	}
	if !found {
		return modconfig.Schema{}, nil
	}
	if !modconfig.IsContentPack(man) {
		return modconfig.Schema{}, nil
	}
	return schemaFromDir(folder), nil
}

func schemaFromDir(folder string) modconfig.Schema {
	raw, err := safefs.ReadFile(filepath.Join(folder, "content.json"))
	if err != nil {
		return modconfig.Schema{}
	}
	schema, err := modconfig.Parse(lenientjson.Clean(raw))
	if err != nil {
		return modconfig.Schema{}
	}
	return schema
}

func decodeConfigBytes(raw []byte) (jsonfile.Node, error) {
	node, err := jsonfile.Parse(raw)
	if err != nil {
		return jsonfile.Node{}, fmt.Errorf("config.json is not valid JSON: %w", err)
	}
	return node, nil
}

func rewriteConfigJSON(data []byte) ([]byte, error) {
	node, err := jsonfile.Parse(data)
	if err != nil {
		if err.Error() == "trailing data after the JSON value" {
			return nil, fmt.Errorf("config.json has trailing data")
		}
		return nil, err
	}
	return node.Marshal()
}

// SetJSONValue writes one value at a dotted field path of a JSON config file the mod writes itself, keeping its key
// order.
func SetJSONValue(raw []byte, field, value string) ([]byte, error) {
	return applyConfigSet(raw, field, value, modconfig.Schema{}, "")
}

func configJSONError(err error, field, key string) error {
	switch {
	case errors.Is(err, jsonfile.ErrBadPath):
		return fmt.Errorf("invalid config field %s", field)
	case errors.Is(err, jsonfile.ErrUnknownKey):
		return fmt.Errorf("unknown config field %s", key)
	case errors.Is(err, jsonfile.ErrNotObject):
		return fmt.Errorf("%s is not an object", key)
	case errors.Is(err, jsonfile.ErrNotBool):
		return fmt.Errorf("config field %s expects a boolean", key)
	case errors.Is(err, jsonfile.ErrNotNumber):
		return fmt.Errorf("config field %s expects a number", key)
	default:
		return err
	}
}

// applyConfigSet writes one value. shipped is the mod's own config.json, whose value kinds type a setting that is
// null in the file.
func applyConfigSet(raw []byte, field, value string, schema modconfig.Schema, shipped string) ([]byte, error) {
	node, err := decodeConfigBytes(raw)
	if err != nil {
		return nil, err
	}
	var defaults jsonfile.Node
	if shipped != "" {
		defaults, _ = decodeConfigBytes([]byte(shipped))
	}
	parts, err := jsonfile.SplitPath(field)
	if err != nil {
		return nil, configJSONError(err, field, "")
	}
	def := defaultKind(defaults, parts)
	existing, found := node.Lookup(parts)
	key := parts[len(parts)-1]
	if !found {
		if f, ok := schema.Lookup(key); ok {
			key = f.Name
			parts[len(parts)-1] = key
		}
	}
	// A Content Patcher field stays text when null; any other null takes the shipped default's kind.
	kind := existing.Kind
	if kind == jsonfile.Null {
		if _, cp := schema.Lookup(key); !cp {
			kind = def
		}
	}
	child, err := jsonfile.Coerce(value, kind)
	if err != nil {
		return nil, configJSONError(err, field, key)
	}
	next, err := jsonfile.Set(node, parts, child)
	if err != nil {
		head := parts[0]
		switch {
		case errors.Is(err, jsonfile.ErrUnknownKey):
			head = strings.TrimPrefix(err.Error(), jsonfile.ErrUnknownKey.Error()+" ")
		case errors.Is(err, jsonfile.ErrNotObject):
			head, _, _ = strings.Cut(err.Error(), " ")
		}
		return nil, configJSONError(err, field, head)
	}
	return next.Marshal()
}

// defaultKind is the JSON kind of the value at parts in the shipped config, or Null when it has none.
func defaultKind(n jsonfile.Node, parts []string) jsonfile.Kind {
	found, ok := n.Lookup(parts)
	if !ok {
		return jsonfile.Null
	}
	return found.Kind
}

func listFields(n jsonfile.Node, prefix string, schema modconfig.Schema, out *[]ConfigField) {
	if n.Kind != jsonfile.Object {
		if prefix != "" {
			*out = append(*out, schemaField(prefix, n.Text(), fieldForPath(schema, prefix)))
		}
		return
	}
	for _, p := range n.Obj {
		path := p.Key
		if prefix != "" {
			path = prefix + "." + p.Key
		}
		if p.Value.Kind == jsonfile.Object {
			listFields(p.Value, path, schema, out)
			continue
		}
		*out = append(*out, schemaField(path, p.Value.Text(), fieldForPath(schema, path)))
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
