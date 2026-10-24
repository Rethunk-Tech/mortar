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
