package contentpatcher

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/packs"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// variantPack mirrors DaisyNiko's Tilesheets ("Off" lets the pack detect the recolour) or, with blank, Fish_Pond,
// whose unrelated "style" token yields a picker word under another mod and must not count.
func variantPack(t *testing.T, current string, blank bool) framework.Mod {
	t.Helper()
	schema := `{"Pick":{"AllowValues":"Off, Vanilla, Earthy, VibrantPastoral","Default":"Off"}}`
	tokens := `[{"Name":"recolour","Value":"Vanilla","When":{"Pick":"Off, Vanilla"}},` +
		`{"Name":"recolour","Value":"Earthy","When":{"HasMod":"Daisy.Earthy","Pick":"Off"}},` +
		`{"Name":"recolour","Value":"VibrantPastoral","When":{"HasMod":"Grape.Vibrant","Pick":"Off"}}]`
	if blank {
		schema = `{"Pick":{"AllowValues":"Vanilla, Earthy, VibrantPastoral","AllowBlank":true,"Default":""}}`
		tokens = `[{"Name":"recolour","Value":"Vanilla"},{"Name":"recolour","Value":"Earthy","When":{"HasMod":"Daisy.Earthy"}},` +
			`{"Name":"recolour","Value":"VibrantPastoral","When":{"HasMod":"Grape.Vibrant"}},` +
			`{"Name":"recolour","Value":"{{Pick}}","When":{"Pick |contains=Vanilla, Earthy, VibrantPastoral":true}},` +
			`{"Name":"style","Value":"Earthy","When":{"HasMod":"Other.Mod"}}]`
	}
	dir := t.TempDir()
	files := map[string]string{
		"manifest.json": `{"Name":"Tiles","UniqueID":"Pack.Tiles","Version":"1.0.0","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`,
		"content.json":  `{"ConfigSchema":` + schema + `,"DynamicTokens":` + tokens + `,"Changes":[]}`,
		"config.json":   `{"Pick":"` + current + `"}`,
	}
	for name, body := range files {
		testfs.WriteFile(t, dir, name, body)
	}
	return packs.FromDisk(framework.Mod{Key: "tiles", Enabled: true, Folder: dir, UniqueID: "Pack.Tiles", Name: "Tiles"})
}

func recolour(id string) framework.Mod {
	return framework.Mod{Key: id, Enabled: true, UniqueID: id, Name: id}
}

func TestVariantSettings(t *testing.T) {
	cases := []struct {
		name    string
		current string
		blank   bool
		mods    []framework.Mod
		want    string // first suggested value, or "-" for no hint
	}{
		{"auto detects on its own", "Off", false, []framework.Mod{recolour("Grape.Vibrant")}, "-"},
		{"manual choice matches the installed recolour", "VibrantPastoral", false, []framework.Mod{recolour("Grape.Vibrant")}, "-"},
		{"manual choice names a missing recolour", "Earthy", false, []framework.Mod{recolour("Grape.Vibrant")}, "Off"},
		{"generic choice while a recolour is installed", "Vanilla", false, []framework.Mod{recolour("Grape.Vibrant")}, "Off"},
		{"generic choice and no recolour", "Vanilla", false, nil, "-"},
		{"manual choice of another installed recolour", "Earthy", false, []framework.Mod{recolour("Daisy.Earthy"), recolour("Grape.Vibrant")}, "-"},
		{"blank is the automatic choice", "", true, []framework.Mod{recolour("Grape.Vibrant")}, "-"},
		{"override names a missing recolour, blank allowed", "Earthy", true, []framework.Mod{recolour("Grape.Vibrant")}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hints := settingsOf(append([]framework.Mod{variantPack(t, c.current, c.blank)}, c.mods...))
			if c.want == "-" {
				if len(hints) != 0 {
					t.Fatalf("hints = %#v", hints)
				}
				return
			}
			if len(hints) != 1 || !hints[0].Variant || len(hints[0].Suggested) == 0 || hints[0].Suggested[0] != c.want {
				t.Fatalf("hints = %#v", hints)
			}
		})
	}
}
