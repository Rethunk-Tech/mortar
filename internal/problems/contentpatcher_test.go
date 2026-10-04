package problems

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/jsonc"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

func testdataPack(t *testing.T, name string) Installed {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	folder := filepath.Join(filepath.Dir(file), "testdata", name)
	raw, err := fsx.ReadFile(filepath.Join(folder, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	id := name
	var doc map[string]any
	if json.Unmarshal(jsonc.Clean(raw), &doc) == nil {
		if u, ok := doc["UniqueID"].(string); ok {
			id = u
		}
	}
	m := fromDisk(Installed{Key: name, Enabled: true, Folder: folder})
	m.Name, m.UniqueID = id, id
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
		got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{testdataPack(t, "aware_a"), editB})
		if len(got.AssetConflicts) != 0 {
			t.Fatalf("got %+v", got.AssetConflicts)
		}
		got = Check(context.Background(), fakeMeta{}, Environment{}, []Installed{testdataPack(t, "aware_a"), editA})
		if len(got.AssetConflicts) != 1 {
			t.Fatalf("unaware pair not reported: %+v", got.AssetConflicts)
		}
	})
	t.Run("packs from one entry or a declared dependency are intended", func(t *testing.T) {
		a, b := editA, editB
		b.Key = a.Key
		if got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{a, b}); len(got.AssetConflicts) != 0 {
			t.Fatalf("same entry reported: %+v", got.AssetConflicts)
		}
		b = editB
		b.Dependencies = []manifest.Dependency{{UniqueID: a.UniqueID}}
		if got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{a, b}); len(got.AssetConflicts) != 0 {
			t.Fatalf("dependency reported: %+v", got.AssetConflicts)
		}
	})
	t.Run("HasMod conditions gate patches", func(t *testing.T) {
		gated := testdataPack(t, "gated_a")
		got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{gated, loadB, editA})
		if len(got.AssetConflicts) != 0 {
			t.Fatalf("gated patches reported: %+v", got.AssetConflicts)
		}
		got = Check(context.Background(), fakeMeta{}, Environment{}, []Installed{gated, loadA})
		if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Target != "portraits/farmer" {
			t.Fatalf("HasMod false with the mod absent should apply: %+v", got.AssetConflicts)
		}
	})
	t.Run("Load/Load conflict", func(t *testing.T) {
		got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{loadA, loadB})
		if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Kind != "load" || got.AssetConflicts[0].Target != "portraits/farmer" {
			t.Fatalf("got %+v", got.AssetConflicts)
		}
	})
	t.Run("EditImage overlap", func(t *testing.T) {
		got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{editA, editB})
		if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Kind != "edit" || got.AssetConflicts[0].Target != "tilesheets/crops" {
			t.Fatalf("got %+v", got.AssetConflicts)
		}
	})
	t.Run("EditData not reported", func(t *testing.T) {
		got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{editA, editB})
		for _, c := range got.AssetConflicts {
			if c.Target == "data/npcdispositions" {
				t.Fatalf("EditData reported: %+v", c)
			}
		}
	})
	t.Run("tokenized target skipped", func(t *testing.T) {
		if skips := readContentPack(tokenA).skips; skips != 1 {
			t.Fatalf("skips = %d", skips)
		}
		got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{tokenA, tokenB})
		if len(got.AssetConflicts) != 0 {
			t.Fatalf("tokenized conflicted: %+v", got.AssetConflicts)
		}
	})
	t.Run("Include followed", func(t *testing.T) {
		got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{inc, includePeer})
		if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Target != "maps/springobjects" {
			t.Fatalf("got %+v", got.AssetConflicts)
		}
	})
}

func TestItemConflictInfo(t *testing.T) {
	hits := []packHit{
		{id: "A", edits: []cpPatch{{shapes: []cpShape{{kind: 'r', x: 208, y: 192, w: 16, h: 16}}}}},
		{id: "B", edits: []cpPatch{{shapes: []cpShape{{kind: 'r', x: 208, y: 192, w: 16, h: 16}}}}},
		{id: "C", edits: []cpPatch{{shapes: []cpShape{{kind: 'r', x: 208, y: 192, w: 16, h: 16}}}}},
	}
	got := conflictOf("edit", "Maps/springobjects", hits)
	if got.Info != "cell (208, 192)" {
		t.Fatalf("item info = %q", got.Info)
	}
}

func TestContentPatcherJSONNoise(t *testing.T) {
	dir := t.TempDir()
	folder := filepath.Join(dir, "Pack.Noise")
	if err := os.MkdirAll(filepath.Join(folder, "patches"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(folder, "manifest.json"), []byte(`{"Name":"N","UniqueID":"Pack.Noise","Version":"1","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(folder, "content.json"), []byte(`{
		// comment
		"Changes": [
			{ "Action": "Include", "FromFile": "patches/extra.json", },
		]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(folder, "patches", "extra.json"), []byte(`{"Changes":[{"Action":"Load","Target":"Maps/springobjects"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	peer := testdataPack(t, "include_b")
	mod := fromDisk(Installed{Key: "noise", Enabled: true, Folder: folder})
	mod.Name, mod.UniqueID = "Pack.Noise", "Pack.Noise"
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{mod, peer})
	if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Target != "maps/springobjects" {
		t.Fatalf("got %+v", got.AssetConflicts)
	}
}

func TestHideDismissedSoftOnly(t *testing.T) {
	in := []AssetConflict{
		{Kind: "load", Target: "a"},
		{Kind: "edit", Target: "b"},
	}
	got, dismissed := hideDismissed(in, []string{dismissToken("edit", "b"), dismissToken("load", "a")})
	if len(got) != 0 || len(dismissed) != 2 {
		t.Fatalf("got %+v dismissed %+v", got, dismissed)
	}
}
