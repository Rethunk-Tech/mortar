package configsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	// eols is each line's own ending, which a write keeps; bom likewise.
	eols []string
	bom  bool
	// plugin and guid come from the header.
	plugin, guid string
	entries      []cfgEntry
}

type cfgEntry struct {
	section string
	line    int
	// netType is the .NET type BepInEx parses the value into.
	netType string
	Entry
}

// netLimits bound the .NET number types narrower than what validate parses; BepInEx replaces a value that overflows
// its type with the default when the game starts.
var netLimits = map[string][2]float64{
	"Byte": {0, math.MaxUint8}, "SByte": {math.MinInt8, math.MaxInt8},
	"Int16": {math.MinInt16, math.MaxInt16}, "UInt16": {0, math.MaxUint16},
	"Int32": {math.MinInt32, math.MaxInt32}, "UInt32": {0, math.MaxUint32}, "UInt64": {0, math.MaxUint64},
	"Single": {-math.MaxFloat32, math.MaxFloat32},
}

// decimalMax is .NET's decimal.MaxValue, which a float64 cannot hold exactly; decimalBound is it for the refusal's detail.
var (
	decimalMax, _ = new(big.Rat).SetString("79228162514264337593543950335")
	decimalBound  = [2]float64{-79228162514264337593543950335, 79228162514264337593543950335}
)

// hexColor is what Unity's ColorUtility.TryParseHtmlString reads after BepInEx adds the "#": RGB, RGBA, RRGGBB or RRGGBBAA.
var hexColor = regexp.MustCompile(`^#?([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

var (
	unsignedTypes = []string{"Byte", "UInt16", "UInt32", "UInt64"}
	// plainDecimal is what .NET's decimal.Parse reads: digits with an optional point, no exponent.
	plainDecimal = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)$`)
)

func parseCfg(text string) cfgDoc {
	text, bom := strings.CutPrefix(text, "\ufeff")
	d := cfgDoc{bom: bom}
	d.lines, d.eols = splitLines(text)
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
			netType := e.Type
			finishEntry(&e)
			d.entries = append(d.entries, cfgEntry{section: section, line: i, netType: netType, Entry: e})
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
func validate(e cfgEntry, value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", refuse("line", "%s: a value is one line", e.Key)
	}
	value = strings.TrimSpace(value)
	switch e.Type {
	case TypeBool:
		switch strings.ToLower(value) {
		case "true", "false":
			return strings.ToLower(value), nil
		}
		return "", refuse("bool", "%s expects true or false", e.Key)
	case TypeInt:
		var n float64
		if slices.Contains(unsignedTypes, e.netType) {
			u, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return "", refuse("whole_number", "%s expects a whole number", e.Key)
			}
			n = float64(u)
		} else {
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return "", refuse("whole_number", "%s expects a whole number", e.Key)
			}
			n = float64(i)
		}
		return value, inRange(e.Entry, n)
	case TypeFloat:
		n, text, err := finite(e.Key, value)
		if err != nil {
			return "", err
		}
		if e.netType == "Decimal" {
			// decimal.Parse takes no exponent: keep plain digits as typed, else spell the number out.
			text = strconv.FormatFloat(n, 'f', -1, 64)
			if plainDecimal.MatchString(value) {
				text = value
			}
		}
		return text, inRange(e.Entry, n)
	case TypeColor:
		if !hexColor.MatchString(value) {
			return "", refuse("color", "%s expects a hex color such as FF8800 or FF8800CC", e.Key)
		}
	case TypeEnum:
		parts := []string{value}
		if e.Flags {
			parts = strings.Split(value, ",")
		}
		// Enum.Parse ignores case; the file gets the member's own spelling.
		for i, p := range parts {
			trimmed := strings.TrimSpace(p)
			j := slices.IndexFunc(e.Values, func(v string) bool { return strings.EqualFold(v, trimmed) })
			if j < 0 {
				return "", refuseEnum(e)
			}
			parts[i] = strings.Replace(p, trimmed, e.Values[j], 1)
		}
		value = strings.Join(parts, ",")
	}
	return value, nil
}

// finite parses a number and writes it the way both .NET and TOML read it: Go also takes "1_000", hex and "Inf", which
// neither does.
func finite(key, value string) (float64, string, error) {
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
		return 0, "", refuse("number", "%s expects a number", key)
	}
	return n, strconv.FormatFloat(n, 'g', -1, 64), nil
}

func refuseEnum(e cfgEntry) error {
	return usererr.Wrap(usererr.Invalid, &valueError{Problem: "enum", Values: e.Values, msg: fmt.Sprintf("%s must be one of %s", e.Key, strings.Join(e.Values, ", "))})
}

