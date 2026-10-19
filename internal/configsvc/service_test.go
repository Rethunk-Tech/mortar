package configsvc

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/modconfig"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

type fakeProfiles struct {
	dir, folder, config, shipped string
	mods                         []profile.Mod
	installed                    []string
}

func (f fakeProfiles) ProfileDir(string, string) (string, error) { return f.dir, nil }
func (f fakeProfiles) ModFolder(_, _, _ string, id mod.ID) (string, error) {
	if id != "Author.Mod" {
		return "", errors.New("no such mod")
	}
	return f.folder, nil
}

func (f fakeProfiles) ReadConfig(string, string, string, mod.ID) (string, error) {
	return f.config, nil
}

func (f fakeProfiles) ShippedConfig(string, string, string, mod.ID) (string, bool) {
	return f.shipped, f.shipped != ""
}

func (fakeProfiles) ContentSchema(string, string, string, mod.ID) (modconfig.Schema, error) {
	return modconfig.Schema{}, nil
}

func (f fakeProfiles) UserMods(string, string) ([]profile.Mod, error) { return f.mods, nil }

func (f fakeProfiles) ConfigView(string, string) (profile.ConfigView, error) {
	return profile.ConfigView{
		ModFolder:     func(key string, id mod.ID) (string, error) { return f.ModFolder("", "", key, id) },
		ReadConfig:    func(key string, id mod.ID) (string, error) { return f.ReadConfig("", "", key, id) },
		ShippedConfig: func(key string, id mod.ID) (string, bool) { return f.ShippedConfig("", "", key, id) },
		PluginGUIDs:   func(id mod.ID) []string { return f.PluginGUIDs("", "", id) },
	}, nil
}

func (fakeProfiles) SeedConfigs(string, string) error { return nil }

func (f fakeProfiles) InstalledPluginGUIDs(string, string) []string { return f.installed }

func (fakeProfiles) PluginGUIDs(_, _ string, id mod.ID) []string {
	switch id {
	case "thunderstore:Ex-BetterStuff":
		return []string{"com.example.BetterStuff"}
	case "folder:carryon":
		return []string{"carryon"}
	}
	return nil
}

func TestFilesOfAPackageAreItsPluginsConfigs(t *testing.T) {
	s, _ := newService(t)
	files, err := s.Files("g", "p", "thunderstore:Ex-BetterStuff")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "com.example.betterstuff.cfg" {
		t.Fatalf("files = %+v, want the plugin's .cfg", files)
	}
	if files, _ := s.Files("g", "p", "thunderstore:Ex-Other"); len(files) != 0 {
		t.Fatalf("another package's files = %+v, want none", files)
	}
}

