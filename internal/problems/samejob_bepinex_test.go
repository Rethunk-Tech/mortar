package problems

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// Both packages ship testdata/mod.dll, whose Fixture.Patches skip Farmer.Update and transpile Game1.Draw and also
// only add a prefix and postfixes, which say nothing about the job.
func TestPackageFootprintsKeepTakeoversAndPairShippingOnePlugin(t *testing.T) {
	dll, err := fsx.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	var pkgs []profile.PackageRef
	var mods []framework.Mod
	for _, name := range []string{"A-One", "B-Two"} {
		dir := filepath.Join(t.TempDir(), "plugins")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Plugin.dll"), dll, 0o600); err != nil {
			t.Fatal(err)
		}
		pkgs = append(pkgs, profile.PackageRef{Key: name, Enabled: true, Dir: filepath.Dir(dir)})
		mods = append(mods, framework.Mod{Key: name, Enabled: true, SourceKind: "thunderstore", Folder: dir, UniqueID: name, Name: name, Format: mod.FormatThunderstore})
	}

	fp, related := packageFootprints(pkgs, mods)

	want := map[string]bool{
		"harmony:StardewValley.Farmer::Update (prefix-skip)": true,
		"harmony:StardewValley.Game1::Draw (transpiler)":     true,
	}
	one := mods[0].ModID().Fold()
	if len(fp) != 2 || len(fp[one]) != len(want) {
		t.Fatalf("footprints = %v", fp)
	}
	for member := range want {
		if !fp[one][member] {
			t.Fatalf("footprints = %v", fp)
		}
	}
	if got := sameJob(fp, mods); len(got) != 1 {
		t.Fatalf("unrelated rows = %+v", got)
	}
	if got := sameJob(fp, related); len(got) != 0 {
		t.Fatalf("two packages shipping one plugin GUID are a plugin clash, not a same-job hint: %+v", got)
	}
}
