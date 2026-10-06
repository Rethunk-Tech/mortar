package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
)

func TestProfilePackVerbsCallTheirMethods(t *testing.T) {
	results := map[string]any{
		"pack.import":        packsvc.Result{Profile: "p2", Queued: 16},
		"pack.exportModpack": packsvc.ModpackResult{Path: "/x/pack.zip", Dependencies: []string{"A-B-1.0.0"}, LeftOut: []string{"Nexus Thing"}},
		"profile.backup":     map[string]string{"path": "/x/farm.zip"},
		"profile.restore":    packsvc.RestoreResult{Profile: "p3", Name: "Farm (2)", Queued: 2, Unavailable: []string{"X.B"}},
	}
	r := invoke(t, results, "profile", "import", "019c9378-d2e6-4fec-5a2f-e4b22f8cf1d1", "--game", "lethal-company", "--name", "Regress R2")
	if got := r.calls[0].params; r.code != 0 || got.Value != "019c9378-d2e6-4fec-5a2f-e4b22f8cf1d1" || got.Game != "lethal-company" || got.Name != "Regress R2" {
		t.Fatalf("import: %+v %+v", r, got)
	}
	r = invoke(t, results, "profile", "export", "lethal-company", "Friday", "pack.zip", "--format", "modpack", "--no-configs")
	if got := r.calls[0].params; r.code != 0 || got.All || !filepath.IsAbs(got.Path) || !strings.Contains(r.out, "Nexus Thing") {
		t.Fatalf("export: %+v %+v", r, got)
	}
	if r = invoke(t, results, "profile", "export", "lethal-company", "Friday", "pack.zip"); r.code == 0 {
		t.Fatal("export without --format modpack must be refused")
	}
	r = invoke(t, results, "profile", "backup", "stardew", "Farm", "farm.zip")
	if got := r.calls[0]; r.code != 0 || got.method != "profile.backup" || got.params.Profile != "Farm" || !filepath.IsAbs(got.params.Path) {
		t.Fatalf("backup: %+v", r)
	}
	r = invoke(t, results, "profile", "restore", "farm.zip", "--game", "stardew")
	if got := r.calls[0].params; r.code != 0 || got.Game != "stardew" || !strings.Contains(r.out, "X.B") || !strings.Contains(r.out, "queued 2") {
		t.Fatalf("restore: %+v %+v", r, got)
	}
}

func TestProfileFarmVerbsCallTheirMethods(t *testing.T) {
	results := map[string]any{
		"pack.farmExport": packsvc.FarmList{Game: "stardew", Name: "Ours", Mods: []packsvc.FarmMod{}},
		"pack.farmCheck":  packsvc.FarmCheck{Host: "Ours", Rows: []packsvc.FarmRow{{Name: "Pathoschild.X", Host: "2.0.0", Mine: "1.0.0", State: packsvc.FarmDifferent}}},
		"pack.farmFix":    packsvc.FarmFix{Queued: 2, Manual: []string{"Local"}},
	}
	r := invoke(t, results, "profile", "farm", "export", "stardew", "Farm")
	if r.code != 0 || r.calls[0].method != "pack.farmExport" || !strings.Contains(r.out, `"name": "Ours"`) {
		t.Fatalf("export: %+v", r)
	}
	r = invoke(t, results, "profile", "farm", "check", "stardew", "Farm", `{"game":"stardew"}`)
	if got := r.calls[0].params; r.code != 0 || got.Value != `{"game":"stardew"}` || !strings.Contains(r.out, "different\tPathoschild.X\thost 2.0.0\tyours 1.0.0") {
		t.Fatalf("check: %+v %+v", r, got)
	}
	r = invoke(t, results, "profile", "farm", "fix", "stardew", "Farm", `{"game":"stardew"}`)
	if r.code != 0 || r.calls[0].method != "pack.farmFix" || !strings.Contains(r.out, "queued 2") || !strings.Contains(r.out, "Local") {
		t.Fatalf("fix: %+v", r)
	}
	if r = invoke(t, results, "profile", "farm", "check", "stardew", "Farm"); r.code == 0 {
		t.Fatal("check without a list must be refused")
	}
}

func TestRunsNameWhyAFailedLaunchDidNotStart(t *testing.T) {
	results := map[string]any{"runs": []launchsvc.Run{{Started: "2026-10-06T02:29:19Z", Outcome: "failed", Error: "Steam would not start"}}}
	r := invoke(t, results, "runs", "lethal-company", "Farm")
	if r.code != 0 || !strings.Contains(r.out, "Steam would not start") {
		t.Fatalf("runs: %+v", r)
	}
}
