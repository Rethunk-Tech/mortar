package problems

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestRelationsOfAThunderstorePackage(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	install := func(name, manifest string) string {
		zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{"manifest.json": manifest, name + ".dll": "x"})
		res, err := profiles.InstallSource("lethal-company", p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-" + name, Version: "1.0.0"})
		if err != nil {
			t.Fatal(err)
		}
		return res.Profile.Entries[len(res.Profile.Entries)-1].Key
	}
	modKey := install("Mod", `{"name":"Mod","version_number":"1.0.0","dependencies":["BepInEx-BepInExPack-5.4.2100","Ns-Lib-1.0.0"]}`)
	libKey := install("Lib", `{"name":"Lib","version_number":"1.0.0","dependencies":[]}`)
	s := NewService(t.TempDir(), nil, profiles, nil)

	r, err := s.Relations("lethal-company", p.ID, modKey, "thunderstore:Ns-Mod")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Needs) != 1 || r.Needs[0].ID != mod.ID("thunderstore:Ns-Lib") || r.Needs[0].Name != "Lib" || r.Needs[0].State != "ok" {
		t.Fatalf("needs = %+v, want only Ns-Lib, met", r.Needs)
	}
	r, err = s.Relations("lethal-company", p.ID, libKey, "thunderstore:Ns-Lib")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.NeededBy) != 1 || r.NeededBy[0].ID != mod.ID("thunderstore:Ns-Mod") {
		t.Fatalf("neededBy = %+v, want Ns-Mod", r.NeededBy)
	}
}
