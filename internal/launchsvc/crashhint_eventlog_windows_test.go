//go:build windows

package launchsvc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// standInWevtutil builds a wevtutil.exe that prints the event log fixture named by MORTAR_TEST_EVENTLOG, whatever it is
// asked, and puts it first on PATH: EventLog reads the Application log only on Windows, through that program.
func standInWevtutil(t *testing.T, fixture string) {
	t.Helper()
	xml, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	testfs.WriteFile(t, src, "go.mod", "module wevtutil\n\ngo 1.21\n")
	testfs.WriteFile(t, src, "main.go", "package main\n\nimport (\n\t\"os\"\n)\n\nfunc main() {\n\tb, err := os.ReadFile(os.Getenv(\"MORTAR_TEST_EVENTLOG\"))\n\tif err != nil {\n\t\tos.Exit(1)\n\t}\n\t_, _ = os.Stdout.Write(b)\n}\n")
	bin := t.TempDir()
	build := exec.Command("go", "build", "-o", filepath.Join(bin, "wevtutil.exe"), ".")
	build.Dir = src
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build the stand-in wevtutil: %v\n%s", err, out)
	}
	t.Setenv("MORTAR_TEST_EVENTLOG", testfs.WriteFile(t, t.TempDir(), "application-log.xml", string(xml)))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// A crash the Application event log recorded is explained when the game kept no dump: its unhandled exception names the
// mod whose code threw, which is the mod the window blames. An event log carries no native frame or report folder.
func TestCrashHintBlamesTheModTheEventLogNames(t *testing.T) {
	set, ps := testenv.Env(t)
	const lc = "lethal-company"
	p := testenv.Profile(t, ps, lc, "A")
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "BetterRepair.zip"), map[string]string{
		"manifest.json":            `{"name":"BetterRepair","version_number":"1.0.0"}`,
		"plugins/BetterRepair.dll": "MZ",
	})
	if _, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-BetterRepair", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	modsDir, err := ps.ModsDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runsDir(modsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	started := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	if err := datadir.WriteJSON(filepath.Join(runsDir(modsDir), "index.json"), runIndex{
		Runs: []Run{{ID: "r1", Started: started, Outcome: launch.OutcomeCrashed}},
	}); err != nil {
		t.Fatal(err)
	}
	standInWevtutil(t, filepath.Join("testdata", "eventlog", "application-log.xml"))
	svc := NewService(t.TempDir(), set, ps)
	// A symbol server names a frame only when it holds the module's PDB; an event log carries no frame to name anyway.
	svc.Symbols = nil

	hint, err := svc.CrashHint(lc, p.ID)
	if err != nil || hint == nil {
		t.Fatalf("hint = %+v, %v", hint, err)
	}
	if hint.ModKey == "" || hint.ModName != "BetterRepair" || hint.Code != "modCode" {
		t.Fatalf("hint blamed %q %q (%q): %+v", hint.ModKey, hint.ModName, hint.Code, hint)
	}
	if !strings.HasPrefix(hint.Evidence, "Windows event log: System.NullReferenceException: boom") ||
		!strings.Contains(hint.Evidence, "BetterRepair.Logic.CardboardStock.Refill(Int32 count)") {
		t.Fatalf("evidence %q", hint.Evidence)
	}
	if hint.NativeFrame != "" || hint.ReportDir != "" {
		t.Fatalf("an event log carries no frame or report folder: %+v", hint)
	}
}
