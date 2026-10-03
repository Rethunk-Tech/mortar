package cli

import "testing"

func TestModsPresetSendsActionAndName(t *testing.T) {
	results := map[string]any{"mods.preset": []string{"farm"}}
	r := invoke(t, results, "mods", "preset", "stardew", "Farm", "A.Mod", "list")
	if r.code != 0 || r.calls[0].method != "mods.preset" || r.calls[0].params.Sub != "list" {
		t.Fatalf("list: %+v", r)
	}
	r = invoke(t, results, "mods", "preset", "stardew", "Farm", "A.Mod", "save", "farm")
	if r.code != 0 || r.calls[0].params.Sub != "save" || r.calls[0].params.Name != "farm" {
		t.Fatalf("save: %+v", r)
	}
	r = invoke(t, results, "mods", "preset", "stardew", "Farm", "A.Mod", "apply")
	if r.code != 2 {
		t.Fatalf("apply without name: %+v", r)
	}
}
