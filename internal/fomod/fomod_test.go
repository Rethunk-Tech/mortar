package fomod

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/Rethunk-Tech/go-safefs"
)

func loadFixture(t *testing.T, name string) Config {
	t.Helper()
	var (
		b   []byte
		err error
	)
	switch name {
	case "choose-one.xml":
		b, err = fs.ReadFile(os.DirFS("testdata"), "choose-one.xml")
	case "flags.xml":
		b, err = fs.ReadFile(os.DirFS("testdata"), "flags.xml")
	case "required.xml":
		b, err = fs.ReadFile(os.DirFS("testdata"), "required.xml")
	case "folder.xml":
		b, err = fs.ReadFile(os.DirFS("testdata"), "folder.xml")
	default:
		t.Fatalf("unknown fixture %s", name)
	}
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestChooseOne(t *testing.T) {
	cfg := loadFixture(t, "choose-one.xml")
	ops := Resolve(cfg, Choices{"Options": {"Pack": {"Beta"}}}, EvalContext{})
	if len(ops) != 1 || ops[0].Source != "beta/manifest.json" {
		t.Fatalf("ops = %+v", ops)
	}
	if Match(cfg, Choices{"Options": {"Pack": {"Nope"}}}, EvalContext{}) {
		t.Fatal("vanished choice matched")
	}
	if !Match(cfg, Choices{"Options": {"Pack": {"Alpha"}}}, EvalContext{}) {
		t.Fatal("alpha should match")
	}
}

func TestFlagsConditionalInstall(t *testing.T) {
	cfg := loadFixture(t, "flags.xml")
	on := Resolve(cfg, Choices{"Pick": {"Mode": {"On"}}}, EvalContext{})
	if len(on) != 1 || on[0].Source != "extra.txt" {
		t.Fatalf("flag on: %+v", on)
	}
	off := Resolve(cfg, Choices{"Pick": {"Mode": {"Off"}}}, EvalContext{})
	if len(off) != 0 {
		t.Fatalf("flag off: %+v", off)
	}
}

func TestRequiredFiles(t *testing.T) {
	cfg := loadFixture(t, "required.xml")
	none := Resolve(cfg, Choices{"More": {"Extra": nil}}, EvalContext{})
	if len(none) != 1 || none[0].Source != "core.txt" {
		t.Fatalf("required only: %+v", none)
	}
	both := Resolve(cfg, Choices{"More": {"Extra": {"Addon"}}}, EvalContext{})
	if len(both) != 2 {
		t.Fatalf("with addon: %+v", both)
	}
}

func TestRequiredPluginIsResolvedWithoutChoice(t *testing.T) {
	cfg, err := Parse([]byte(`<config><installSteps><installStep name="S"><optionalFileGroups>
		<group name="G" type="SelectExactlyOne"><plugins>
			<plugin name="Required"><files><file source="required.txt" destination="required.txt"/></files>
				<typeDescriptor><type name="Required"/></typeDescriptor></plugin>
			<plugin name="Other"><files><file source="other.txt" destination="other.txt"/></files>
				<typeDescriptor><type name="Optional"/></typeDescriptor></plugin>
		</plugins></group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	ops := Resolve(cfg, nil, EvalContext{})
	if len(ops) != 1 || ops[0].Source != "required.txt" {
		t.Fatalf("required plugin = %+v", ops)
	}
	if !Match(cfg, nil, EvalContext{}) {
		t.Fatal("required-only group should match")
	}
}

func TestUnchosenPluginsInstallTheirAlwaysAndUsableFiles(t *testing.T) {
	cfg, err := Parse([]byte(`<config><installSteps><installStep name="S"><optionalFileGroups>
		<group name="G" type="SelectAny"><plugins order="Explicit">
			<plugin name="Usable"><files><file source="always.txt" alwaysInstall="true"/><file source="usable.txt" installIfUsable="1"/>
				<file source="chosen.txt"/></files><typeDescriptor><type name="Optional"/></typeDescriptor></plugin>
			<plugin name="Unusable"><files><file source="always2.txt" alwaysInstall="true"/><file source="usable2.txt" installIfUsable="true"/></files>
				<typeDescriptor><type name="NotUsable"/></typeDescriptor></plugin>
		</plugins></group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, op := range Resolve(cfg, Choices{"S": {"G": nil}}, EvalContext{}) {
		got = append(got, op.Source)
	}
	if strings.Join(got, ",") != "always.txt,usable.txt,always2.txt" {
		t.Fatalf("ops = %v", got)
	}
}

func TestFindConfigDepthAndUTF16(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "inner", "fomod")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	u := utf16.Encode([]rune(`<?xml version="1.0"?><config><moduleName>Wide</moduleName></config>`))
	raw := make([]byte, 2+len(u)*2)
	raw[0], raw[1] = 0xFF, 0xFE
	for i, r := range u {
		binary.LittleEndian.PutUint16(raw[2+i*2:], r)
	}
	path := filepath.Join(dir, "ModuleConfig.xml")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := FindConfig(root)
	if err != nil || got != path {
		t.Fatalf("FindConfig = %q %v", got, err)
	}
	cfg, err := Load(path)
	if err != nil || cfg.ModuleName != "Wide" {
		t.Fatalf("Load = %+v %v", cfg, err)
	}
}

func TestFileDependencyUnmetWithoutIndex(t *testing.T) {
	cfg, err := Parse([]byte(`<config><moduleName>D</moduleName>
		<installSteps><installStep name="S"><visible>
			<fileDependency file="other.dll" state="Active"/>
		</visible><optionalFileGroups><group name="G" type="SelectAny">
			<plugins><plugin name="P"><files><file source="p.txt" destination="p.txt"/></files>
			<typeDescriptor><type name="Optional"/></typeDescriptor></plugin></plugins>
		</group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(VisibleSteps(cfg, nil, EvalContext{})) != 0 {
		t.Fatal("missing file should hide the step")
	}
	files := FileIndex(func(name string) string {
		if name == "other.dll" {
			return FileActive
		}
		return FileMissing
	})
	if len(VisibleSteps(cfg, nil, EvalContext{Files: files})) != 1 {
		t.Fatal("present file should show the step")
	}
}

func TestGameDependencyUnknownVersionUnmet(t *testing.T) {
	cfg, err := Parse([]byte(`<config><moduleName>D</moduleName>
		<installSteps><installStep name="S"><visible>
			<gameDependency version="1.6.0"/>
		</visible><optionalFileGroups><group name="G" type="SelectAny">
			<plugins><plugin name="P"><files><file source="p.txt" destination="p.txt"/></files>
			<typeDescriptor><type name="Optional"/></typeDescriptor></plugin></plugins>
		</group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(VisibleSteps(cfg, nil, EvalContext{})) != 0 {
		t.Fatal("unknown game version should hide the step")
	}
}

func TestGameDependencyMetWhenInstalledVersionSufficient(t *testing.T) {
	cfg, err := Parse([]byte(`<config><moduleName>D</moduleName>
		<installSteps><installStep name="S"><visible>
			<gameDependency version="1.6.0"/>
		</visible><optionalFileGroups><group name="G" type="SelectAny">
			<plugins><plugin name="P"><files><file source="p.txt" destination="p.txt"/></files>
			<typeDescriptor><type name="Optional"/></typeDescriptor></plugin></plugins>
		</group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := EvalContext{GameVersion: "1.6.15"}
	if len(VisibleSteps(cfg, nil, ctx)) != 1 {
		t.Fatal("installed game version should satisfy minimum")
	}
	if len(VisibleSteps(cfg, nil, EvalContext{GameVersion: "1.5.0"})) != 0 {
		t.Fatal("older game version should not satisfy minimum")
	}
}

func TestResolveSortsExtremePriorities(t *testing.T) {
	cfg := Config{Required: []Op{{Source: "high", Priority: math.MaxInt}, {Source: "low", Priority: -1}}}
	ops := Resolve(cfg, nil, EvalContext{})
	if ops[0].Source != "low" || ops[1].Source != "high" {
		t.Fatalf("ops = %+v", ops)
	}
}

func TestFolderElementSetsFolder(t *testing.T) {
	cfg := loadFixture(t, "folder.xml")
	if len(cfg.Required) != 1 || !cfg.Required[0].Folder || cfg.Required[0].Source != "pack" {
		t.Fatalf("required = %+v", cfg.Required)
	}
}

func TestRequiredPluginNotUsableUnderFlagsMatches(t *testing.T) {
	cfg, err := Parse([]byte(`<config><installSteps>
		<installStep name="A"><optionalFileGroups><group name="G" type="SelectExactlyOne"><plugins>
			<plugin name="On"><conditionFlags><flag name="f">1</flag></conditionFlags>
				<typeDescriptor><type name="Optional"/></typeDescriptor></plugin>
		</plugins></group></optionalFileGroups></installStep>
		<installStep name="B"><optionalFileGroups><group name="H" type="SelectAny"><plugins>
			<plugin name="R"><files><file source="r.txt"/></files><typeDescriptor><dependencyType>
				<defaultType name="Required"/><patterns><pattern><dependencies><flagDependency flag="f" value="1"/></dependencies>
				<type name="NotUsable"/></pattern></patterns></dependencyType></typeDescriptor></plugin>
		</plugins></group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	choices := Choices{"A": {"G": {"On"}}}
	if !Match(cfg, choices, EvalContext{}) {
		t.Fatal("a plugin NotUsable under the flags must not be forced in, so the choices still match")
	}
	for _, op := range Resolve(cfg, choices, EvalContext{}) {
		if op.Source == "r.txt" {
			t.Fatal("NotUsable plugin installed")
		}
	}
}

func TestLargeGroupIsLinear(t *testing.T) {
	run := func(n int) time.Duration {
		g := Group{Name: "G", Type: SelectAny}
		names := make([]string, n)
		for i := range n {
			names[i] = "p" + strconv.Itoa(i)
			g.Plugins = append(g.Plugins, Plugin{Name: names[i], TypeName: TypeOptional, Files: []Op{{Source: names[i]}}})
		}
		cfg := Config{Steps: []Step{{Name: "S", Groups: []Group{g}}}}
		choices := Choices{"S": {"G": names}}
		start := time.Now()
		if !Match(cfg, choices, EvalContext{}) {
			t.Fatal("no match")
		}
		if got := len(Resolve(cfg, choices, EvalContext{})); got != n {
			t.Fatalf("ops = %d", got)
		}
		return time.Since(start)
	}
	// A ratio rather than a wall-clock limit, so a loaded runner passes: 8x the plugins costs about 8x when linear
	// and 64x when quadratic.
	small, large := run(5000), run(40000)
	if ratio := float64(large) / float64(small); ratio > 24 {
		t.Fatalf("8x the plugins took %.1fx as long (%v vs %v)", ratio, large, small)
	}
}

func utf16Bytes(s string, order binary.AppendByteOrder) []byte {
	var raw []byte
	for _, u := range utf16.Encode([]rune(s)) {
		raw = order.AppendUint16(raw, u)
	}
	return raw
}

func TestDecodeXMLBOMlessUTF16(t *testing.T) {
	body := `<config><moduleName>Wide</moduleName></config>`
	for _, doc := range []string{`<?xml version="1.0" encoding="utf-16"?>` + body, body, "\n  <?xml version=\"1.0\"?>" + body} {
		for name, order := range map[string]binary.AppendByteOrder{"LE": binary.LittleEndian, "BE": binary.BigEndian} {
			cfg, err := Parse(decodeXML(utf16Bytes(doc, order)))
			if err != nil || cfg.ModuleName != "Wide" {
				t.Fatalf("%s %q: %+v %v", name, doc, cfg, err)
			}
		}
	}
}

func TestParseDeclaredAndUndeclaredLegacyEncodings(t *testing.T) {
	for _, decl := range []string{`encoding="utf-16"`, `encoding="UTF-16"`} {
		cfg, err := Parse([]byte(`<?xml version="1.0" ` + decl + `?><config><moduleName>Caf` + "é" + `</moduleName></config>`))
		if err != nil || cfg.ModuleName != "Café" {
			t.Fatalf("%s: %+v %v", decl, cfg, err)
		}
	}
	for _, decl := range []string{`<?xml version="1.0" encoding="windows-1252"?>`, `<?xml version="1.0" encoding="ISO-8859-1"?>`, ""} {
		cfg, err := Parse([]byte(decl + "<config><moduleName>Caf\xe9</moduleName></config>"))
		if err != nil || cfg.ModuleName != "Café" {
			t.Fatalf("%q: %+v %v", decl, cfg, err)
		}
	}
}

func TestParseDecidesEncodingByBytes(t *testing.T) {
	for _, c := range []struct{ doc, want string }{
		{"<?xml version=\"1.0\" encoding=\"windows-1252\"?><config><moduleName>W\u00efde</moduleName></config>", "W\u00efde"},
		{"<?xml version=\"1.0\" encoding=\"utf-8\"?><config><moduleName>Caf\xe9</moduleName></config>", "Caf\u00e9"},
		{"<?xml version=\"1.0\" encoding=\"us-ascii\"?><config><moduleName>Plain</moduleName></config>", "Plain"},
		{"<?xml version=\"1.0\" encoding=\"windows-1252\"?><config><moduleName>Caf\xe9</moduleName></config>", "Caf\u00e9"},
	} {
		cfg, err := Parse([]byte(c.doc))
		if err != nil || cfg.ModuleName != c.want {
			t.Fatalf("%q: %+v %v", c.doc, cfg, err)
		}
	}
}

func TestLoadRefusesOversizedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ModuleConfig.xml")
	if err := os.WriteFile(path, make([]byte, maxConfigBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); !errors.Is(err, safefs.ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

func orderConfig(t *testing.T, steps, groups, plugins string) Config {
	t.Helper()
	attr := func(o string) string {
		if o == "" {
			return ""
		}
		return ` order="` + o + `"`
	}
	plug := func(n string) string {
		return `<plugin name="` + n + `"><files><file source="` + n + `" priority="1"/></files>` +
			`<typeDescriptor><type name="Optional"/></typeDescriptor></plugin>`
	}
	grp := func(n string) string {
		return `<group name="` + n + `" type="SelectAny"><plugins` + attr(plugins) + `>` + plug("b") + plug("C") + plug("a") + `</plugins></group>`
	}
	step := func(n string) string {
		return `<installStep name="` + n + `"><optionalFileGroups` + attr(groups) + `>` + grp("y") + grp("z") + grp("x") +
			`</optionalFileGroups></installStep>`
	}
	cfg, err := Parse([]byte(`<config><installSteps` + attr(steps) + `>` + step("2") + step("3") + step("1") + `</installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestOrderAttribute(t *testing.T) {
	for _, c := range []struct {
		order, wantOrder string
	}{
		{"", OrderAscending},
		{"Ascending", OrderAscending},
		{"Descending", OrderDescending},
		{"Explicit", OrderExplicit},
	} {
		// Capitals sort before lowercase, as in MO2.
		want := map[string]string{"": "Cab", "Ascending": "Cab", "Descending": "baC", "Explicit": "bCa"}[c.order]
		t.Run(c.order, func(t *testing.T) {
			steps := orderConfig(t, c.order, "", "")
			if got := stepNames(steps); got != orderOf(c.order, "231", "123") {
				t.Fatalf("steps = %s", got)
			}
			if steps.StepOrder != c.wantOrder {
				t.Fatalf("StepOrder = %s", steps.StepOrder)
			}
			groups := orderConfig(t, "Explicit", c.order, "")
			if got := groupNames(groups.Steps[0]); got != orderOf(c.order, "yzx", "xyz") {
				t.Fatalf("groups = %s", got)
			}
			if groups.Steps[0].GroupOrder != c.wantOrder {
				t.Fatalf("GroupOrder = %s", groups.Steps[0].GroupOrder)
			}
			plugins := orderConfig(t, "Explicit", "Explicit", c.order)
			g := plugins.Steps[0].Groups[0]
			var got string
			for _, p := range g.Plugins {
				got += p.Name
			}
			if got != want {
				t.Fatalf("plugins = %s, want %s", got, want)
			}
			if g.PluginOrder != c.wantOrder {
				t.Fatalf("PluginOrder = %s", g.PluginOrder)
			}
		})
	}
}

// orderOf is the document-order result for Explicit, the ascending one for Ascending or unset, else descending.
func orderOf(order, doc, asc string) string {
	switch order {
	case "Explicit":
		return doc
	case "Descending":
		b := []byte(asc)
		slices.Reverse(b)
		return string(b)
	default:
		return asc
	}
}

func stepNames(c Config) string {
	var s string
	for _, st := range c.Steps {
		s += st.Name
	}
	return s
}

func groupNames(st Step) string {
	var s string
	for _, g := range st.Groups {
		s += g.Name
	}
	return s
}

func TestOrderDecidesResolveTie(t *testing.T) {
	cfg, err := Parse([]byte(`<config><installSteps><installStep name="S"><optionalFileGroups>` +
		`<group name="G" type="SelectAll"><plugins>` +
		`<plugin name="b"><files><file source="b" destination="x" priority="1"/></files></plugin>` +
		`<plugin name="a"><files><file source="a" destination="x" priority="1"/></files></plugin>` +
		`</plugins></group></optionalFileGroups></installStep></installSteps></config>`))
	if err != nil {
		t.Fatal(err)
	}
	ops := Resolve(cfg, Choices{}, EvalContext{})
	if len(ops) != 2 || ops[0].Source != "a" || ops[1].Source != "b" {
		t.Fatalf("ops = %+v", ops)
	}
}

func TestFindConfigPrefersTheExactNameAmongCaseVariants(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"FOMOD/ModuleConfig.xml", "fomod/moduleconfig.xml"} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("<config/>"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := FindConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "fomod", "moduleconfig.xml"); got != want {
		t.Fatalf("FindConfig = %q, want %q", got, want)
	}
}
