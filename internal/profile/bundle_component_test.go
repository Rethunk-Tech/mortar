package profile

import (
	"strings"
	"testing"
)

// Two Mortar bundles share source kind mortar, so the profile tells them apart by the component: replacing the
// overlay leaves the companion alone, and replacing the companion leaves the overlay alone.
func TestABundleReplacesOnlyItsOwnComponent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "bridge-0.6.0-aaaaaaaaaaaa", map[string]string{"MortarSmapiBridge/manifest.json": manifestJSON("Rethunk.MortarSmapiBridge")})
	e.item(t, "bridge-0.6.1-aaaaaaaaaaaa", map[string]string{"MortarSmapiBridge/manifest.json": manifestJSON("Rethunk.MortarSmapiBridge")})
	e.item(t, "overlay-0.1.0-bbbbbbbbbbbb", map[string]string{"StreamOverlay/manifest.json": manifestJSON("tech.rethunk.mortar.StreamOverlay")})
	p := mustCreate(t, e, "P")
	bridge := Source{Kind: SourceMortar, Name: "Mortar", Component: BundleBridge}
	overlay := Source{Kind: SourceMortar, Name: "Mortar", Component: BundleOverlay}
	for _, b := range []Bundle{{Key: "bridge-0.6.0-aaaaaaaaaaaa", Source: bridge}, {Key: "overlay-0.1.0-bbbbbbbbbbbb", Source: overlay}} {
		if err := e.ApplyBundled("stardew", b); err != nil {
			t.Fatal(err)
		}
	}
	got, err := e.Get("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if keys := bundleEntryKeys(got, BundleBridge); len(keys) != 1 || keys[0] != "bridge-0.6.0-aaaaaaaaaaaa" {
		t.Fatalf("bridge entries = %v", keys)
	}
	if keys := bundleEntryKeys(got, BundleOverlay); len(keys) != 1 || keys[0] != "overlay-0.1.0-bbbbbbbbbbbb" {
		t.Fatalf("overlay entries = %v", keys)
	}

	// A newer companion swaps only the bridge.
	if err := e.ApplyBundled("stardew", Bundle{Key: "bridge-0.6.1-aaaaaaaaaaaa", Source: bridge}); err != nil {
		t.Fatal(err)
	}
	got, _ = e.Get("stardew", p.ID)
	if keys := bundleEntryKeys(got, BundleBridge); len(keys) != 1 || keys[0] != "bridge-0.6.1-aaaaaaaaaaaa" {
		t.Fatalf("bridge entries after the swap = %v", keys)
	}
	if keys := bundleEntryKeys(got, BundleOverlay); len(keys) != 1 {
		t.Fatalf("the bridge swap took the overlay: %v", keys)
	}

	// Switching the overlay off removes it and nothing else.
	if err := e.RemoveBundled("stardew", Bundle{Source: overlay}); err != nil {
		t.Fatal(err)
	}
	got, _ = e.Get("stardew", p.ID)
	if keys := bundleEntryKeys(got, BundleOverlay); len(keys) != 0 {
		t.Fatalf("overlay entries after removal = %v", keys)
	}
	if keys := bundleEntryKeys(got, BundleBridge); len(keys) != 1 {
		t.Fatalf("the overlay removal took the bridge: %v", keys)
	}
}

// An entry a build before the split wrote names no component; it is the bridge, not an orphan beside it.
func TestAMortarEntryWithoutAComponentReadsAsTheBridge(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "bridge-0.5.4-aaaaaaaaaaaa", map[string]string{"MortarSmapiBridge/manifest.json": manifestJSON("Rethunk.MortarSmapiBridge")})
	p := mustCreate(t, e, "P")
	old := Source{Kind: SourceMortar, Name: "Mortar"}
	if err := e.ApplyBundled("stardew", Bundle{Key: "bridge-0.5.4-aaaaaaaaaaaa", Source: old}); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.Get("stardew", p.ID); len(bundleEntryKeys(got, BundleBridge)) != 1 {
		t.Fatalf("entries = %+v", got.Entries)
	}
	overlay := Source{Kind: SourceMortar, Name: "Mortar", Component: BundleOverlay}
	if err := e.RemoveBundled("stardew", Bundle{Source: overlay}); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.Get("stardew", p.ID); len(bundleEntryKeys(got, BundleBridge)) != 1 {
		t.Fatal("removing the overlay took a bridge entry that names no component")
	}
}

func bundleEntryKeys(p Profile, component string) []string {
	var out []string
	for _, e := range p.Entries {
		if e.Source.Kind == SourceMortar && e.Source.BundleComponent() == component {
			out = append(out, e.Key)
		}
	}
	return out
}

// The hidden pair stays out of what the player manages, which is what makes the second one invisible.
func TestTheOverlayBundleIsHiddenLikeTheBridge(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "overlay-0.1.0-bbbbbbbbbbbb", map[string]string{"StreamOverlay/manifest.json": manifestJSON("tech.rethunk.mortar.StreamOverlay")})
	p := mustCreate(t, e, "P")
	if err := e.ApplyBundled("stardew", Bundle{Key: "overlay-0.1.0-bbbbbbbbbbbb", Source: Source{Kind: SourceMortar, Name: "Mortar", Component: BundleOverlay}}); err != nil {
		t.Fatal(err)
	}
	mods, err := e.UserMods("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 0 {
		t.Fatalf("user mods = %+v", mods)
	}
	all, err := e.Mods("stardew", p.ID)
	if err != nil || len(all) != 1 || !strings.Contains(string(all[0].ID), "StreamOverlay") {
		t.Fatalf("mods = %+v, %v", all, err)
	}
	if _, err := e.RemoveEntry("stardew", p.ID, "overlay-0.1.0-bbbbbbbbbbbb"); err == nil {
		t.Fatal("a bundled overlay entry was removable")
	}
}
