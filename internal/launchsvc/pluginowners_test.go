//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// A disabled package is not loaded, so a loader log naming its plugin blames nothing, as the Problems tab's own
// attribution already holds.
func TestDisabledPackageOwnsNoLogNames(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	_, ps := testenv.Stores(t)
	const lc = "lethal-company"
	p := testenv.Profile(t, ps, lc, "A")
	dll, err := os.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	zip := tsPackage(t, "Fixture", map[string]string{"plugins/Mod.dll": string(dll)})
	res, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Fixture", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), nil, ps)
	key := res.Profile.Entries[0].Key
	installed, err := ps.Installed(lc, p.ID)
	if err != nil || len(installed) != 1 {
		t.Fatalf("installed = %+v, %v", installed, err)
	}
	if got := svc.pluginOwners(lc, p.ID); len(got) == 0 {
		t.Fatal("an enabled package that declares a plugin owns its names")
	}
	if _, err := ps.SetModEnabled(lc, p.ID, key, installed[0].ModID(), false); err != nil {
		t.Fatal(err)
	}
	if got := svc.pluginOwners(lc, p.ID); len(got) != 0 {
		t.Fatalf("a disabled package owns %v", got)
	}
}
