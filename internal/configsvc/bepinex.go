package configsvc

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// The layout parsed here is what BepInEx's ConfigFile.Save and ConfigEntryBase.WriteDescription write
// (BepInEx.Core/Configuration/ConfigFile.cs and ConfigEntryBase.cs, BepInEx 5.4.x/6):
//
//	## Settings file was created by plugin <Name> v<Version>
//	## Plugin GUID: <GUID>
//
//	[Section]
//
//	## <description, one "## " line per line of text>
//	# Setting type: <.NET type name>
//	# Default value: <value>
//	# Acceptable values: a, b, c               (AcceptableValueList, or the names of an enum)
//	# Acceptable value range: From x to y      (AcceptableValueRange)
//	# Multiple values can be set at the same time by separating them with , (e.g. Debug, Warning)   (a [Flags] enum)
//	Key = value
//
// The reader skips lines that start with '#', and splits "Key = value" at the first '='.

var (
	settingType = regexp.MustCompile(`^# Setting type: (.+)$`)
	defaultVal  = regexp.MustCompile(`^# Default value: ?(.*)$`)
	acceptList  = regexp.MustCompile(`^# Acceptable values: (.*)$`)
	acceptRange = regexp.MustCompile(`^# Acceptable value range: From (.+) to (.+)$`)
	flagsLine   = "# Multiple values can be set at the same time by separating them with ,"
	sectionLine = regexp.MustCompile(`^\[(.+)\]$`)
	pluginName  = regexp.MustCompile(`^## Settings file was created by plugin (.+) v\S+$`)
	pluginGUID  = regexp.MustCompile(`^## Plugin GUID: (.+)$`)
)

// cfgDoc is a parsed .cfg: the original lines, so a write changes only the edited line.
type cfgDoc struct {
	lines []string
	// bom and crlf are the file's own encoding, which a write keeps.
	bom, crlf bool
	// plugin and guid come from the header.
	plugin, guid string
	entries      []cfgEntry
}

type cfgEntry struct {
	section string
	line    int
	Entry
}

func parseCfg(text string) cfgDoc {
	text, bom := strings.CutPrefix(text, "\ufeff")
	d := cfgDoc{bom: bom, crlf: strings.Contains(text, "\r\n")}
	d.lines = strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var section string
	var pending Entry
	var desc []string
	reset := func() { pending, desc = Entry{}, nil }
	for i, raw := range d.lines {
		line := strings.TrimSpace(raw)
		switch {
		case line == "":
			reset()
		case pluginName.MatchString(line):
			d.plugin = pluginName.FindStringSubmatch(line)[1]
		case pluginGUID.MatchString(line):
			d.guid = pluginGUID.FindStringSubmatch(line)[1]
		case strings.HasPrefix(line, "## "):
			desc = append(desc, strings.TrimPrefix(line, "## "))
		case strings.HasPrefix(line, "##"):
			desc = append(desc, strings.TrimPrefix(line, "##"))
		case strings.HasPrefix(line, "#"):
			readMeta(&pending, line)
		case sectionLine.MatchString(line):
			section = sectionLine.FindStringSubmatch(line)[1]
			reset()
		default:
			key, val, ok := strings.Cut(line, "=")
			if !ok {
				reset()
				continue
			}
			e := pending
			e.Key, e.Value = strings.TrimSpace(key), strings.TrimSpace(val)
			e.Description = strings.Join(desc, "\n")
			finishEntry(&e)
			d.entries = append(d.entries, cfgEntry{section: section, line: i, Entry: e})
			reset()
		}
	}
	return d
}

