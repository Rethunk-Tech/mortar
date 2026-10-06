package profile

import (
	"path/filepath"
	"testing"

	_ "github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
)

func TestInstalledLoaderReadsAPerProfileLoaderFromTheProfile(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "Main")
	if err != nil {
		t.Fatal(err)
	}
	if got := e.InstalledLoader("lethal-company", p.ID); got != "" {
		t.Fatalf("no loader yet, got %q", got)
	}
	dir, err := e.ProfileDir("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, filepath.Join("BepInEx", "core", "BepInEx.Preloader.dll"), "dll")
	if got := e.InstalledLoader("lethal-company", p.ID); got != "bepinex5" {
		t.Fatalf("installed loader %q", got)
	}
}
