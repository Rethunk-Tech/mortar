package problems

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestRelationsOfAThunderstorePackage(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "LC")
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "p.zip"), map[string]string{
		"manifest.json": `{"name":"Mod","version_number":"1.0.0","dependencies":["Ns-Lib-1.0.0"]}`, "Mod.dll": "x",
	})
	res, err := profiles.InstallSource("lethal-company", p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(t.TempDir(), nil, profiles, nil)
	if _, err := s.Relations("lethal-company", p.ID, res.Profile.Entries[0].Key, "thunderstore:Ns-Mod"); err != nil {
		t.Fatal(err)
	}
}
