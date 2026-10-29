package runtime

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

func TestLauncherPrefixResolvesSavesInTheLaunchersPrefix(t *testing.T) {
	inst := Install{GOOS: "linux", Platform: "windows", Store: "gog-heroic", Prefix: "/g/Prefixes/X/pfx", Game: "x", Dir: "/g/X"}
	if IDOf(inst) != LauncherPrefix {
		t.Fatalf("runtime %q", IDOf(inst))
	}
	got, err := Resolve(inst, components.PathTemplate{Windows: "{localLow}/Studio/Game"})
	want := filepath.FromSlash("/g/Prefixes/X/pfx/drive_c/users/steamuser/AppData/LocalLow/Studio/Game")
	if err != nil || got != want {
		t.Fatalf("got %q (%v), want %q", got, err, want)
	}
	if c, ok := CompatDataOf(inst); !ok || c != filepath.Dir(inst.Prefix) {
		t.Fatalf("compat %q %v", c, ok)
	}
	inst.Prefix = ""
	if IDOf(inst) != MortarProton {
		t.Fatalf("with no prefix: %q", IDOf(inst))
	}
}
