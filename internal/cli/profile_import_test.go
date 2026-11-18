package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/migrate"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
)

func TestProfilePackVerbsCallTheirMethods(t *testing.T) {
	results := map[string]any{
		"pack.import":        sharesvc.Result{Profile: profile.Profile{ID: "p2"}, Queued: 16},
		"pack.exportModpack": packsvc.ModpackResult{Path: "/x/pack.zip", Dependencies: []string{"A-B-1.0.0"}, LeftOut: []string{"Nexus Thing"}},
		"pack.exportR2z":     map[string]string{"path": "/x/friends.r2z"},
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
		t.Fatal("export without --format modpack or r2z must be refused")
	}
	r = invoke(t, results, "profile", "export", "lethal-company", "Friday", "friends.r2z", "--format", "r2z")
	if got := r.calls[0]; r.code != 0 || got.method != "pack.exportR2z" || !filepath.IsAbs(got.params.Path) || !strings.Contains(r.out, "friends.r2z") {
		t.Fatalf("export r2z: %+v %+v", r, got)
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

func TestProfileImportPreviewAndPackFile(t *testing.T) {
	pack := filepath.Join(t.TempDir(), "friends.r2z")
	if err := os.WriteFile(pack, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	results := map[string]any{
		"pack.import": sharesvc.Preview{
			Name: "Friday", Settings: 2,
			Mods: []sharesvc.Mod{
				{Package: "A-Core", Version: "1.0", State: sharesvc.StateUnavailable, Detail: "gone from the index"},
				{Name: "Old", Version: "0.1", State: sharesvc.StateUnavailable, Reason: "removed"},
				{Name: "Off", Version: "2.0", Disabled: []mod.ID{"smapi:Off"}},
				{Name: "Ok", Version: "3.0"},
			},
		},
	}
	r := invoke(t, results, "profile", "import", "--preview", "r2mm://code", "--game", "lethal-company")
	if r.code != 0 || !strings.Contains(r.out, "Friday (lethal-company): 4 mods, 2 config files") ||
		!strings.Contains(r.out, "A-Core 1.0  (unavailable: gone from the index)") ||
		!strings.Contains(r.out, "Old 0.1  (unavailable: removed)") ||
		!strings.Contains(r.out, "Off 2.0  (disabled)") ||
		!strings.Contains(r.out, "Ok 3.0") {
		t.Fatalf("preview: %+v", r)
	}
	r = invoke(t, map[string]any{"pack.import": sharesvc.Result{Profile: profile.Profile{ID: "p2"}, Queued: 1}},
		"profile", "import", pack, "--game", "lethal-company")
	if got := r.calls[0].params; r.code != 0 || got.Value != "" || got.Path != pack {
		t.Fatalf("file: %+v %+v", r, got)
	}
	if r = invoke(t, nil, "profile", "import"); r.code == 0 {
		t.Fatal("import without a code or file must be refused")
	}
}

func TestProfileExportNamesDisabledPackages(t *testing.T) {
	results := map[string]any{
		"pack.exportModpack": packsvc.ModpackResult{Path: "/x/pack.zip", Dependencies: []string{"A-B-1.0.0"}, Disabled: []string{"Off Mod"}, Configs: 3},
	}
	r := invoke(t, results, "profile", "export", "lethal-company", "Friday", "pack.zip", "--format", "modpack")
	if r.code != 0 || !strings.Contains(r.out, "3 config files") || !strings.Contains(r.out, "Left out (switched off): Off Mod") {
		t.Fatalf("export: %+v", r)
	}
	if r = invoke(t, nil, "profile", "export", "lethal-company"); r.code == 0 {
		t.Fatal("export without a profile and file must be refused")
	}
}

func TestProfileImportFromAnotherManager(t *testing.T) {
	results := map[string]any{
		"external.sources": []migrate.SourceInfo{{
			Profiles: []migrate.ProfileInfo{{ID: "vortex-farm", Name: "Farm", Mods: 4}},
		}},
		"external.import": migrate.ProfilePreview{
			Name: "Farm", Source: "vortex",
			Mods: []migrate.ModPreview{
				{Name: "On", Version: "1.0", Enabled: true},
				{Name: "Off", Version: "2.0"},
			},
		},
	}
	r := invoke(t, results, "profile", "import", "--from", "vortex", "--game", "stardew")
	if r.code != 0 || r.calls[0].method != "external.sources" || !strings.Contains(r.out, "vortex-farm\tFarm\t4 mods") {
		t.Fatalf("list: %+v", r)
	}
	r = invoke(t, results, "profile", "import", "--from", "vortex", "--preview", "vortex-farm", "--game", "stardew")
	if r.code != 0 || r.calls[0].method != "external.import" || r.calls[0].params.ID != "vortex-farm" ||
		!strings.Contains(r.out, "Farm (vortex): 2 mods") || !strings.Contains(r.out, "Off 2.0  (disabled)") {
		t.Fatalf("preview: %+v", r)
	}
	results["external.import"] = sharesvc.Result{Profile: profile.Profile{Name: "Farm"}, Queued: 3}
	r = invoke(t, results, "profile", "import", "--from", "vortex", "vortex-farm", "--game", "stardew")
	if r.code != 0 || r.calls[0].method != "external.import" || !strings.Contains(r.out, "Farm\tqueued 3 downloads") {
		t.Fatalf("import: %+v", r)
	}
	if r = invoke(t, map[string]any{"games": []any{}}, "profile", "import", "--from", "vortex"); r.code == 0 {
		t.Fatal("import --from without a game must be refused when none is installed")
	}
}

func TestRunsNameWhyAFailedLaunchDidNotStart(t *testing.T) {
	results := map[string]any{"runs": []launchsvc.Run{{Started: "2026-10-06T02:29:19Z", Outcome: "failed", Error: "Steam would not start"}}}
	r := invoke(t, results, "runs", "lethal-company", "Farm")
	if r.code != 0 || !strings.Contains(r.out, "Steam would not start") {
		t.Fatalf("runs: %+v", r)
	}
}
