package configsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-dotnet"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// testdata/bound-settings.dll is go-dotnet's compiled fixture: plugin com.fixture.plugin, whose code binds an int with a
// range (Numbers.Count, 12 from 1 to 100), a float (Numbers.Speed, 1.5) and an enum ConfigSync shares (Shared.Mode, Wild).
const boundGUID = "com.fixture.plugin"

// boundService returns the service, the path of the plugin's .cfg and of its DLL.
func boundService(t *testing.T) (svc *Service, cfg, dll string) {
	t.Helper()
	s, _ := newService(t)
	raw, err := safefs.ReadFile(filepath.Join("testdata", "bound-settings.dll"))
	if err != nil {
		t.Fatal(err)
	}
	// A copy, so a test can set its time.
	dll = filepath.Join(t.TempDir(), "bound-settings.dll")
	if err := os.WriteFile(dll, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	p, ok := s.Profiles.(fakeProfiles)
	if !ok {
		t.Fatal("the service's profiles are not the fake")
	}
	p.dlls = map[string]profile.PluginDLL{boundGUID: {Path: dll, Name: "Fixture Plugin", Version: "1.2.3"}}
	s.Profiles = p
	return s, filepath.Join(p.dir, "BepInEx", "config", boundGUID+".cfg"), dll
}

func entry(t *testing.T, sc Schema, section, key string) Entry {
	t.Helper()
	for _, sec := range sc.Sections {
		for _, e := range sec.Entries {
			if sec.Name == section && e.Key == key {
				return e
			}
		}
	}
	t.Fatalf("no %s.%s in %+v", section, key, sc)
	return Entry{}
}

func TestAPluginTheGameHasNotRunShowsItsSettingsAndTheFirstEditWritesItsFile(t *testing.T) {
	s, path, _ := boundService(t)
	files, err := s.Files("lc", "p", "thunderstore:Ex-Fixture")
	if err != nil || len(files) != 1 || files[0].Name != boundGUID+".cfg" || files[0].Label != "Fixture Plugin" {
		t.Fatalf("files %+v %v, want the plugin's unwritten .cfg", files, err)
	}
	sc, err := s.Schema("lc", "p", "thunderstore:Ex-Fixture", files[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	count := entry(t, sc, "Numbers", "Count")
	if count.Value != "12" || count.Default != "12" || count.Type != TypeInt || count.Min == nil || *count.Max != 100 {
		t.Fatalf("Count = %+v, want 12 from 1 to 100", count)
	}
	if mode := entry(t, sc, "Shared", "Mode"); mode.Type != TypeEnum || mode.Value != "Wild" || mode.Sync != "hostEnforced" {
		t.Fatalf("Mode = %+v, want the shared enum at Wild", mode)
	}
	if err := s.Set("lc", "p", "thunderstore:Ex-Fixture", files[0].Name, "Numbers", "Count", "20"); err != nil {
		t.Fatal(err)
	}
	raw, err := safefs.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if text := string(raw); !strings.Contains(text, "## Plugin GUID: com.fixture.plugin") || !strings.Contains(text, "Count = 20") || !strings.Contains(text, "Speed = 1.5") {
		t.Fatalf("written file:\n%s", text)
	}
}

// The .cfg is the newer file here: a game run with the old build wrote it, then the new build was installed with its
// build time. The DLL still wins.
func TestTheDLLGivesTheDefaultsAndSettingsOverAnOlderBuildsCfg(t *testing.T) {
	s, path, dll := boundService(t)
	old := "## Plugin GUID: com.fixture.plugin\n\n[Numbers]\n\n# Setting type: Int32\n# Default value: 5\nCount = 5\n"
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(dll, past, past); err != nil {
		t.Fatal(err)
	}
	sc, err := s.Schema("lc", "p", "thunderstore:Ex-Fixture", boundGUID+".cfg")
	if err != nil {
		t.Fatal(err)
	}
	if count := entry(t, sc, "Numbers", "Count"); count.Value != "5" || count.Default != "12" {
		t.Fatalf("Count = %+v, want the file's 5 against the DLL's default 12", count)
	}
	if speed := entry(t, sc, "Numbers", "Speed"); speed.Value != "1.5" {
		t.Fatalf("Speed = %+v, want the setting the file lacks at its default", speed)
	}
	if err := s.Set("lc", "p", "thunderstore:Ex-Fixture", boundGUID+".cfg", "Numbers", "Speed", "2"); err != nil {
		t.Fatal(err)
	}
	raw, _ := safefs.ReadFile(path)
	if !strings.Contains(string(raw), "Speed = 2") || !strings.Contains(string(raw), "Count = 5") {
		t.Fatalf("written file:\n%s", raw)
	}
}

// A bound setting whose section, key or default would break the .cfg's lines is left out of the rendered file.
func TestRenderSkipsSettingsThatWouldBreakTheFile(t *testing.T) {
	t.Parallel()
	good := dotnet.Setting{Section: "S", Key: "ok", Type: "int", Default: "1"}
	b := bound{guid: "g", settings: []dotnet.Setting{
		good,
		{Section: "S", Key: "a\nb", Type: "int", Default: "1"},
		{Section: "S", Key: "k", Type: "string", Default: "x\ny = z"},
		{Section: "S]\n[Evil", Key: "k", Type: "int", Default: "1"},
		{Section: "T]", Key: "k", Type: "int", Default: "1"},
	}}
	got := b.render()
	if want := "\n[S]\n" + entryText(good); !strings.HasSuffix(got, want) || strings.Count(got, "[") != 1 {
		t.Fatalf("rendered %q", got)
	}
}
