package configsvc

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// MelonLoader keeps every melon's settings in UserData/MelonPreferences.cfg, a TOML file (MelonLoader/Preferences,
// Tomlet): one [Category] table per melon or settings group, and per entry its description as the comment lines above
// `Key = value` and an optional inline `# comment`. The file names no types, ranges or defaults, so the type of an entry
// is read off its value.
const melonName = "MelonPreferences.cfg"

func (s *Service) melonPath(game, id string) (string, error) {
	dir, err := s.Profiles.ProfileDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "UserData", melonName), nil
}

type melonEntry struct {
	section string
	// line is where the entry starts and end where its value ends: Tomlet writes some values over several lines.
	line, end int
	// prefix is the line up to and including "=" and its following space; suffix an inline comment with its spaces.
	prefix, suffix string
	Entry
}

type melonDoc struct {
	lines     []string
	bom, crlf bool
	entries   []melonEntry
}

var (
	tomlInt   = regexp.MustCompile(`^[+-]?\d[\d_]*$`)
	tomlFloat = regexp.MustCompile(`^[+-]?(\d[\d_]*(\.\d[\d_]*)?([eE][+-]?\d+)?|inf|nan)$`)
)

func parseMelon(text string) melonDoc {
	text, bom := strings.CutPrefix(text, "\ufeff")
	d := melonDoc{bom: bom, crlf: strings.Contains(text, "\r\n")}
	d.lines = strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var section string
	var desc []string
	for i := 0; i < len(d.lines); i++ {
		raw := d.lines[i]
		line := strings.TrimSpace(raw)
		switch {
		case line == "":
			desc = nil
		case strings.HasPrefix(line, "#"):
			desc = append(desc, strings.TrimSpace(strings.TrimPrefix(line, "#")))
		case strings.HasPrefix(line, "[[") || strings.HasPrefix(line, "["):
			section, desc = strings.Trim(line, "[] \t"), nil
		default:
			eq := strings.Index(raw, "=")
			if eq < 0 {
				desc = nil
				continue
			}
			value, suffix := splitInline(strings.TrimSpace(raw[eq+1:]))
			end := valueEnd(d.lines, i, value)
			if end > i {
				value, suffix = strings.Join(append([]string{value}, d.lines[i+1:end+1]...), "\n"), ""
			}
			typ, val := melonType(value)
			e := melonEntry{
				section: section, line: i, end: end, prefix: raw[:eq+1] + " ", suffix: suffix,
				Key:         strings.Trim(strings.TrimSpace(raw[:eq]), `"'`),
				Description: strings.Join(desc, "\n"),
				Type:        typ,
				Value:       val,
			}
			d.entries = append(d.entries, e)
			desc = nil
			i = end
		}
	}
	return d
}

// splitInline separates a value from a trailing `# comment`, which only counts outside a quoted string.
func splitInline(v string) (value, comment string) {
	var quote rune
	for i, r := range v {
		switch {
		case quote != 0:
			if r == quote && (quote == '\'' || v[i-1] != '\\') {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '#':
			return strings.TrimRight(v[:i], " \t"), " " + v[i:]
		}
	}
	return v, ""
}

// valueEnd is the line a value opening on line i ends on: a multi-line string runs to its closing quotes, an array
// until its brackets close.
func valueEnd(lines []string, i int, value string) int {
	for _, q := range []string{"'''", `"""`} {
		if strings.HasPrefix(value, q) && !strings.Contains(value[len(q):], q) {
			for j := i + 1; j < len(lines); j++ {
				if strings.Contains(lines[j], q) {
					return j
				}
			}
			return len(lines) - 1
		}
	}
	depth := 0
	for j := i; j < len(lines) && strings.HasPrefix(value, "["); j++ {
		text := lines[j]
		if j == i {
			text = value
		}
		if depth += bracketDepth(text); depth <= 0 {
			return j
		}
	}
	return i
}

// bracketDepth is how many more brackets a line opens than it closes, outside strings and comments.
func bracketDepth(line string) int {
	value, _ := splitInline(strings.TrimSpace(line))
	var quote rune
	depth := 0
	for i, r := range value {
		switch {
		case quote != 0:
			if r == quote && (quote == '\'' || value[i-1] != '\\') {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '[':
			depth++
		case r == ']':
			depth--
		}
	}
	return depth
}

// melonType is the schema type of a TOML value and the text to show for it.
func melonType(v string) (typ, shown string) {
	for _, q := range []string{"'''", `"""`} {
		if inner, ok := strings.CutPrefix(v, q); ok {
			inner = strings.TrimPrefix(strings.TrimSuffix(inner, q), "\n")
			return TypeString, inner
		}
	}
	switch {
	case v == "true" || v == "false":
		return TypeBool, v
	case tomlInt.MatchString(v):
		return TypeInt, v
	case tomlFloat.MatchString(v):
		return TypeFloat, v
	case strings.HasPrefix(v, `"`):
		if u, err := strconv.Unquote(v); err == nil {
			return TypeString, u
		}
	case strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'") && len(v) >= 2:
		return TypeString, v[1 : len(v)-1]
	case strings.HasPrefix(v, "["):
		return TypeList, v
	}
	return TypeString, v
}

// literal is the TOML text for a value of the entry's type, or an error when the text is not one.
func (e melonEntry) literal(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("%s: a value is one line", e.Key)
	}
	value = strings.TrimSpace(value)
	switch e.Type {
	case TypeBool:
		if v := strings.ToLower(value); v == "true" || v == "false" {
			return v, nil
		}
		return "", fmt.Errorf("%s expects true or false", e.Key)
	case TypeInt:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return "", fmt.Errorf("%s expects a whole number", e.Key)
		}
		return value, nil
	case TypeFloat:
		_, value, err := finite(e.Key, value)
		if err != nil {
			return "", err
		}
		if tomlInt.MatchString(value) {
			value += ".0"
		}
		return value, nil
	case TypeList:
		if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
			return "", fmt.Errorf("%s expects a list in [brackets]", e.Key)
		}
		return value, nil
	}
	return tomlString(value), nil
}

