package configsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	if _, err := d.set("General", "Gone", "1"); usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("a missing setting: %v", err)
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

func TestSetKeepsBOMAndCRLF(t *testing.T) {
	text := "\ufeff" + strings.ReplaceAll(fixture(t), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", "\r\n")
	got, err := parseCfg(text).set("General", "LeaveDelay", "150")
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(text, "LeaveDelay = 120", "LeaveDelay = 150", 1); got != want {
		t.Fatalf("BOM and CRLF must survive a write:\n%q", got)
	}
}

// BepInEx parses a value into its .NET type and drops one that does not fit for the default when the game starts.
func TestSetRefusesWhatTheDotNetTypeCannotHold(t *testing.T) {
	d := parseCfg("[A]\n# Setting type: Int32\nN = 1\n# Setting type: Byte\nB = 1\n# Setting type: Single\nF = 1\n")
	for _, bad := range [][2]string{{"N", "3000000000"}, {"B", "256"}, {"B", "-1"}, {"F", "Inf"}, {"F", "1e39"}} {
		if out, err := d.set("A", bad[0], bad[1]); err == nil {
			t.Errorf("%v accepted: %q", bad, out)
		}
	}
	if out, err := d.set("A", "B", "255"); err != nil || !strings.Contains(out, "B = 255\n") {
		t.Fatalf("%q %v", out, err)
	}
	// Go reads "1_000" and hex floats, .NET does not: the value is written in a form both read.
	if out, err := d.set("A", "F", "1_000"); err != nil || !strings.Contains(out, "F = 1000\n") {
		t.Fatalf("%q %v", out, err)
	}
}

func TestSyncSidecarMarksSharedEntries(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "com.example.betterstuff.cfg")
	schema := func() Schema { return parseCfg(fixture(t)).schema(ConfigFile{Name: "com.example.betterstuff.cfg"}) }
	side := strings.TrimSuffix(cfg, ".cfg") + syncSidecar
	sync := func(s Schema) (got []string) {
		for _, sec := range s.Sections {
			for _, e := range sec.Entries {
				if e.Sync != "" {
					got = append(got, sec.Name+"."+e.Key+"="+e.Sync)
				}
			}
		}
		return got
	}

	body := `{"version":1,"settings":[{"section":"General","key":"ShowHud","level":"hostEnforced"},{"section":"Advanced","key":"Mode","level":"hostRecommended"},{"section":"General","key":"Gone","level":"hostEnforced"}]}`
	if err := os.WriteFile(side, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(sync(withSync(schema(), cfg)), " "); got != "General.ShowHud=hostEnforced Advanced.Mode=hostRecommended" {
		t.Fatalf("levels = %q", got)
	}

	if err := os.WriteFile(side, []byte(`{"settings":[`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := sync(withSync(schema(), cfg)); len(got) != 0 {
		t.Fatalf("malformed sidecar gave %v", got)
	}
}

func TestHostValuesKeepOnlyTheFilesPlugin(t *testing.T) {
	raw := []byte(`[{"mod":"tech.rethunk.mortar.Dropship","section":"General","key":"Distance","value":"60","inUse":true},` +
		`{"mod":"other.mod","section":"General","key":"Distance","value":"5","inUse":true}]`)
	got, err := hostValuesOf(raw, "tech.rethunk.mortar.Dropship")
	if err != nil || len(got) != 1 || got[0] != (HostValue{Section: "General", Key: "Distance", Value: "60", InUse: true}) {
		t.Fatalf("hostValuesOf = %+v, %v", got, err)
	}
}

func TestSetKeepsEachLinesOwnEnding(t *testing.T) {
	const text = "[A]\r\n# Setting type: Int32\nK = 1\nL = 2\r\n# Setting type: Int32\nM = 3"
	got, err := parseCfg(text).set("A", "L", "5")
	if want := strings.Replace(text, "L = 2", "L = 5", 1); err != nil || got != want {
		t.Fatalf("%q %v", got, err)
	}
}

func TestSetWritesDecimalsPlainAndAcceptsWhatDotNetReads(t *testing.T) {
	d := parseCfg("[A]\n# Setting type: Decimal\nD = 1\n# Setting type: UInt64\nU = 1\n# Setting type: Int32\n# Acceptable values: Easy, Hard\nMode = Easy\n# Setting type: Mode\n# Acceptable values: Easy, Hard, Wild\n# Multiple values can be set at the same time by separating them with ,\nM = Easy\n")
	for in, want := range map[string]string{"0.00001": "D = 0.00001\n", "1e-5": "D = 0.00001\n", "123456789012345678901234": "D = 123456789012345678901234\n"} {
		if out, err := d.set("A", "D", in); err != nil || !strings.Contains(out, want) {
			t.Errorf("%s: %q %v", in, out, err)
		}
	}
	if out, err := d.set("A", "U", "18446744073709551615"); err != nil || !strings.Contains(out, "U = 18446744073709551615\n") {
		t.Errorf("uint64: %q %v", out, err)
	}
	if _, err := d.set("A", "U", "-1"); err == nil {
		t.Error("negative UInt64 accepted")
	}
	if out, err := d.set("A", "Mode", "hARD"); err != nil || !strings.Contains(out, "Mode = Hard\n") {
		t.Errorf("enum case: %q %v", out, err)
	}
	if out, err := d.set("A", "M", "easy,WILD"); err != nil || !strings.Contains(out, "M = Easy,Wild\n") {
		t.Errorf("flags case: %q %v", out, err)
	}
}

func TestRefusedValueIsInvalidAndNamesItsProblem(t *testing.T) {
	d := parseCfg(fixture(t))
	for _, c := range []struct{ section, key, value, want string }{
		{"General", "LeaveDelay", "5", "range"},
		{"General", "LeaveDelay", "1.5", "whole_number"},
		{"General", "ShowHud", "maybe", "bool"},
		{"Advanced", "Mode", "Nightmare", "enum"},
		{"General", "ScrapMultiplier", "x", "number"},
	} {
		_, err := d.set(c.section, c.key, c.value)
		detail, _ := usererr.DetailOf(err).(*valueError)
		if usererr.KindOf(err) != usererr.Invalid || detail == nil || detail.Problem != c.want || strings.Contains(err.Error(), "config_value") {
			t.Errorf("%s=%s: %v %+v", c.key, c.value, err, detail)
		}
	}
	_, err := d.set("General", "LeaveDelay", "5")
	if detail, _ := usererr.DetailOf(err).(*valueError); detail.Min == nil || *detail.Min != 10 || *detail.Max != 300 {
		t.Errorf("range bounds: %+v", detail)
	}
}

func TestDecimalAndColorAreCheckedTheWayBepInExParsesThem(t *testing.T) {
	d := parseCfg("[A]\n# Setting type: Decimal\nD = 1\n# Setting type: Color\nC = FF0000FF\n")
	const top = "79228162514264337593543950335"
	if out, err := d.set("A", "D", top); err != nil || !strings.Contains(out, "D = "+top+"\n") {
		t.Errorf("decimal max: %q %v", out, err)
	}
	for _, v := range []string{"79228162514264337593543950336", "-79228162514264337593543950336", "1e30"} {
		if _, err := d.set("A", "D", v); err == nil {
			t.Errorf("decimal %s accepted", v)
		}
	}
	for _, v := range []string{"FF8800", "#ff880080", "f80"} {
		if _, err := d.set("A", "C", v); err != nil {
			t.Errorf("color %s: %v", v, err)
		}
	}
	for _, v := range []string{"zzz", "FF880", "FF88000", ""} {
		if _, err := d.set("A", "C", v); err == nil {
			t.Errorf("color %q accepted", v)
		}
	}
}
