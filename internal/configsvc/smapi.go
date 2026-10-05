package configsvc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
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
// captured options. Either of the last two may be empty.
func jsonSchema(file ConfigFile, current, shipped string, capture *gmcm.Capture) (Schema, error) {
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
	return s, nil
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
