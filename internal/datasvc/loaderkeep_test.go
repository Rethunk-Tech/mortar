package datasvc

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestAPerProfileLoaderKeepsItsStoreItem(t *testing.T) {
	testfs.DataHome(t)
	items, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "Main")
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	const key = "bepinex5-5.4.2305"
	src := t.TempDir()
	testfs.WriteFile(t, src, "installer.zip", "pack")
	if err := items.AddDir("lethal-company", key, src); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "profiles", "lethal-company", p.ID)
	testfs.WriteFile(t, dir, filepath.Join("BepInEx", "core", "BepInEx.Preloader.dll"), "dll")
	testfs.WriteFile(t, dir, ".mortar-bepinex.json", `{"version":"5.4.2305","doorstop":4}`)

	svc := NewService(items, profiles, nil)
	report, err := svc.Report()
	if err != nil {
		t.Fatal(err)
	}
	if unused := report["lethal-company"].Unused; len(unused) != 0 {
		t.Fatalf("the profile's loader is reported unused: %+v", unused)
	}
	preview, err := svc.CleanupPreview()
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range preview.Items {
		if it.Kind == "store" {
			t.Fatalf("cleanup offers the profile's loader: %+v", it)
		}
	}
	if err := svc.RemoveItems("lethal-company", []string{key}); !errors.Is(err, errInUse) {
		t.Fatalf("Remove must refuse the profile's loader, got %v", err)
	}
}
