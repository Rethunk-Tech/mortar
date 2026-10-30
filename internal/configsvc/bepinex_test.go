package configsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-cfgedit"
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

func TestSyncSidecarMarksSharedEntries(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "com.example.betterstuff.cfg")
	schema := func() Schema {
		return cfgSchema(cfgedit.ParseBepInEx(fixture(t)), ConfigFile{Name: "com.example.betterstuff.cfg"})
	}
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

func TestRefusedValueIsInvalidAndNamesItsProblem(t *testing.T) {
	d := cfgedit.ParseBepInEx(fixture(t))
	for _, c := range []struct{ section, key, value, want string }{
		{"General", "LeaveDelay", "5", "range"},
		{"General", "LeaveDelay", "1.5", "whole_number"},
		{"General", "ShowHud", "maybe", "bool"},
		{"Advanced", "Mode", "Nightmare", "enum"},
		{"General", "ScrapMultiplier", "x", "number"},
	} {
		_, err := d.Set(c.section, c.key, c.value)
		err = valueErr(err)
		detail, _ := usererr.DetailOf(err).(*valueError)
		if usererr.KindOf(err) != usererr.Invalid || detail == nil || detail.Problem != c.want || strings.Contains(err.Error(), "config_value") {
			t.Errorf("%s=%s: %v %+v", c.key, c.value, err, detail)
		}
	}
	_, err := d.Set("General", "LeaveDelay", "5")
	err = valueErr(err)
	if detail, _ := usererr.DetailOf(err).(*valueError); detail.Min == nil || *detail.Min != 10 || *detail.Max != 300 {
		t.Errorf("range bounds: %+v", detail)
	}
}

func TestMissingSettingIsNotFound(t *testing.T) {
	_, err := cfgedit.ParseBepInEx(fixture(t)).Set("General", "Gone", "1")
	if err = valueErr(err); usererr.KindOf(err) != usererr.NotFound || !strings.HasSuffix(err.Error(), "no setting Gone in [General]") {
		t.Fatalf("a missing setting: %v", err)
	}
}
