//go:build !windows

package smapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/go-safefs"
)

// Vortex deploys SMAPI into the game folder as links into its staging folder; Mortar's install must not write
// through them.
func TestSharedSMAPIIsDetectedAndUnsharedBeforeInstall(t *testing.T) {
	staging, game := t.TempDir(), t.TempDir()
	write := func(path, body string) {
		testfs.WriteAt(t, path, []byte(body))
	}
	write(filepath.Join(staging, "StardewModdingAPI.dll"), "dll")
	write(filepath.Join(staging, "StardewModdingAPI.deps.json"), "deps")
	write(filepath.Join(staging, "smapi-internal", "config.json"), "cfg")
	if err := os.Symlink(filepath.Join(staging, "StardewModdingAPI.dll"), filepath.Join(game, "StardewModdingAPI.dll")); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(staging, "StardewModdingAPI.deps.json"), filepath.Join(game, "StardewModdingAPI.deps.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(staging, "smapi-internal"), filepath.Join(game, "smapi-internal")); err != nil {
		t.Fatal(err)
	}
	if shared, from := sharedSMAPI(game); !shared || !safefs.SamePath(from, staging) {
		t.Fatalf("sharedSMAPI = %v, %q", shared, from)
	}
	if err := unshareSMAPI(game); err != nil {
		t.Fatal(err)
	}
	if shared, _ := sharedSMAPI(game); shared {
		t.Fatal("SMAPI is still shared after unsharing")
	}
	for _, rel := range []string{"StardewModdingAPI.dll", "StardewModdingAPI.deps.json", "smapi-internal/config.json"} {
		write(filepath.Join(game, rel), "mortar")
	}
	for rel, want := range map[string]string{"StardewModdingAPI.dll": "dll", "StardewModdingAPI.deps.json": "deps", "smapi-internal/config.json": "cfg"} {
		if b, _ := safefs.ReadFile(filepath.Join(staging, rel)); string(b) != want {
			t.Fatalf("staging %s changed to %q", rel, b)
		}
	}
}