// valueError is a value the editor will not write. Its English sentence is the error text; Detail is what the GUI words
// its own sentence from (usererr.Detailer).
type valueError struct {
	// Problem is one of line, bool, whole_number, number, enum, range, empty, list, raw, toml and color.
	Problem string   `json:"problem"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Values  []string `json:"values,omitempty"`
	msg     string
}

func (e *valueError) Error() string { return e.msg }

func (e *valueError) Detail() any { return e }

func refuse(problem, format string, args ...any) error {
	return usererr.Wrap(usererr.Invalid, &valueError{Problem: problem, msg: fmt.Sprintf(format, args...)})
}

func refuseRange(key string, lo, hi *float64) error {
	f := func(p *float64) string { return strconv.FormatFloat(*p, 'f', -1, 64) }
	var msg string
	switch {
	case lo == nil:
		msg = fmt.Sprintf("%s must be at most %s", key, f(hi))
	case hi == nil:
		msg = fmt.Sprintf("%s must be at least %s", key, f(lo))
	default:
		msg = fmt.Sprintf("%s must be between %s and %s", key, f(lo), f(hi))
	}
	return usererr.Wrap(usererr.Invalid, &valueError{Problem: "range", Min: lo, Max: hi, msg: msg})
}

func inRange(e Entry, n float64) error {
	if (e.Min != nil && n < *e.Min) || (e.Max != nil && n > *e.Max) {
		return refuseRange(e.Key, e.Min, e.Max)
	}
	return nil
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
	val, err := validate(e, value)
	if err != nil {
		return "", err
	}
	if e.netType == "Decimal" {
		if r, ok := new(big.Rat).SetString(val); ok && r.Abs(r).Cmp(decimalMax) > 0 {
			return "", refuseRange(key, &decimalBound[0], &decimalBound[1])
		}
	}
	if lim, ok := netLimits[e.netType]; ok {
		if n, _ := strconv.ParseFloat(val, 64); n < lim[0] || n > lim[1] {
			return "", refuseRange(key, &lim[0], &lim[1])
		}
	}
	lines := append([]string(nil), d.lines...)
	old := lines[e.line]
	eq := strings.Index(old, "=")
	lines[e.line] = strings.TrimRight(old[:eq+1], " ") + " " + val
	return joinLines(lines, d.eols, d.bom), nil
}

func (d cfgDoc) schema(file ConfigFile) Schema {
	s := Schema{File: file}
	for _, e := range d.entries {
		s.Sections = addToSection(s.Sections, e.section, e.Entry)
	}
	return s
}

// addToSection appends e to the last section when it is named section, else to a new one: entries read in file order
// group by the section run they sit in.
func addToSection(sections []Section, section string, e Entry) []Section {
	if n := len(sections); n == 0 || sections[n-1].Name != section {
		sections = append(sections, Section{Name: section})
	}
	last := &sections[len(sections)-1]
	last.Entries = append(last.Entries, e)
	return sections
}

// withSync marks the entries ConfigSync shares from the host, read from the sidecar beside the .cfg at path. A
// missing or malformed sidecar marks nothing.
func withSync(s Schema, path string) Schema {
	raw, err := fsx.ReadFile(strings.TrimSuffix(path, filepath.Ext(path)) + syncSidecar)
	if err != nil {
		return s
	}
	var side struct {
		Settings []struct {
			Section string `json:"section"`
			Key     string `json:"key"`
			Level   string `json:"level"`
		} `json:"settings"`
	}
	if json.Unmarshal(raw, &side) != nil {
		return s
	}
	for _, sh := range side.Settings {
		for i := range s.Sections {
			if s.Sections[i].Name != sh.Section {
				continue
			}
			for j := range s.Sections[i].Entries {
				if e := &s.Sections[i].Entries[j]; e.Key == sh.Key {
					e.Sync = sh.Level
				}
			}
		}
	}
	return s
}

// HostValue is the value a ConfigSync host sent for a shared setting. InUse is false while the player keeps their own
// value for a host recommendation.
type HostValue struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	InUse   bool   `json:"inUse"`
}

// askHostFor bounds the bridge query the config editor repeats while it is open.
const askHostFor = 2 * time.Second

// HostValues asks the running game which values the host sent for the settings the mod behind file shares through
// ConfigSync. There are none while the game is not running, before the bridge is up, or with a bridge older than
// the query.
func (s *Service) HostValues(game, profileID, file string) ([]HostValue, error) {
	none := []HostValue{}
	if s.Running == nil || !s.Running(game, profileID) || !strings.EqualFold(filepath.Ext(file), cfgExt) {
		return none, nil
	}
	dir, err := s.Profiles.ProfileDir(game, profileID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), askHostFor)
	defer cancel()
	raw, err := bridge.Query(ctx, filepath.Join(dir, filepath.FromSlash(bridge.BepInEx.StateFile)), "configsync")
	var rejected *bridge.RejectedError
	if errors.Is(err, bridge.ErrNotRunning) || errors.Is(err, bridge.ErrNotReady) || errors.As(err, &rejected) {
		return none, nil
	}
	if err != nil {
		return nil, err
	}
	return hostValuesOf(raw, strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))
}

// hostValuesOf picks the bridge's rows for one plugin: a BepInEx config file is named after its plugin's GUID.
func hostValuesOf(raw json.RawMessage, guid string) ([]HostValue, error) {
	var rows []struct {
		Mod string `json:"mod"`
		HostValue
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := []HostValue{}
	for _, r := range rows {
		if r.Mod == guid {
			out = append(out, r.HostValue)
		}
	}
	return out, nil
}

// SaveHostValues keeps the host's current values of the settings the file's mod shares, to write into the player's own
// .cfg once the game is closed: Mortar writes no config while the game runs, which may rewrite it. It returns how many.
func (s *Service) SaveHostValues(game, profileID, modID, file string) (int, error) {
	values, err := s.HostValues(game, profileID, file)
	if err != nil || len(values) == 0 {
		return 0, err
	}
	edits := make([]edit, 0, len(values))
	for _, v := range values {
		edits = append(edits, edit{v.Section, v.Key, v.Value})
	}
	s.hostMu.Lock()
	defer s.hostMu.Unlock()
	s.loadKept()
	s.hostKept[hostKey{game, profileID, modID, file}] = edits
	s.saveKept()
	return len(edits), nil
}

type keptRecord struct {
	Game    string     `json:"game"`
	Profile string     `json:"profile"`
	Mod     string     `json:"mod"`
	File    string     `json:"file"`
	Edits   []keptEdit `json:"edits"`
}

type keptEdit struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

const keptFile = "host-kept.json"

// loadKept reads the kept values the first time they are needed, so they survive a restart; callers hold hostMu.
func (s *Service) loadKept() {
	if s.hostKept != nil {
		return
	}
	s.hostKept = map[hostKey][]edit{}
	dir, err := datadir.Dir()
	if err != nil {
		return
	}
	var recs []keptRecord
	if _, err := datadir.ReadJSON(filepath.Join(dir, keptFile), &recs); err != nil {
		return
	}
	for _, r := range recs {
		edits := make([]edit, 0, len(r.Edits))
		for _, e := range r.Edits {
			edits = append(edits, edit{e.Section, e.Key, e.Value})
		}
		s.hostKept[hostKey{r.Game, r.Profile, r.Mod, r.File}] = edits
	}
}

// saveKept persists the kept values; callers hold hostMu.
func (s *Service) saveKept() {
	dir, err := datadir.Dir()
	if err != nil {
		return
	}
	recs := make([]keptRecord, 0, len(s.hostKept))
	for k, edits := range s.hostKept {
		r := keptRecord{Game: k.game, Profile: k.profile, Mod: k.mod, File: k.file}
		for _, e := range edits {
			r.Edits = append(r.Edits, keptEdit{e.section, e.key, e.value})
		}
		recs = append(recs, r)
	}
	slices.SortFunc(recs, func(a, b keptRecord) int {
		return strings.Compare(a.Game+"\x00"+a.Profile+"\x00"+a.Mod+"\x00"+a.File, b.Game+"\x00"+b.Profile+"\x00"+b.Mod+"\x00"+b.File)
	})
	if b, err := json.Marshal(recs); err == nil {
		_ = datadir.WriteFile(filepath.Join(dir, keptFile), b, 0o600)
	}
}

// FlushKept writes the kept host values of every profile whose game is no longer running; the app calls it when a
// profile unlocks, so the values land without the config page being opened.
func (s *Service) FlushKept() {
	s.hostMu.Lock()
	s.loadKept()
	keys := make([]hostKey, 0, len(s.hostKept))
	for k := range s.hostKept {
		keys = append(keys, k)
	}
	s.hostMu.Unlock()
	for _, k := range keys {
		if s.Running != nil && s.Running(k.game, k.profile) {
			continue
		}
		files, err := s.Files(k.game, k.profile, k.mod)
		if err != nil {
			log.Printf("config: kept values of %s: %v", k.file, err)
			continue
		}
		for _, f := range files {
			if f.Name != k.file {
				continue
			}
			if _, err := s.writeKept(k.game, k.profile, k.mod, f); err != nil {
				log.Printf("config: kept values of %s: %v", k.file, err)
			}
		}
	}
}

// writeKept writes the host's values kept for f, once the game is no longer running.
func (s *Service) writeKept(game, profileID, modID string, f ConfigFile) (bool, error) {
	if s.Running != nil && s.Running(game, profileID) {
		return false, nil
	}
	k := hostKey{game, profileID, modID, f.Name}
	s.hostMu.Lock()
	s.loadKept()
	edits, ok := s.hostKept[k]
	if ok {
		delete(s.hostKept, k)
		s.saveKept()
	}
	s.hostMu.Unlock()
	if !ok {
		return false, nil
	}
	return true, s.write(game, profileID, modID, f, edits)
}
