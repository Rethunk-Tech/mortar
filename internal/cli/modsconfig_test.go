package cli

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestModsConfigListsAndSets(t *testing.T) {
	results := map[string]any{
		"mods.config": []profile.ConfigField{
			{Path: "Season", Value: "Spring", AllowValues: []string{"Spring", "Summer", "Fall"}},
		},
	}
	r := invoke(t, results, "mods", "config", "stardew", "Farm", "A.Pack")
	if r.code != 0 || r.calls[0].method != "mods.config" || len(r.calls[0].params.IDs) != 1 {
		t.Fatalf("list: %+v", r)
	}
	if !strings.Contains(r.out, "Season = Spring") || !strings.Contains(r.out, "Spring, Summer, Fall") {
		t.Fatalf("print: %q", r.out)
	}
	r = invoke(t, results, "mods", "config", "stardew", "Farm", "A.Pack", "nested.foo", "2")
	if r.code != 0 || r.calls[0].params.Key != "nested.foo" || r.calls[0].params.Value != "2" {
		t.Fatalf("set: %+v", r)
	}
}
