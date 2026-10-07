//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// Two packages ship the same plugin (GUID com.fixture.plugin, version 1.2.3), as the matrix's DupA and DupB do, and a
// third an older copy of it; BepInEx loads one.
func TestOnlyTheCopyBepInExLoadedReadsLoaded(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ps, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	const lc = "lethal-company"
	p, err := ps.Create(lc, "A")
	if err != nil {
		t.Fatal(err)
	}
	dll, err := os.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"DupA", "DupB"} {
		zip := tsPackage(t, name, map[string]string{"plugins/Mod.dll": string(dll)})
		if _, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-" + name, Version: "1.0.0"}); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(t.TempDir(), set, ps)
	pkgs := svc.livePackages(lc, p.ID)()
	a, b := mod.ID("thunderstore:Ns-DupA"), mod.ID("thunderstore:Ns-DupB")
	loaded := func(plugins ...loader.LivePlugin) map[mod.ID]bool {
		out := map[mod.ID]bool{}
		for _, m := range liveMods(pkgs, plugins) {
			out[m.ID] = m.Loaded
		}
		return out
	}
	if got := loaded(loader.LivePlugin{GUID: "com.fixture.plugin", Version: "1.2.3", Location: "Ns-DupB/Mod.dll"}); len(got) != 2 || got[a] || !got[b] {
		t.Fatalf("the bridge named DupB's file: %v (files %v)", got, pkgs.files)
	}
	if got := loaded(loader.LivePlugin{GUID: "com.fixture.plugin", Version: "1.2.3"}); !got[a] || !got[b] {
		t.Fatalf("no location, equal versions: %v", got)
	}
	if got := loaded(loader.LivePlugin{GUID: "com.fixture.plugin", Version: "2.0.0"}); got[a] || got[b] {
		t.Fatalf("another copy's version loaded: %v", got)
	}
}

func TestPluginVersionsCompareAsSystemVersionWritesThem(t *testing.T) {
	for _, c := range []struct {
		declared, loaded string
		same             bool
	}{{"1.02.0", "1.2.0", true}, {"1.0.0", "1.0.0", true}, {"1.0", "1.0.0", false}, {"1.0.1", "1.0.0", false}, {"x", "x", false}} {
		if sameVersion(c.declared, c.loaded) != c.same {
			t.Errorf("sameVersion(%q, %q) != %v", c.declared, c.loaded, c.same)
		}
	}
}
