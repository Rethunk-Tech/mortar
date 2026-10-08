package configsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestModsListsOwnedConfigsOtherCfgsAndModsWithoutConfig(t *testing.T) {
	s, _ := newService(t)
	f, ok := s.Profiles.(fakeProfiles)
	if !ok {
		t.Fatal("newService builds a fakeProfiles")
	}
	cfg := filepath.Join(f.dir, "BepInEx", "config")
	if err := os.WriteFile(filepath.Join(cfg, "BepInEx.cfg"), []byte("[Logging]\n## Enable\n# Setting type: Boolean\n# Default value: true\nEnabled = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.mods = []profile.Mod{
		{Key: "m", ID: "Author.Mod", Name: "Author Mod", Enabled: true},
		{Key: "bs", ID: "thunderstore:Ex-BetterStuff", Name: "Better Stuff", Enabled: false},
		{Key: "n", ID: "Author.NoConfig", Name: "No Config", Enabled: true},
	}
	s.Profiles = f
	got, err := s.Mods("g", "p")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Mods) != 2 || got.Without != 1 {
		t.Fatalf("list = %+v", got)
	}
	smapi := got.Mods[0]
	if smapi.ID != "Author.Mod" || len(smapi.Files) != 1 || smapi.Files[0].Format != FormatSMAPI || !smapi.Files[0].Changed {
		t.Fatalf("SMAPI mod = %+v (Speed 2 differs from the shipped 1)", smapi)
	}
	f.config = `{"Speed": 1}`
	if err := os.WriteFile(filepath.Join(f.folder, "config.json"), []byte(f.config+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.Profiles = f
	again, err := s.Mods("g", "p")
	if err != nil || again.Mods[0].Files[0].Changed {
		t.Fatalf("after the file was reset to the shipped value: %+v, %v", again.Mods[0], err)
	}
	pkg := got.Mods[1]
	if pkg.Enabled || len(pkg.Files) != 1 || pkg.Files[0].Name != "com.example.betterstuff.cfg" || pkg.Files[0].Format != FormatBepInEx {
		t.Fatalf("package = %+v", pkg)
	}
	if len(got.Other) != 1 || got.Other[0].Name != "BepInEx.cfg" {
		t.Fatalf("other = %+v", got.Other)
	}
}
