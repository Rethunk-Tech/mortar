package devsvc

import (
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
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
	got := diffDependencies(declared, installed, []string{"I-Tolerated"}, []string{"Me-Mine"})
	want := DepSync{
		Missing:    []Dependency{{Package: "A-Needed", Version: "1.0.0"}},
		Undeclared: []DepPackage{{Package: "F-Extra", Version: "2.0.0"}},
		Differs:    []DepVersion{{Package: "D-Old", Declared: "1.0.0", Installed: "1.2.0", Soft: true}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}
