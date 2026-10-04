package launchsvc

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

func TestReadStartupReportsMergesSamples(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "2026-10-04T010203Z.json")
	if err := datadir.WriteJSON(reportPath, StartupReport{
		ID:   "ignored-by-filename",
		Mods: []StartupMod{{ID: "FixtureMod"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteJSON(filepath.Join(dir, "2026-10-04T010203Z.samples.json"), startupSamples{
		Schema:  1,
		Mods:    map[string]int64{"FixtureMod": 312},
		OtherMs: 88,
	}); err != nil {
		t.Fatal(err)
	}
	reports, err := readStartupReports(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}
	if reports[0].Mods[0].SampleMs != 312 || reports[0].SampledOtherMs != 88 {
		t.Fatalf("merged report = %+v", reports[0])
	}
}
