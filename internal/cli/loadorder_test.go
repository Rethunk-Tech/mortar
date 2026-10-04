package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/loadorder"
)

func TestProfileLoadOrder(t *testing.T) {
	results := map[string]any{
		"profile.loadOrder": []loadorder.Row{
			{Position: 1, UniqueID: "Z.Lib", Name: "Lib", Dependents: []string{"Z.Addon"}},
			{Position: 2, UniqueID: "Z.Addon", Name: "Addon", Required: []string{"Z.Lib", "Missing.Mod"}, MissingRequired: []string{"Missing.Mod"}, Cycle: true},
		},
	}
	r := invoke(t, results, "profile", "load-order", "stardew", "Farm")
	if r.code != 0 || r.calls[0].method != "profile.loadOrder" || r.calls[0].params.Profile != "Farm" {
		t.Fatalf("call: %+v", r)
	}
	if !strings.Contains(r.out, "MOD ID") || !strings.Contains(r.out, "dependency cycle") || !strings.Contains(r.out, "missing Missing.Mod") {
		t.Fatalf("table: %q", r.out)
	}
	if !strings.Contains(r.out, "Addon") || !strings.Contains(r.out, "Lib") {
		t.Fatalf("names: %q", r.out)
	}
	if strings.Contains(r.out, "UNIQUEID") || strings.Contains(r.out, "\tcycle") {
		t.Fatalf("internal labels: %q", r.out)
	}
	r = invoke(t, results, "profile", "load-order", "stardew", "Farm", "--json")
	var rows []loadorder.Row
	if err := json.Unmarshal([]byte(r.out), &rows); err != nil || len(rows) != 2 || !rows[1].Cycle {
		t.Fatalf("--json: %v %q", err, r.out)
	}
}
