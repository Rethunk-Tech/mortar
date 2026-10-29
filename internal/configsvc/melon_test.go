package configsvc

import (
	"strings"
	"testing"
)

// The layout is what MelonPreferences_Entry.Save writes through Tomlet: the description as a preceding comment, the
// optional inline comment after the value.
const melonSample = `[MelonLoader]
# Mod load order
load_order = "default" # inline

[BetterHands]
# Turns the feature on
Enabled = true
# Multiplier
Speed = 2.5
Count = 4
Names = ["a", "b"]
Label = "Hello # not a comment"
`

func TestMelonPreferencesAreTypedByTheirValuesAndEditedInPlace(t *testing.T) {
	d := parseMelon(melonSample)
	schema := d.schema(ConfigFile{Name: melonName, Format: FormatMelon}, func(s string) bool { return s == "BetterHands" })
	if len(schema.Sections) != 1 {
		t.Fatalf("sections %+v", schema.Sections)
	}
	got := map[string]Entry{}
	for _, e := range schema.Sections[0].Entries {
		got[e.Key] = e
	}
	for key, want := range map[string]string{"Enabled": TypeBool, "Speed": TypeFloat, "Count": TypeInt, "Names": TypeList, "Label": TypeString} {
		if got[key].Type != want {
			t.Errorf("%s type %q, want %q", key, got[key].Type, want)
		}
	}
	if got["Enabled"].Description != "Turns the feature on" || got["Label"].Value != "Hello # not a comment" {
		t.Errorf("entries %+v", got)
	}
	out, err := d.set("BetterHands", "Speed", "3")
	if err != nil || !strings.Contains(out, "Speed = 3.0\n") || strings.Count(out, "\n") != strings.Count(melonSample, "\n") {
		t.Fatalf("set float: %q %v", out, err)
	}
	out, err = d.set("MelonLoader", "load_order", "custom")
	if err != nil || !strings.Contains(out, `load_order = "custom" # inline`+"\n") {
		t.Fatalf("set string keeps the inline comment: %q %v", out, err)
	}
	if _, err := d.set("BetterHands", "Count", "x"); err == nil {
		t.Fatal("a non-number was accepted for an int")
	}
	if !ownsMelonCategory([]string{"thunderstore:Ada-Better_Hands"}, "BetterHands") || ownsMelonCategory([]string{"thunderstore:Ada-Other"}, "BetterHands") {
		t.Fatal("category ownership")
	}
}

// Tomlet reads a float only as lowercase TOML ("1e5", "inf"); "1E5", "Infinity" or "0x1p3" fails the whole file.
func TestMelonFloatIsWrittenAsTOML(t *testing.T) {
	d := parseMelon(melonSample)
	for in, want := range map[string]string{"1E5": "Speed = 100000.0\n", "0x1p3": "Speed = 8.0\n", ".5": "Speed = 0.5\n"} {
		if out, err := d.set("BetterHands", "Speed", in); err != nil || !strings.Contains(out, want) {
			t.Errorf("%s: %q %v", in, out, err)
		}
	}
	for _, in := range []string{"Infinity", "NaN"} {
		if _, err := d.set("BetterHands", "Speed", in); err == nil {
			t.Errorf("%s accepted", in)
		}
	}
}

func TestMelonStringIsWrittenWithTOMLEscapes(t *testing.T) {
	d := parseMelon("[Mod]\nName = \"x\"\n")
	out, err := d.set("Mod", "Name", "a\a\v\x01é\"\\\tz")
	if err != nil || out != "[Mod]\nName = \"a\\u0007\\u000B\\u0001é\\\"\\\\\\tz\"\n" {
		t.Fatalf("%q %v", out, err)
	}
}

func TestMelonQuotedCategoryIsNamedWithoutItsQuotes(t *testing.T) {
	d := parseMelon("[\"My Mod\"]\nOn = true\n['Lit.Mod']\nOn = true\n")
	if d.entries[0].section != "My Mod" || d.entries[1].section != "Lit.Mod" {
		t.Fatalf("sections %q, %q", d.entries[0].section, d.entries[1].section)
	}
	if out, err := d.set("My Mod", "On", "false"); err != nil || out != "[\"My Mod\"]\nOn = false\n['Lit.Mod']\nOn = true\n" {
		t.Fatalf("%q %v", out, err)
	}
}

// Tomlet writes a string holding a newline and a backslash or double quote as a multi-line literal, and an array of
// arrays one element per line; an edit replaces the whole value, and no line inside it reads as a setting.
func TestMelonMultiLineValuesAreEditedWhole(t *testing.T) {
	const text = "[Mod]\nNote = '''\nsay \"a=b\"\nC:\\x'''\nGrid = [\n\t[ \"[\", 1, ],\n\t[ 2, ],\n]\nAfter = 1\n"
	d := parseMelon(text)
	var keys []string
	for _, e := range d.entries {
		keys = append(keys, e.Key)
	}
	if strings.Join(keys, ",") != "Note,Grid,After" || d.entries[0].Value != "say \"a=b\"\nC:\\x" {
		t.Fatalf("entries %+v", d.entries)
	}
	out, err := d.set("Mod", "Note", "short")
	if err != nil || out != "[Mod]\nNote = \"short\"\nGrid = [\n\t[ \"[\", 1, ],\n\t[ 2, ],\n]\nAfter = 1\n" {
		t.Fatalf("set string: %q %v", out, err)
	}
	out, err = d.set("Mod", "Grid", "[]")
	if err != nil || out != "[Mod]\nNote = '''\nsay \"a=b\"\nC:\\x'''\nGrid = []\nAfter = 1\n" {
		t.Fatalf("set array: %q %v", out, err)
	}
}

func TestMelonSetKeepsEachLinesOwnEnding(t *testing.T) {
	const text = "[M]\r\nA = 1\nB = [\r\n 1,\r\n]\nC = 3"
	d := parseMelon(text)
	if out, err := d.set("M", "A", "2"); err != nil || out != "[M]\r\nA = 2\nB = [\r\n 1,\r\n]\nC = 3" {
		t.Fatalf("%q %v", out, err)
	}
	if out, err := d.set("M", "B", "[]"); err != nil || out != "[M]\r\nA = 1\nB = []\nC = 3" {
		t.Fatalf("%q %v", out, err)
	}
}
