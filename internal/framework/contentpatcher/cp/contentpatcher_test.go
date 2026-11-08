package cp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-lenientjson"
	"github.com/Rethunk-Tech/go-safefs"
)

func testdataPack(t *testing.T, name string) Mod {
	t.Helper()
	folder := testdataFolder(t, name)
	m := Mod{Key: name, Name: name, UniqueID: SMAPI(name), Folder: folder, Enabled: true}
	if raw, err := safefs.ReadFile(filepath.Join(folder, "manifest.json")); err == nil {
		var doc map[string]any
		if json.Unmarshal(lenientjson.Clean(raw), &doc) == nil {
			if id, ok := doc["UniqueID"].(string); ok {
				m.UniqueID, m.Name = SMAPI(id), id
			}
		}
	}
	m.ContentPackFor = SMAPI("Pathoschild.ContentPatcher")
	return m
}

func TestAssetConflicts(t *testing.T) {
	loadA := testdataPack(t, "load_a")
	loadB := testdataPack(t, "load_b")
	editA := testdataPack(t, "edit_a")
	editB := testdataPack(t, "edit_b")
	tokenA := testdataPack(t, "token_a")
	tokenB := testdataPack(t, "token_b")
	inc := testdataPack(t, "include_a")
	includePeer := testdataPack(t, "include_b")

	t.Run("edits by packs that name each other are intended", func(t *testing.T) {
		got := check([]Mod{testdataPack(t, "aware_a"), editB})
		if len(got.Conflicts) != 0 {
			t.Fatalf("got %+v", got.Conflicts)
		}
		got = check([]Mod{testdataPack(t, "aware_a"), editA})
		if len(got.Conflicts) != 1 {
			t.Fatalf("unaware pair not reported: %+v", got.Conflicts)
		}
	})
	t.Run("packs from one entry or a declared dependency are intended", func(t *testing.T) {
		a, b := editA, editB
		b.Key = a.Key
		if got := check([]Mod{a, b}); len(got.Conflicts) != 0 {
			t.Fatalf("same entry reported: %+v", got.Conflicts)
		}
		b = editB
		b.Dependencies = []Dependency{{UniqueID: a.UniqueID}}
		if got := check([]Mod{a, b}); len(got.Conflicts) != 0 {
			t.Fatalf("dependency reported: %+v", got.Conflicts)
		}
	})
	t.Run("HasMod conditions gate patches", func(t *testing.T) {
		gated := testdataPack(t, "gated_a")
		got := check([]Mod{gated, loadB, editA})
		if len(got.Conflicts) != 0 {
			t.Fatalf("gated patches reported: %+v", got.Conflicts)
		}
		got = check([]Mod{gated, loadA})
		if len(got.Conflicts) != 1 || got.Conflicts[0].Target != "portraits/farmer" {
			t.Fatalf("HasMod false with the mod absent should apply: %+v", got.Conflicts)
		}
	})
	t.Run("Load/Load conflict", func(t *testing.T) {
		got := check([]Mod{loadA, loadB})
		if len(got.Conflicts) != 1 || got.Conflicts[0].Kind != "load" || got.Conflicts[0].Target != "portraits/farmer" {
			t.Fatalf("got %+v", got.Conflicts)
		}
	})
	t.Run("EditImage overlap", func(t *testing.T) {
		got := check([]Mod{editA, editB})
		if len(got.Conflicts) != 1 || got.Conflicts[0].Kind != "edit" || got.Conflicts[0].Target != "tilesheets/crops" {
			t.Fatalf("got %+v", got.Conflicts)
		}
	})
	t.Run("EditData not reported", func(t *testing.T) {
		got := check([]Mod{editA, editB})
		for _, c := range got.Conflicts {
			if c.Target == "data/npcdispositions" {
				t.Fatalf("EditData reported: %+v", c)
			}
		}
	})
	t.Run("tokenized target skipped", func(t *testing.T) {
		if skips := ReadPack(tokenA, false).Skips; skips != 1 {
			t.Fatalf("skips = %d", skips)
		}
		got := check([]Mod{tokenA, tokenB})
		if len(got.Conflicts) != 0 {
			t.Fatalf("tokenized conflicted: %+v", got.Conflicts)
		}
	})
	t.Run("Include followed", func(t *testing.T) {
		got := check([]Mod{inc, includePeer})
		if len(got.Conflicts) != 1 || got.Conflicts[0].Target != "maps/springobjects" {
			t.Fatalf("got %+v", got.Conflicts)
		}
	})
}

// An item conflict names the cells every pack's clashing patches write, which the caller words as "cell (x, y)".
func TestSharedCellsOfAnItemConflict(t *testing.T) {
	hits := []packHit{
		{ID: "smapi:A", Edits: []Patch{{Shapes: []Shape{{Kind: 'r', X: 208, Y: 192, W: 16, H: 16}}}}},
		{ID: "smapi:B", Edits: []Patch{{Shapes: []Shape{{Kind: 'r', X: 208, Y: 192, W: 16, H: 16}}}}},
		{ID: "smapi:C", Edits: []Patch{{Shapes: []Shape{{Kind: 'r', X: 208, Y: 192, W: 16, H: 16}}}}},
	}
	got := conflictOf("edit", "Maps/springobjects", hits)
	if want := []string{"13,12"}; !slices.Equal(got.Cells, want) {
		t.Fatalf("cells = %q, want %q", got.Cells, want)
	}
}

func TestContentPatcherJSONNoise(t *testing.T) {
	dir := t.TempDir()
	folder := filepath.Join(dir, "Pack.Noise")
	if err := os.MkdirAll(filepath.Join(folder, "patches"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(filepath.Join(folder, "manifest.json"), []byte(`{"Name":"N","UniqueID":"Pack.Noise","Version":"1","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(filepath.Join(folder, "content.json"), []byte(`{
		// comment
		"Changes": [
			{ "Action": "Include", "FromFile": "patches/extra.json", },
		]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(filepath.Join(folder, "patches", "extra.json"), []byte(`{"Changes":[{"Action":"Load","Target":"Maps/springobjects"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	peer := testdataPack(t, "include_b")
	im := packFromDisk(Mod{Key: "noise", Enabled: true, Folder: folder})
	im.Name, im.UniqueID = "Pack.Noise", "Pack.Noise"
	got := check([]Mod{im, peer})
	if len(got.Conflicts) != 1 || got.Conflicts[0].Target != "maps/springobjects" {
		t.Fatalf("got %+v", got.Conflicts)
	}
}