func readMeta(e *Entry, line string) {
	switch {
	case settingType.MatchString(line):
		e.Type = settingType.FindStringSubmatch(line)[1]
	case defaultVal.MatchString(line):
		e.Default, e.HasDefault = defaultVal.FindStringSubmatch(line)[1], true
	case acceptList.MatchString(line):
		for v := range strings.SplitSeq(acceptList.FindStringSubmatch(line)[1], ", ") {
			e.Values = append(e.Values, strings.TrimSpace(v))
		}
	case acceptRange.MatchString(line):
		m := acceptRange.FindStringSubmatch(line)
		if lo, err := strconv.ParseFloat(strings.TrimSpace(m[1]), 64); err == nil {
			e.Min = &lo
		}
		if hi, err := strconv.ParseFloat(strings.TrimSpace(m[2]), 64); err == nil {
			e.Max = &hi
		}
	case strings.HasPrefix(line, flagsLine):
		e.Flags = true
	}
}

// finishEntry turns the .NET type name into one of the schema types.
func finishEntry(e *Entry) {
	switch name := e.Type; {
	case len(e.Values) > 0:
		e.Type = TypeEnum
	case name == "Boolean":
		e.Type = TypeBool
	case slices.Contains([]string{"Int32", "Int64", "Int16", "UInt32", "UInt64", "UInt16", "Byte", "SByte"}, name):
		e.Type = TypeInt
	case slices.Contains([]string{"Single", "Double", "Decimal"}, name):
		e.Type = TypeFloat
	case name == "Color":
		e.Type = TypeColor
	case strings.HasSuffix(name, "[]"):
		e.Type = TypeList
	default:
		e.Type = TypeString
	}
}

// validate checks value against the entry's type, range and acceptable values and returns the text to write.
func validate(e Entry, value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("%s: a value is one line", e.Key)
	}
	value = strings.TrimSpace(value)
	switch e.Type {
	case TypeBool:
		switch strings.ToLower(value) {
		case "true", "false":
			return strings.ToLower(value), nil
		}
		return "", fmt.Errorf("%s expects true or false", e.Key)
	case TypeInt:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return "", fmt.Errorf("%s expects a whole number", e.Key)
		}
		return value, inRange(e, float64(n))
	case TypeFloat:
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "", fmt.Errorf("%s expects a number", e.Key)
		}
		return value, inRange(e, n)
	case TypeEnum:
		parts := []string{value}
		if e.Flags {
			parts = strings.Split(value, ",")
		}
		for _, p := range parts {
			if !slices.Contains(e.Values, strings.TrimSpace(p)) {
				return "", fmt.Errorf("%s must be one of %s", e.Key, strings.Join(e.Values, ", "))
			}
		}
	}
	return value, nil
}

func inRange(e Entry, n float64) error {
	if (e.Min != nil && n < *e.Min) || (e.Max != nil && n > *e.Max) {
		return fmt.Errorf("%s must be between %v and %v", e.Key, bound(e.Min), bound(e.Max))
	}
	return nil
}

func bound(p *float64) any {
	if p == nil {
		return "any"
	}
	return *p
}

func (d cfgDoc) find(section, key string) (cfgEntry, bool) {
	for _, e := range d.entries {
		if e.section == section && e.Key == key {
			return e, true
		}
	}
	return cfgEntry{}, false
}

// set returns the file text with one value replaced, every other byte as it was.
func (d cfgDoc) set(section, key, value string) (string, error) {
	e, ok := d.find(section, key)
	if !ok {
		return "", fmt.Errorf("no setting %s in [%s]", key, section)
	}
	val, err := validate(e.Entry, value)
	if err != nil {
		return "", err
	}
	lines := append([]string(nil), d.lines...)
	old := lines[e.line]
	eq := strings.Index(old, "=")
	lines[e.line] = strings.TrimRight(old[:eq+1], " ") + " " + val
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

func (d cfgDoc) schema(file ConfigFile) Schema {
	s := Schema{File: file}
	for _, e := range d.entries {
		if n := len(s.Sections); n == 0 || s.Sections[n-1].Name != e.section {
			s.Sections = append(s.Sections, Section{Name: e.section})
		}
		last := &s.Sections[len(s.Sections)-1]
		last.Entries = append(last.Entries, e.Entry)
	}
	return s
}
