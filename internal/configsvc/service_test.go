package configsvc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

type fakeProfiles struct{ dir, folder, config, shipped string }

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

func (fakeProfiles) SeedConfigs(string, string) error { return nil }

func (fakeProfiles) PluginGUIDs(_, _ string, id mod.ID) []string {
	if id == "thunderstore:Ex-BetterStuff" {
		return []string{"com.example.BetterStuff"}
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
