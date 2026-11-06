package launchsvc

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-nettrace"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestChargeUsesInnermostMappedAssembly(t *testing.T) {
	samples := []nettrace.Sample{
		{Timestamp: 0, ThreadID: 7, Frames: []nettrace.Frame{{Assembly: "Game"}, {Assembly: "Spin"}}},
		{Timestamp: 100_000_000, ThreadID: 7, Frames: []nettrace.Frame{{Assembly: "Spin"}}},
		{Timestamp: 250_000_000, ThreadID: 7, Frames: []nettrace.Frame{{Assembly: "Game"}}},
		{Timestamp: 400_000_000, ThreadID: 7, Frames: []nettrace.Frame{{Assembly: "Game"}}},
		{Timestamp: 500_000_000, ThreadID: 8, Frames: []nettrace.Frame{{Assembly: "Spin"}}},
	}
	result := charge(samples, 7, map[string]string{"Spin": "FixtureMod"})
	if result.ThreadSamples != 4 {
		t.Fatalf("thread samples = %d, want 4", result.ThreadSamples)
	}
	if result.Mods["FixtureMod"] != 250 {
		t.Fatalf("mod time = %d, want 250ms", result.Mods["FixtureMod"])
	}
	if result.OtherMs != 150 {
		t.Fatalf("other time = %d, want 150ms", result.OtherMs)
	}
}

func TestModOfReadsPathsAndFullAssemblyNames(t *testing.T) {
	m := map[string]string{"Spin": "FixtureMod"}
	for _, assembly := range []string{"Spin", `C:\Mods\Spin.dll`, "Spin, Version=1.0.0.0, Culture=neutral"} {
		if got := modOf(assembly, m); got != "FixtureMod" {
			t.Fatalf("modOf(%q) = %q", assembly, got)
		}
	}
}

func TestATraceWithNoEventsWritesNoSamples(t *testing.T) {
	file, err := os.Open(filepath.Join("testdata", "no-events.nettrace"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	dir := t.TempDir()
	if _, err := writeStartupSamples(file, dir, "run", t.TempDir()); !errors.Is(err, nettrace.ErrNoEvents) {
		t.Fatalf("err = %v, want ErrNoEvents", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "run.samples.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a samples file was written: %v", err)
	}
}

func TestPruneStartupReportsKeepsTheNewestWithTheirSamples(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []string{"20260101", "20260102", "20260103", "20260104"} {
		testfs.WriteFile(t, dir, id+".json", "{}")
		testfs.WriteFile(t, dir, id+".samples.json", "{}")
	}
	pruneStartupReports(dir, 2)
	left, err := startupReportIDs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 2 || !left["20260104"] || !left["20260103"] {
		t.Fatalf("kept %v, want the two newest", left)
	}
	if _, err := os.Stat(filepath.Join(dir, "20260101.samples.json")); !os.IsNotExist(err) {
		t.Fatalf("an old report's samples file was left behind: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "20260103.samples.json")); err != nil {
		t.Fatalf("a kept report lost its samples file: %v", err)
	}
}
