//go:build !windows

package launchsvc

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// bridgeFolder must find the companion entry, not the overlay bundle beside it: sending a console command into the
// overlay's folder reaches no bridge.
func TestBridgeFolderSkipsTheOverlayBundle(t *testing.T) {
	svc, p := startEnv(t)
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	for key, folder := range map[string]string{
		"overlay-0.1.0-bbbbbbbbbbbb": "StreamOverlay",
		"bridge-0.6.0-aaaaaaaaaaaa":  "MortarSmapiBridge",
	} {
		src := t.TempDir()
		testfs.WriteFile(t, src, filepath.Join(folder, "manifest.json"), `{"Name":"Mortar","Version":"1","UniqueID":"Rethunk.MortarSmapiBridge","EntryDll":"x.dll"}`)
		if err := items.AddDir(t.Context(), "stardew", key, src); err != nil {
			t.Fatal(err)
		}
	}
	overlay := profile.Source{Kind: profile.SourceMortar, Name: "Mortar", Component: profile.BundleOverlay}
	bridge := profile.Source{Kind: profile.SourceMortar, Name: "Mortar", Component: profile.BundleBridge}
	for _, b := range []profile.Bundle{{Key: "overlay-0.1.0-bbbbbbbbbbbb", Source: overlay}, {Key: "bridge-0.6.0-aaaaaaaaaaaa", Source: bridge}} {
		if err := svc.profiles.ApplyBundled("stardew", b); err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.bridgeFolder(game.Find("stardew"), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(mods, "bridge-0.6.0-aaaaaaaaaaaa", "MortarSmapiBridge"); got != want {
		t.Fatalf("bridgeFolder = %q, want %q", got, want)
	}
}
