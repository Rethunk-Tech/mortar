package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/control"
)

func TestModsFilesListsExtras(t *testing.T) {
	results := map[string]any{
		"mods.files": []control.ModExtraFile{
			{Key: "nexus-1-2", Label: "Optional (1.0)"},
		},
	}
	r := invoke(t, results, "mods", "files", "stardew", "Farm", "A.Mod")
	if r.code != 0 || r.calls[0].method != "mods.files" || r.calls[0].params.UniqueIDs[0] != "A.Mod" {
		t.Fatalf("call: %+v", r)
	}
	if !strings.Contains(r.out, "nexus-1-2") || !strings.Contains(r.out, "Optional (1.0)") {
		t.Fatalf("table: %q", r.out)
	}
	r = invoke(t, results, "mods", "files", "stardew", "Farm", "A.Mod", "--json")
	var keys []string
	if err := json.Unmarshal([]byte(r.out), &keys); err != nil || len(keys) != 1 || keys[0] != "nexus-1-2" {
		t.Fatalf("json: %v %q", err, r.out)
	}
}
