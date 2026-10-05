package smapi

import (
	"context"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// TestSmokeRealInstaller runs the real SMAPI installer against a copy of a real Stardew install.
// It needs the network and MORTAR_SMOKE=1; MORTAR_SMOKE_GAME overrides the install to copy.
func TestSmokeRealInstaller(t *testing.T) {
	if os.Getenv("MORTAR_SMOKE") != "1" {
		t.Skip("set MORTAR_SMOKE=1 to run the real installer")
	}
	src := os.Getenv("MORTAR_SMOKE_GAME")
	if src == "" {
		home, _ := os.UserHomeDir()
		src = filepath.Join(home, ".local/share/Steam/steamapps/common/Stardew Valley")
	}
	// The real install is never a target; the installer only ever sees this copy.
	copyDir := filepath.Join("/var/tmp/mortar-smoke", "Stardew Valley")
	if err := os.RemoveAll(copyDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(copyDir)) })
	if err := os.MkdirAll(filepath.Dir(copyDir), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := datadir.CopyTree(src, copyDir); err != nil {
		t.Fatalf("copy install: %v", err)
	}
	g := Loader{CacheDir: t.TempDir(), LogDir: t.TempDir()}
	// A source that already has SMAPI exercises the update path, which is the same installer run.
	ctx := context.Background()
	before, _ := g.Status(loader.Target{InstallDir: copyDir})
	t.Logf("copy before install: %+v", before)
	version, err := g.Latest(ctx, components.GameInfo{})
	if err != nil {
		t.Fatal(err)
	}
	zip := filepath.Join(t.TempDir(), "installer.zip")
	if err := g.Fetch(ctx, components.GameInfo{}, version, zip); err != nil {
		t.Fatal(err)
	}
	var steps []loader.Step
	var mods []string
	bundled := func(_, dir string) error {
		ents, err := os.ReadDir(dir)
		for _, e := range ents {
			mods = append(mods, e.Name())
		}
		return err
	}
	steps = append(steps, loader.StepDownloaded)
	if _, err := g.Install(ctx, loader.Target{InstallDir: copyDir, Bundled: bundled}, loader.Package{ID: ID, Version: version, Archive: zip}, func(s loader.Step) { steps = append(steps, s) }); err != nil {
		t.Fatal(err)
	}
	st, _ := g.Status(loader.Target{InstallDir: copyDir})
	t.Logf("installed SMAPI %s; steps %v; bundled %v; status %+v", version, steps, mods, st)
	if !st.Installed || st.Broken || len(steps) != 4 || len(mods) != 2 {
		t.Fatal("real install did not produce a working SMAPI")
	}
}
