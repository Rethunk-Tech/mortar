package devsvc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// A template is a .NET template (.template.config/template.json, the format `dotnet new` reads) with Mortar's host
// file beside it (.template.config/mortar.host.json), which marks the parameters Mortar remembers per player or fills
// itself and how each is checked.
const (
	templateFile = ".template.config/template.json"
	hostFile     = ".template.config/mortar.host.json"
	// Fills Mortar knows: the repository it created, and the newest loader pack.
	fillRepositoryURL     = "repositoryUrl"
	fillLoaderPackVersion = "loaderPackVersion"
	maxTemplateJSON       = 1 << 20
)

var (
	shortName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	paramID   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
)

// Choice is one value a choice parameter takes.
type Choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Param is one value the wizard asks for, in the template's words.
type Param struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	// Kind is "string" or "choice".
	Kind      string   `json:"kind"`
	Default   string   `json:"default"`
	Required  bool     `json:"required"`
	Multiple  bool     `json:"multiple"`
	Choices   []Choice `json:"choices"`
	Remember  bool     `json:"remember"`
	Pattern   string   `json:"pattern"`
	MaxLength int      `json:"maxLength"`
	fill      string
}

// Form is a template's name and the parameters the wizard asks for, in the template's order.
type Form struct {
	Name       string  `json:"name"`
	Parameters []Param `json:"parameters"`
	shortName  string
	all        []Param
}

type templateJSON struct {
	Name       string          `json:"name"`
	ShortName  json.RawMessage `json:"shortName"`
	SourceName string          `json:"sourceName"`
	Symbols    json.RawMessage `json:"symbols"`
}

type symbolJSON struct {
	Type                string `json:"type"`
	Datatype            string `json:"datatype"`
	Description         string `json:"description"`
	DefaultValue        string `json:"defaultValue"`
	Replaces            string `json:"replaces"`
	IsRequired          bool   `json:"isRequired"`
	AllowMultipleValues bool   `json:"allowMultipleValues"`
	Choices             []struct {
		Choice      string `json:"choice"`
		DisplayName string `json:"displayName"`
	} `json:"choices"`
}

type hostJSON struct {
	SymbolInfo []struct {
		ID        string `json:"id"`
		Remember  bool   `json:"remember"`
		Fill      string `json:"fill"`
		Pattern   string `json:"pattern"`
		MaxLength int    `json:"maxLength"`
	} `json:"symbolInfo"`
}

func bad(format string, args ...any) error {
	return usererr.Wrap(usererr.Damaged, fmt.Errorf("mod template: "+format, args...))
}

