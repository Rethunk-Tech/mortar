package modname

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestOf(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		game  string
		files map[string]string
		want  Info
	}{
		{
			"smapi manifest", "stardew",
			map[string]string{"manifest.json": `{"Name":"Alpha","Author":"me","Version":"1.2.3","UniqueID":"Me.Alpha"}`},
			Info{ID: mod.SMAPI("Me.Alpha"), Name: "Alpha", Version: "1.2.3"},
		},
		{"smapi folder without a manifest", "stardew", map[string]string{"readme.txt": "x"}, Info{}},
		{
			"thunderstore namespace", "lethal-company",
			map[string]string{"manifest.json": `{"name":"Pkg","version_number":"2.0.0","namespace":"Ns"}`},
			Info{ID: mod.NewID(mod.FormatThunderstore, "Ns-Pkg"), Name: "Pkg", Version: "2.0.0"},
		},
		{
			"thunderstore without a publisher", "lethal-company",
			map[string]string{"manifest.json": `{"name":"Pkg","version_number":"2.0.0"}`},
			Info{ID: mod.NewID(mod.FormatThunderstore, "Unknown-Pkg"), Name: "Pkg", Version: "2.0.0"},
		},
		{
			"first dll, extension in any case", "lethal-company",
			map[string]string{"plugins/zeta.dll": "x", "plugins/Alpha.DLL": "x", "readme.txt": "x"},
			Info{ID: mod.NewID(mod.FormatBepInEx, "Alpha"), Name: "Alpha"},
		},
		{"package game with nothing in it", "lethal-company", map[string]string{"readme.txt": "x"}, Info{}},
		{
			"loose-file game", "sims4",
			map[string]string{"manifest.json": `{"name":"Pkg","version_number":"2.0.0"}`, "a.dll": "x"},
			Info{},
		},
	} {
		dir := t.TempDir()
		for rel, body := range tc.files {
			testfs.WriteFile(t, dir, rel, body)
		}
		if got := Of(tc.game, dir); got != tc.want {
			t.Errorf("%s: Of = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}
