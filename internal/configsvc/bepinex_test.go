package configsvc

import (
	"os"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
)

func fixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/example.cfg")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseReadsTypesRangesAndDescriptions(t *testing.T) {
	d := parseCfg(fixture(t))
	if d.plugin != "LC Better Stuff" || d.guid != "com.example.betterstuff" {
		t.Fatalf("header %q %q", d.plugin, d.guid)
	}
	cases := []struct {
		section, key, typ, def, value string
	}{
		{"General", "ShowHud", TypeBool, "true", "true"},
		{"General", "LeaveDelay", TypeInt, "90", "120"},
		{"General", "ScrapMultiplier", TypeFloat, "1", "1.25"},
		{"Advanced", "Mode", TypeEnum, "Normal", "Hard"},
		{"Advanced", "Label", TypeString, "Crew", "Night Shift"},
		{"Advanced", "Tint", TypeColor, "FFFFFFFF", "FF8800FF"},
	}
	for _, c := range cases {
		e, ok := d.find(c.section, c.key)
		if !ok || e.Type != c.typ || e.Default != c.def || e.Value != c.value || !e.HasDefault {
			t.Fatalf("%s.%s = %+v", c.section, c.key, e)
		}
	}
	delay, _ := d.find("General", "LeaveDelay")
	if delay.Min == nil || *delay.Min != 10 || *delay.Max != 300 {
		t.Fatalf("range %+v", delay)
	}
	if delay.Description != "How many seconds before the ship leaves.\nLonger waits are safer." {
		t.Fatalf("description %q", delay.Description)
	}
	mode, _ := d.find("Advanced", "Mode")
	if strings.Join(mode.Values, ",") != "Easy,Normal,Hard" {
		t.Fatalf("values %v", mode.Values)
	}
}

func TestSetChangesOnlyTheEditedLine(t *testing.T) {
	text := fixture(t)
	d := parseCfg(text)
	got, err := d.set("General", "LeaveDelay", "150")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(text, "LeaveDelay = 120", "LeaveDelay = 150", 1)
	if got != want {
		t.Fatalf("a write must keep comments and order:\n%s", got)
	}
	for _, bad := range [][3]string{{"General", "LeaveDelay", "5"}, {"General", "ShowHud", "maybe"}, {"Advanced", "Mode", "Nightmare"}, {"General", "LeaveDelay", "1.5"}} {
		if _, err := d.set(bad[0], bad[1], bad[2]); err == nil {
			t.Fatalf("%v must be refused", bad)
		}
	}
}

func TestJSONSchemaKeepsOrderAndTakesDefaultsAndGMCM(t *testing.T) {
	cur := `{"Enabled": true, "Speed": 2.5, "Mode": "b", "Nested": {"Count": 3, "Tags": ["x","y"]}, "Extra": "kept"}`
	ship := `{"Enabled": false, "Speed": 1.0, "Mode": "a", "Nested": {"Count": 1}}`
	lo, hi := 0.0, 5.0
	field := "Speed"
	mode := "Mode"
	capture := &gmcm.Capture{Pages: []gmcm.Page{{Options: []gmcm.Option{
		{FieldID: &field, Name: "Move speed", Tooltip: "How fast.", Min: &lo, Max: &hi},
		{FieldID: &mode, Choices: []gmcm.Choice{{Value: "a"}, {Value: "b"}}},
	}}}}
	s, err := jsonSchema(ConfigFile{Name: "config.json"}, cur, ship, capture, nil)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, sec := range s.Sections {
		for _, e := range sec.Entries {
			keys = append(keys, sec.Name+"/"+e.Key)
		}
	}
	if strings.Join(keys, " ") != "/Enabled /Speed /Mode /Extra Nested/Count Nested/Tags" {
		t.Fatalf("order %v", keys)
	}
	e := s.Sections[0].Entries
	if e[0].Type != TypeBool || e[0].Default != "false" || !e[0].HasDefault {
		t.Fatalf("bool %+v", e[0])
	}
	if e[1].Type != TypeFloat || e[1].Label != "Move speed" || *e[1].Max != 5 || e[1].Description != "How fast." {
		t.Fatalf("float %+v", e[1])
	}
	if e[2].Type != TypeEnum || len(e[2].Values) != 2 {
		t.Fatalf("enum %+v", e[2])
	}
	if s.Sections[1].Entries[1].Type != TypeList || s.Sections[1].Entries[1].Value != `["x","y"]` || e[3].HasDefault {
		t.Fatalf("list/unknown %+v %+v", s.Sections[1], e[3])
	}
}