// ReadForm reads the template in dir. Its placeholders (the source name and every symbol's replaces) must each be
// unique ignoring case, none inside another, so one value can never land inside another's text.
func ReadForm(dir string) (Form, error) {
	body, err := fsx.ReadFile(filepath.Join(dir, filepath.FromSlash(templateFile)))
	if err != nil || len(body) > maxTemplateJSON {
		return Form{}, bad("no %s", templateFile)
	}
	var t templateJSON
	if err := json.Unmarshal(body, &t); err != nil {
		return Form{}, bad("%s: %v", templateFile, err)
	}
	f := Form{Name: t.Name}
	// shortName may be one name or a list; the first one instantiates it.
	var names []string
	if json.Unmarshal(t.ShortName, &f.shortName) != nil && json.Unmarshal(t.ShortName, &names) == nil && len(names) > 0 {
		f.shortName = names[0]
	}
	if !shortName.MatchString(f.shortName) || t.SourceName == "" {
		return Form{}, bad("needs a shortName and a sourceName")
	}
	ids, symbols, err := orderedSymbols(t.Symbols)
	if err != nil {
		return Form{}, bad("symbols: %v", err)
	}
	tokens := []string{t.SourceName}
	for _, id := range ids {
		sym := symbols[id]
		if sym.Replaces != "" {
			tokens = append(tokens, sym.Replaces)
		}
		if sym.Type != "parameter" {
			continue
		}
		if !paramID.MatchString(id) {
			return Form{}, bad("parameter %q", id)
		}
		p := Param{ID: id, Description: sym.Description, Kind: "string", Default: sym.DefaultValue, Required: sym.IsRequired, Multiple: sym.AllowMultipleValues}
		if strings.EqualFold(sym.Datatype, "choice") {
			p.Kind = "choice"
			for _, c := range sym.Choices {
				p.Choices = append(p.Choices, Choice{Value: c.Choice, Label: cmpOr(c.DisplayName, c.Choice)})
			}
		}
		f.all = append(f.all, p)
	}
	if err := distinct(tokens); err != nil {
		return Form{}, err
	}
	if err := applyHost(dir, f.all); err != nil {
		return Form{}, err
	}
	for _, p := range f.all {
		if p.fill == "" {
			f.Parameters = append(f.Parameters, p)
		}
	}
	return f, nil
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// applyHost sets what Mortar's host file says about each parameter in all.
func applyHost(dir string, all []Param) error {
	body, err := fsx.ReadFile(filepath.Join(dir, filepath.FromSlash(hostFile)))
	if errors.Is(err, fs.ErrNotExist) {
		// A template without Mortar's host file asks for every parameter.
		return nil
	}
	if err != nil {
		return err
	}
	var h hostJSON
	if err := json.Unmarshal(body, &h); err != nil {
		return bad("%s: %v", hostFile, err)
	}
	for _, info := range h.SymbolInfo {
		i := slices.IndexFunc(all, func(p Param) bool { return p.ID == info.ID })
		if i < 0 {
			return bad("%s names %q, which is not a parameter", hostFile, info.ID)
		}
		if info.Pattern != "" {
			if _, err := regexp.Compile(info.Pattern); err != nil {
				return bad("%s: pattern for %q: %v", hostFile, info.ID, err)
			}
		}
		all[i].Remember, all[i].fill, all[i].Pattern, all[i].MaxLength = info.Remember, info.Fill, info.Pattern, info.MaxLength
	}
	return nil
}

// distinct refuses placeholders where one, ignoring case, holds another.
func distinct(tokens []string) error {
	for i, a := range tokens {
		for j, b := range tokens {
			if i != j && strings.Contains(strings.ToLower(a), strings.ToLower(b)) {
				return bad("placeholder %q holds %q; each must be unique ignoring case", a, b)
			}
		}
	}
	return nil
}

// orderedSymbols decodes the symbols object keeping its key order, which is the order the wizard asks in.
func orderedSymbols(raw json.RawMessage) ([]string, map[string]symbolJSON, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, nil, errors.New("not an object")
	}
	var ids []string
	out := map[string]symbolJSON{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		id, _ := tok.(string)
		var s symbolJSON
		if err := dec.Decode(&s); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", id, err)
		}
		ids = append(ids, id)
		out[id] = s
	}
	return ids, out, nil
}

// args are `dotnet new`'s options for values: each asked parameter checked against the template's rules, and the
// fills Mortar knows. A value never starts with "-", so it cannot be read as an option.
func (f Form) args(values, fills map[string]string) ([]string, map[string]string, error) {
	var out []string
	used := map[string]string{}
	invalid := func(format string, a ...any) error { return usererr.Wrap(usererr.Invalid, fmt.Errorf(format, a...)) }
	for _, p := range f.all {
		v, ok := values[p.ID]
		if p.fill != "" {
			v, ok = fills[p.fill]
		}
		v = strings.TrimSpace(v)
		if !ok || v == "" {
			if p.Required && p.fill == "" {
				return nil, nil, invalid("fill in %s", cmpOr(p.Description, p.ID))
			}
			continue
		}
		if strings.HasPrefix(v, "-") || strings.ContainsAny(v, "\r\n\x00") {
			return nil, nil, invalid("%s cannot start with a dash or span lines", cmpOr(p.Description, p.ID))
		}
		if p.MaxLength > 0 && len([]rune(v)) > p.MaxLength {
			return nil, nil, invalid("%s takes at most %d characters", cmpOr(p.Description, p.ID), p.MaxLength)
		}
		if p.Pattern != "" && !regexp.MustCompile(p.Pattern).MatchString(v) {
			return nil, nil, invalid("%s is not in the form the template needs", cmpOr(p.Description, p.ID))
		}
		parts := []string{v}
		if p.Multiple {
			parts = strings.Split(v, "|")
		}
		for _, part := range parts {
			if p.Kind == "choice" && !slices.ContainsFunc(p.Choices, func(c Choice) bool { return c.Value == part }) {
				return nil, nil, invalid("%q is not a choice for %s", part, cmpOr(p.Description, p.ID))
			}
			out = append(out, "--"+p.ID, part)
		}
		used[p.ID] = v
	}
	return out, used, nil
}
