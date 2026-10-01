package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func TestInstallArchiveAsksForFomodThenInstallsChoices(t *testing.T) {
	e := newEnv(t)
	p, err := e.Create("stardew", "A")
	if err != nil {
		t.Fatal(err)
	}
	xml, err := fsx.ReadFile(filepath.Join("..", "fomod", "testdata", "choose-one.xml"))
	if err != nil {
		t.Fatal(err)
	}
	z := buildZip(t, "fomod.zip", map[string]string{
		"fomod/ModuleConfig.xml": string(xml),
		"alpha/manifest.json":    manifestJSON("A.Alpha"),
		"beta/manifest.json":     manifestJSON("A.Beta"),
	})
	res, err := e.InstallArchive("stardew", p.ID, z)
	if err != nil {
		t.Fatal(err)
	}
	if res.Fomod == nil || res.Fomod.ModuleName != "Choose One" {
		t.Fatalf("want fomod ask, got %+v", res)
	}
	if len(res.Profile.Entries) != 0 {
		t.Fatalf("asked but already installed: %+v", res.Profile.Entries)
	}
	res, err = e.InstallFomod("stardew", p.ID, res.Fomod.Key, res.Fomod.Source, map[string]map[string][]string{
		"Options": {"Pack": {"Alpha"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Fomod != nil || len(res.Added) != 1 || res.Added[0] != "A.Alpha" {
		t.Fatalf("install: %+v", res)
	}
	got, err := os.ReadFile(filepath.Join(e.mods(p.ID), res.Profile.Entries[0].Key, "manifest.json"))
	if err != nil || string(got) != manifestJSON("A.Alpha") {
		t.Fatalf("layout %q %v", got, err)
	}
	if res.Profile.Entries[0].Fomod["Options"]["Pack"][0] != "Alpha" {
		t.Fatalf("stored choices: %+v", res.Profile.Entries[0].Fomod)
	}
}

func TestFomodReplayMismatchAsksAgain(t *testing.T) {
	e := newEnv(t)
	p, err := e.Create("stardew", "A")
	if err != nil {
		t.Fatal(err)
	}
	xml, err := fsx.ReadFile(filepath.Join("..", "fomod", "testdata", "choose-one.xml"))
	if err != nil {
		t.Fatal(err)
	}
	e.item(t, "local-fomod", map[string]string{
		"fomod/ModuleConfig.xml": string(xml),
		"alpha/manifest.json":    manifestJSON("A.Alpha"),
		"beta/manifest.json":     manifestJSON("A.Beta"),
	})
	res, err := e.InstallFomod("stardew", p.ID, "local-fomod", Source{Kind: KindLocal, Name: "x"}, map[string]map[string][]string{
		"Options": {"Pack": {"Gone"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Fomod == nil {
		t.Fatal("vanished choice should ask again")
	}
}
