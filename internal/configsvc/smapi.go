package configsvc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/modconfig"
)

type flat struct {
	section string
	Entry
}

// flatten walks a config.json in file order. Nested objects become sections named by their dotted path; arrays are
// list entries holding compact JSON. Types come from the values.
func flatten(text string) ([]flat, error) {
	return flattenObject([]byte(text), "")
}

func flattenObject(raw []byte, section string) ([]flat, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("config.json is not an object")
	}
	var out []flat
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := keyTok.(string)
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
		if len(val) > 0 && val[0] == '{' {
			sub := key
			if section != "" {
				sub = section + "." + key
			}
			inner, err := flattenObject(val, sub)
			if err != nil {
				return nil, err
			}
			out = append(out, inner...)
			continue
		}
		out = append(out, flat{section: section, Entry: valueEntry(key, val)})
	}
	return out, nil
}

func valueEntry(key string, val json.RawMessage) Entry {
	e := Entry{Key: key, Type: TypeString}
	switch s := strings.TrimSpace(string(val)); {
	case s == "true" || s == "false":
		e.Type, e.Value = TypeBool, s
	case s == "null":
		e.Value = ""
	case strings.HasPrefix(s, "["):
		var buf bytes.Buffer
		if json.Compact(&buf, val) == nil {
			s = buf.String()
		}
		e.Type, e.Value = TypeList, s
	case strings.HasPrefix(s, `"`):
		var str string
		if json.Unmarshal(val, &str) == nil {
			e.Value = str
		}
		if len(str) == 7 && str[0] == '#' || len(str) == 9 && str[0] == '#' {
			if _, err := strconv.ParseUint(str[1:], 16, 64); err == nil {
				e.Type = TypeColor
			}
		}
	default:
		e.Value = s
		e.Type = TypeFloat
		if _, err := strconv.ParseInt(s, 10, 64); err == nil {
			e.Type = TypeInt
		}
	}
	return e
}

// jsonSchema builds a SMAPI file's schema from its current text, the shipped config.json (defaults) and GMCM's
// captured options (either may be empty), and for a Content Patcher pack its ConfigSchema (cp), which says what its
// string-valued settings really are.
func jsonSchema(file ConfigFile, current, shipped string, capture *gmcm.Capture, cp modconfig.Schema) (Schema, error) {
	cur, err := flatten(current)
	if err != nil {
		return Schema{}, err
	}
	defaults := map[string]Entry{}
	if shipped != "" {
		if ship, err := flatten(shipped); err == nil {
			for _, f := range ship {
				defaults[path(f.section, f.Key)] = f.Entry
			}
		}
	}
	opts := gmcmOptions(capture)
	s := Schema{File: file}
	for _, f := range cur {
		e := f.Entry
		if d, ok := defaults[path(f.section, f.Key)]; ok {
			e.Default, e.HasDefault = d.Value, true
			e.Type = typeFromDefault(e, d)
		}
		if spec, ok := cp.Lookup(f.Key); ok && f.section == "" {
			applyContentField(&e, spec)
		}
		if o, ok := opts[strings.ToLower(path(f.section, f.Key))]; ok {
			applyOption(&e, o)
		} else if o, ok := opts[strings.ToLower(f.Key)]; ok {
			applyOption(&e, o)
		}
		i := slices.IndexFunc(s.Sections, func(sec Section) bool { return sec.Name == f.section })
		if i < 0 {
			s.Sections = append(s.Sections, Section{Name: f.section})
			i = len(s.Sections) - 1
		}
		s.Sections[i].Entries = append(s.Sections[i].Entries, e)
	}
	addUnsetContentFields(&s, cp)
	return s, nil
}

// applyContentField types a Content Patcher setting from its ConfigSchema entry: exactly true and false is a switch,
// other allowed values a choice (several at once when AllowMultiple), and none a free text.
func applyContentField(e *Entry, spec modconfig.Field) {
	e.Description = spec.Description
	if spec.Default != "" {
		e.Default, e.HasDefault = spec.Default, true
	}
	switch {
	case isSwitch(spec.AllowValues):
		e.Type = TypeBool
	case len(spec.AllowValues) > 0:
		e.Type, e.Values, e.Flags = TypeEnum, spec.AllowValues, spec.AllowMultiple
	default:
		e.Type = TypeString
	}
}

func isSwitch(allow []string) bool {
	if len(allow) != 2 {
		return false
	}
	a, b := strings.ToLower(allow[0]), strings.ToLower(allow[1])
	return a == "true" && b == "false" || a == "false" && b == "true"
}

// addUnsetContentFields lists the ConfigSchema settings a pack has not written to config.json yet, at their default.
func addUnsetContentFields(s *Schema, cp modconfig.Schema) {
	have := map[string]bool{}
	for _, sec := range s.Sections {
		if sec.Name == "" {
			for _, e := range sec.Entries {
				have[strings.ToLower(e.Key)] = true
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(cp)) {
		if have[strings.ToLower(name)] {
			continue
		}
		e := Entry{Key: name, Value: cp[name].Default}
		applyContentField(&e, cp[name])
		i := slices.IndexFunc(s.Sections, func(sec Section) bool { return sec.Name == "" })
		if i < 0 {
			s.Sections = append(s.Sections, Section{})
			i = len(s.Sections) - 1
		}
		s.Sections[i].Entries = append(s.Sections[i].Entries, e)
	}
}

// typeFromDefault is the type a setting shows as: the shipped default's when the current value cannot say (a JSON null,
// or text that reads as the default's number or boolean), else its own.
func typeFromDefault(cur, def Entry) string {
	if cur.Type != TypeString || def.Type == TypeString {
		return cur.Type
	}
	v := strings.TrimSpace(cur.Value)
	switch def.Type {
	case TypeBool:
		if v == "" || v == "true" || v == "false" {
			return TypeBool
		}
	case TypeInt, TypeFloat:
		if _, err := strconv.ParseFloat(v, 64); v == "" || err == nil {
			return def.Type
		}
	}
	return cur.Type
}

func path(section, key string) string {
	if section == "" {
		return key
	}
	return section + "." + key
}

func gmcmOptions(c *gmcm.Capture) map[string]gmcm.Option {
	out := map[string]gmcm.Option{}
	if c == nil {
		return out
	}
	for _, p := range c.Pages {
		for _, o := range p.Options {
			if o.FieldID != nil && *o.FieldID != "" {
				out[strings.ToLower(*o.FieldID)] = o
			}
		}
	}
	return out
}

func applyOption(e *Entry, o gmcm.Option) {
	if o.Name != "" && !strings.EqualFold(o.Name, e.Key) {
		e.Label = o.Name
	}
	e.Description = o.Tooltip
	e.Min, e.Max = o.Min, o.Max
	if len(o.Choices) > 0 {
		e.Type = TypeEnum
		for _, c := range o.Choices {
			e.Values = append(e.Values, gmcm.Text(c.Value))
		}
	}
}
