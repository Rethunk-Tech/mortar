package game

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
)

func TestANativeLinuxBuildRunsNativelyAndAWindowsBuildUnderProton(t *testing.T) {
	if goruntime.GOOS != "linux" {
		t.Skip("native Linux builds are found on Linux only")
	}
	vh, ok := components.Game("valheim")
	if !ok {
		t.Fatal("no Valheim in the catalog")
	}
	for marker, want := range map[string][2]string{"valheim.x86_64": {"linux", runtime.Native}, "valheim.exe": {"windows", runtime.Proton}} {
		dir := filepath.Join(t.TempDir(), "steamapps", "common", "Valheim")
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, marker), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		in := newInstall(vh, "steam", dir, "", "discovered")
		if in.Platform != want[0] || in.Runtime != want[1] {
			t.Errorf("%s: platform %q runtime %q, want %v", marker, in.Platform, in.Runtime, want)
		}
		if err := catalogOnly("valheim").ValidInstall(dir); err != nil {
			t.Errorf("%s: %v", marker, err)
		}
	}
}