func newService(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "BepInEx", "config")
	if err := os.MkdirAll(cfg, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "com.example.betterstuff.cfg"), []byte(fixture(t)), 0o600); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(dir, "mods", "m")
	if err := os.MkdirAll(folder, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "config.json"), []byte(`{"Speed": 2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	p := fakeProfiles{dir: dir, folder: folder, config: `{"Speed": 2}`, shipped: `{"Speed": 1}`}
	return &Service{Profiles: p}, filepath.Join(cfg, "com.example.betterstuff.cfg")
}

func TestFilesSetResetOnABepInExFile(t *testing.T) {
	s, path := newService(t)
	files, err := s.Files("lc", "p", "com.example.betterstuff")
	if err != nil || len(files) != 1 || files[0].Format != FormatBepInEx || files[0].Label != "LC Better Stuff" {
		t.Fatalf("files %+v %v", files, err)
	}
	if err := s.Set("lc", "p", "com.example.betterstuff", files[0].Name, "General", "LeaveDelay", "60"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("lc", "p", "com.example.betterstuff", files[0].Name, "General", "LeaveDelay", "9999"); err == nil {
		t.Fatal("out of range must be refused")
	}
	if n, err := s.ResetAll("lc", "p", "com.example.betterstuff", files[0].Name); err != nil || n != 0 {
		t.Fatalf("reset all: %d %v", n, err)
	}
	got, _ := os.ReadFile(filepath.Clean(path))
	want := strings.NewReplacer(
		"LeaveDelay = 120", "LeaveDelay = 90",
		"ScrapMultiplier = 1.25", "ScrapMultiplier = 1",
		"Mode = Hard", "Mode = Normal",
		"Label = Night Shift", "Label = Crew",
		"Tint = FF8800FF", "Tint = FFFFFFFF",
	).Replace(fixture(t))
	if string(got) != want {
		t.Fatalf("reset all must change only values:\n%s", got)
	}
}

func TestJSONFileGoesThroughTheProfileSetter(t *testing.T) {
	s, _ := newService(t)
	var field, value string
	s.SetJSON = func(_, _ string, _ mod.ID, f, v string) error { field, value = f, v; return nil }
	files, _ := s.Files("stardew", "p", "Author.Mod")
	if len(files) != 1 || files[0].Format != FormatSMAPI {
		t.Fatalf("files %+v", files)
	}
	if err := s.Reset("stardew", "p", "Author.Mod", "config.json", "", "Speed"); err != nil || field != "Speed" || value != "1" {
		t.Fatalf("reset wrote %q=%q: %v", field, value, err)
	}
	s.Running = func(string, string) bool { return true }
	if err := s.Set("stardew", "p", "Author.Mod", "config.json", "", "Speed", "3"); err == nil {
		t.Fatal("a running game must block writes")
	}
}

func TestResetAllLeavesEntriesWithoutADefaultAndCountsThem(t *testing.T) {
	s, _ := newService(t)
	base := s.Profiles
	s.Profiles = noExtra{base, `{"Speed": 2, "Extra": "x"}`}
	var wrote []string
	s.SetJSON = func(_, _ string, _ mod.ID, f, v string) error { wrote = append(wrote, f+"="+v); return nil }
	n, err := s.ResetAll("stardew", "p", "Author.Mod", "config.json")
	if err != nil || n != 1 || len(wrote) != 1 || wrote[0] != "Speed=1" {
		t.Fatalf("skipped %d wrote %v err %v", n, wrote, err)
	}
}

type noExtra struct {
	Profiles
	config string
}

func (n noExtra) ReadConfig(string, string, string, mod.ID) (string, error) { return n.config, nil }

func TestSetRefusesNamesOutsideTheConfigFolder(t *testing.T) {
	s, _ := newService(t)
	for _, name := range []string{"../../evil.cfg", "sub/x.cfg", "x.json", "/etc/passwd"} {
		if _, err := s.cfgPath("lethal-company", "p", name); err == nil {
			t.Errorf("cfgPath accepted %q", name)
		}
	}
}

func TestSMAPITypesComeFromTheValueThenTheShippedDefault(t *testing.T) {
	t.Parallel()
	shipped := `{"Enabled":true,"Count":3,"Ratio":0.5,"Name":"x","Mode":true}`
	current := `{"Enabled":null,"Count":"7","Ratio":0.25,"Name":"y","Mode":"maybe","Extra":false}`
	sc, err := jsonSchema(ConfigFile{Name: jsonName, Format: FormatSMAPI}, current, shipped, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range sc.Sections[0].Entries {
		got[e.Key] = e.Type
	}
	want := map[string]string{"Enabled": TypeBool, "Count": TypeInt, "Ratio": TypeFloat, "Name": TypeString, "Mode": TypeString, "Extra": TypeBool}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s is %s, want %s", k, got[k], w)
		}
	}
}

func TestContentPatcherFieldsTypeFromConfigSchema(t *testing.T) {
	t.Parallel()
	cp, err := modconfig.Parse([]byte(`{"ConfigSchema":{
		"MistEffects":{"AllowValues":"true, false","Default":"true","Description":"Mist"},
		"Mixed":{"AllowValues":"False, True","Default":"False"},
		"Season":{"AllowValues":"Spring, Summer","Default":"Spring"},
		"Crops":{"AllowValues":"Corn, Kale","AllowMultiple":true,"Default":"Corn"},
		"Nick":{},
		"NotYet":{"AllowValues":"true, false","Default":"false"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	current := `{"MistEffects":"true","Mixed":"True","Season":"Summer","Crops":"Corn, Kale","Nick":"default"}`
	sc, err := jsonSchema(ConfigFile{Name: jsonName, Format: FormatSMAPI}, current, "", nil, cp)
	if err != nil {
		t.Fatal(err)
	}
	type shape struct {
		typ, def string
		values   int
		flags    bool
	}
	got := map[string]shape{}
	for _, e := range sc.Sections[0].Entries {
		got[e.Key] = shape{e.Type, e.Default, len(e.Values), e.Flags}
	}
	want := map[string]shape{
		"MistEffects": {TypeBool, "true", 0, false},
		"Mixed":       {TypeBool, "False", 0, false},
		"Season":      {TypeEnum, "Spring", 2, false},
		"Crops":       {TypeEnum, "Corn", 2, true},
		"Nick":        {TypeString, "", 0, false},
		"NotYet":      {TypeBool, "false", 0, false},
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s = %+v, want %+v", k, got[k], w)
		}
	}
}

// A Vintage Story mod's settings are the profile's copy of ModConfig, matched to the mod by the mod id its
// modinfo.json gives, in the file name's own spelling.
func TestModConfigJSONIsMatchedByModIDAndEditedInPlace(t *testing.T) {
	s, _ := newService(t)
	p, ok := s.Profiles.(fakeProfiles)
	if !ok {
		t.Fatal("not the fake profile store")
	}
	mc := filepath.Join(p.dir, "ModConfig")
	if err := os.MkdirAll(mc, 0o750); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"CarryOnConfig.json": `{"InteractDistance": 2.5, "Allowed": true}`, "OtherMod.json": `{"A": 1}`} {
		if err := os.WriteFile(filepath.Join(mc, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p.mods = []profile.Mod{{ID: "folder:carryon", Name: "Carry On", Enabled: true}}
	s.Profiles = p
	files, err := s.Files("vintage-story", "p", "folder:carryon")
	if err != nil || len(files) != 1 || files[0].Name != "CarryOnConfig.json" || files[0].Format != FormatJSON {
		t.Fatalf("files = %+v, %v", files, err)
	}
	if err := s.Set("vintage-story", "p", "folder:carryon", "CarryOnConfig.json", "", "InteractDistance", "3"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Clean(filepath.Join(mc, "CarryOnConfig.json"))); !strings.Contains(string(got), `"InteractDistance": 3`) || strings.Index(string(got), "InteractDistance") > strings.Index(string(got), "Allowed") {
		t.Fatalf("file = %s", got)
	}
	list, err := s.Mods("vintage-story", "p")
	if err != nil || len(list.Mods) != 1 || len(list.Other) != 2 || !slices.ContainsFunc(list.Other, func(f ModFile) bool { return f.Name == "OtherMod.json" }) {
		t.Fatalf("list = %+v, %v", list, err)
	}
}

// A mod's other text files in BepInEx's config folder (its package's subfolder, or a file named for it) are listed for
// that mod and edited whole; anything else is listed as nobody's.
func TestTextFilesAreMatchedByPackageAndEditedWhole(t *testing.T) {
	s, cfgPath := newService(t)
	cfg := filepath.Dir(cfgPath)
	for name, body := range map[string]string{
		"Ex-BetterStuff/rules.yaml": "speed: 2\n",
		"BetterStuff.toml":          "speed = 2\n",
		"Unrelated.xml":             "\ufeff<x/>",
		"Ex-BetterStuff/icon.png":   "png",
	} {
		p := filepath.Join(cfg, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const id = "thunderstore:Ex-BetterStuff"
	files, err := s.Files("g", "p", id)
	if err != nil {
		t.Fatal(err)
	}
	var text []string
	for _, f := range files {
		if f.Format == FormatText {
			text = append(text, f.Name)
		}
	}
	slices.Sort(text)
	if !slices.Equal(text, []string{"BetterStuff.toml", "Ex-BetterStuff/rules.yaml"}) {
		t.Fatalf("text files = %v", text)
	}
	list, err := s.Mods("g", "p")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(list.Other, func(f ModFile) bool { return f.Name == "Unrelated.xml" && f.Format == FormatText }) {
		t.Fatalf("other files = %+v, want Unrelated.xml", list.Other)
	}
	if got, err := s.Text("g", "p", id, "Ex-BetterStuff/rules.yaml"); err != nil || got != "speed: 2\n" {
		t.Fatalf("text %q %v", got, err)
	}
	if err := s.SetText("g", "p", id, "Ex-BetterStuff/rules.yaml", "speed: 3\n"); err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(filepath.Join(cfg, "Ex-BetterStuff", "rules.yaml")); string(b) != "speed: 3\n" {
		t.Fatalf("written %q", b)
	}
	if got, _ := s.Text("g", "p", "", "Unrelated.xml"); got != "<x/>" {
		t.Fatalf("a BOM reached the editor: %q", got)
	}
	if err := s.SetText("g", "p", "", "Unrelated.xml", "<y/>"); err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(filepath.Join(cfg, "Unrelated.xml")); string(b) != "\ufeff<y/>" {
		t.Fatalf("the BOM was not kept: %q", b)
	}
	if err := s.SetText("g", "p", id, "../escape.yaml", "x"); err == nil {
		t.Fatal("a name outside the config folder was written")
	}
	s.Running = func(string, string) bool { return true }
	if err := s.SetText("g", "p", id, "BetterStuff.toml", "x"); err == nil {
		t.Fatal("a running game's file was written")
	}
}