// tomlString is value as a TOML basic string; strconv.Quote's \a, \v and \x escapes are not TOML.
func tomlString(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\u%04X`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func (d melonDoc) set(section, key, value string) (string, error) {
	for _, e := range d.entries {
		if e.section != section || e.Key != key {
			continue
		}
		lit, err := e.literal(value)
		if err != nil {
			return "", err
		}
		lines := slices.Delete(append([]string(nil), d.lines...), e.line+1, e.end+1)
		lines[e.line] = e.prefix + lit + e.suffix
		eol := "\n"
		if d.crlf {
			eol = "\r\n"
		}
		out := strings.Join(lines, eol)
		if d.bom {
			out = "\ufeff" + out
		}
		return out, nil
	}
	return "", fmt.Errorf("no setting %s in [%s]", key, section)
}

// schema lists the categories keep accepts, in file order.
func (d melonDoc) schema(file ConfigFile, keep func(section string) bool) Schema {
	s := Schema{File: file, Sections: []Section{}}
	for _, e := range d.entries {
		if !keep(e.section) {
			continue
		}
		if n := len(s.Sections); n == 0 || s.Sections[n-1].Name != e.section {
			s.Sections = append(s.Sections, Section{Name: e.section})
		}
		last := &s.Sections[len(s.Sections)-1]
		last.Entries = append(last.Entries, e.Entry)
	}
	return s
}

// ownsMelonCategory reports whether a category is the settings of a mod known by names: the category's name and the
// mod's name or package name contain one another once case and punctuation are dropped.
func ownsMelonCategory(names []string, section string) bool {
	c := squash(section)
	if len(c) < minConfigName {
		return false
	}
	for _, n := range names {
		_, pkg, _ := strings.Cut(n, ":")
		_, name, _ := strings.Cut(pkg, "-")
		for _, k := range []string{squash(pkg), squash(name)} {
			if len(k) >= minConfigName && (strings.Contains(k, c) || strings.Contains(c, k)) {
				return true
			}
		}
	}
	return false
}

// melonFile lists MelonPreferences.cfg for the mod when one of its categories is in the file, or for the whole profile
// when modID is empty.
func (s *Service) melonFile(game, profileID, modID string, names []string) []ConfigFile {
	path, err := s.melonPath(game, profileID)
	if err != nil || !fsx.IsFile(path) {
		return nil
	}
	if modID != "" {
		raw, err := fsx.ReadFile(path)
		if err != nil {
			return nil
		}
		found := false
		for _, e := range parseMelon(string(raw)).entries {
			if ownsMelonCategory(names, e.section) {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return []ConfigFile{{Name: melonName, Format: FormatMelon, Label: melonName}}
}

func (s *Service) melonSchema(game, profileID, modID string, f ConfigFile) (Schema, error) {
	path, err := s.melonPath(game, profileID)
	if err != nil {
		return Schema{}, err
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return Schema{}, err
	}
	keep := func(string) bool { return true }
	if modID != "" {
		keep = func(section string) bool { return ownsMelonCategory([]string{modID}, section) }
	}
	return parseMelon(string(raw)).schema(f, keep), nil
}

func (s *Service) writeMelon(game, profileID string, edits []edit) error {
	path, err := s.melonPath(game, profileID)
	if err != nil {
		return err
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(raw)
	for _, e := range edits {
		if text, err = parseMelon(text).set(e.section, e.key, e.value); err != nil {
			return err
		}
	}
	return datadir.WriteFile(path, []byte(text), 0o600)
}
