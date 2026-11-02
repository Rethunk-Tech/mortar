package devsvc

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func installedTS(name, version string, enabled, implicit bool) profile.Installed {
	return profile.Installed{Source: profile.Source{Kind: profile.KindThunderstore, Name: name, Version: version}, Enabled: enabled, Implicit: implicit}
}

func TestDiffDependencies(t *testing.T) {
	declared := []Dependency{
		{Package: "A-Needed", Version: "1.0.0"},
		{Package: "B-Soft", Version: "1.0.0", Soft: true},
		{Package: "C-Held", Version: "1.0.0"},
		{Package: "D-Old", Version: "1.0.0", Soft: true},
		{Package: "E-Unpinned"},
	}
	installed := []profile.Installed{
		installedTS("C-Held", "1.0.0", true, false),
		installedTS("D-Old", "1.2.0", true, false),
		installedTS("E-Unpinned", "3.0.0", true, false),
		installedTS("F-Extra", "2.0.0", true, false),
		installedTS("G-Off", "1.0.0", false, false),
		installedTS("H-Brought", "1.0.0", true, true),
		installedTS("I-Tolerated", "1.0.0", true, false),
		installedTS("Me-Mine", "0.1.0", true, false),
		installedTS("BepInEx-BepInExPack", "5.4.2100", true, false),
		installedTS("f-extra", "2.0.0", true, false),
		{Source: profile.Source{Kind: profile.KindNexus, Name: "Z-Nexus"}, Enabled: true},
	}
	got := diffDependencies(declared, installed, []string{"I-Tolerated"}, []string{"Me-Mine"}, false)
	want := DepSync{
		Missing:    []Dependency{{Package: "A-Needed", Version: "1.0.0"}},
		Undeclared: []DepPackage{{Package: "F-Extra", Version: "2.0.0"}},
		Differs:    []DepVersion{{Package: "D-Old", Declared: "1.0.0", Installed: "1.2.0", Soft: true}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestDependencySyncNamesTheDevelopmentProfile(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	prof, err := svc.Profiles.Create("lethal-company", "Dev")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	csproj := filepath.Join(dir, "src", "Mod", "Mod.csproj")
	if err := os.MkdirAll(filepath.Dir(csproj), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(csproj, []byte("<Project/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.Settings.AddDevProject("lethal-company", settings.DevProject{Dir: dir, Name: "Mod", Profile: prof.ID}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.DependencySync("lethal-company", dir)
	if err != nil || got.Profile != prof.ID {
		t.Fatalf("profile = %q, %v", got.Profile, err)
	}
}

func TestDiffDependenciesCountsAProjectBuiltFromItsOwnFolder(t *testing.T) {
	declared := []Dependency{{Package: "Mortar-ConfigSync", Version: "0.1.0"}}
	dev := profile.Installed{
		Source:   profile.Source{Kind: profile.KindLocal, Name: "debug"},
		Enabled:  true,
		UniqueID: string(mod.NewID(mod.FormatThunderstore, "Mortar-ConfigSync")),
		Version:  "0.1.1",
	}
	got := diffDependencies(declared, []profile.Installed{dev}, nil, []string{"Mortar-ConfigSync"}, false)
	if len(got.Missing) != 0 || len(got.Differs) != 0 || len(got.Undeclared) != 0 {
		t.Fatalf("a dev build counts as present and is not offered an update: %+v", got)
	}
}

func TestDiffDependenciesInASharedProfileReportsNothingUndeclared(t *testing.T) {
	declared := []Dependency{{Package: "A-Needed", Version: "1.0.0"}}
	installed := []profile.Installed{installedTS("F-Extra", "2.0.0", true, false)}
	got := diffDependencies(declared, installed, nil, nil, true)
	if len(got.Undeclared) != 0 || len(got.Missing) != 1 {
		t.Fatalf("got %+v", got)
	}
	if got := diffDependencies(declared, installed, nil, nil, false); len(got.Undeclared) != 1 {
		t.Fatalf("a project alone in its profile still hears about extras: %+v", got)
	}
}
