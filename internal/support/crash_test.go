package support

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/doctor"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/problems"
)

func mortarDataDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDetectLastRunCrashedGrew(t *testing.T) {
	dir := mortarDataDir(t)
	if err := os.WriteFile(filepath.Join(dir, crashLogName), []byte(strings.Repeat("x", 80)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, crashSeenName), []byte("10\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !DetectLastRunCrashed(dir) {
		t.Fatal("want crash when crash.log grew")
	}
	if DetectLastRunCrashed(dir) {
		t.Fatal("want the same growth reported once")
	}
}

func TestDetectLastRunCrashedUnchanged(t *testing.T) {
	dir := mortarDataDir(t)
	body := []byte(strings.Repeat("y", 40))
	if err := os.WriteFile(filepath.Join(dir, crashLogName), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, crashSeenName), []byte("40\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if DetectLastRunCrashed(dir) {
		t.Fatal("want no crash when crash.log size matches crash.seen")
	}
}

func TestDetectLastRunCrashedMissingFiles(t *testing.T) {
	dir := mortarDataDir(t)
	if DetectLastRunCrashed(dir) {
		t.Fatal("want no crash when crash.log and mortar.prev.log are missing")
	}
}

func TestDetectLastRunCrashedCleanLine(t *testing.T) {
	dir := mortarDataDir(t)
	line := "time=2026-01-02T03:04:05.000Z level=INFO msg=shutdown clean=true\n"
	if err := os.WriteFile(filepath.Join(dir, prevLogName), []byte("started\n"+line), 0o600); err != nil {
		t.Fatal(err)
	}
	if DetectLastRunCrashed(dir) {
		t.Fatal("want no crash when mortar.prev.log ends with the clean-shutdown line")
	}
	s := NewService("1.2.3", func(string) problems.Environment { return problems.Environment{} }, "", nil)
	if s.LastRunCrashed() {
		t.Fatal("LastRunCrashed should match DetectLastRunCrashed")
	}
}

func TestDetectLastRunCrashedLateLinesAfterShutdown(t *testing.T) {
	dir := mortarDataDir(t)
	body := "time=2026-01-02T03:04:05.000Z level=INFO msg=shutdown clean=true\n" +
		"time=2026-01-02T03:04:05.001Z level=INFO msg=\"LAN sharing: LAN sharing service is shut down\"\n"
	if err := os.WriteFile(filepath.Join(dir, prevLogName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if DetectLastRunCrashed(dir) {
		t.Fatal("want no crash when lines follow the clean-shutdown record")
	}
}

func TestDetectLastRunCrashedUncleanPrevLog(t *testing.T) {
	dir := mortarDataDir(t)
	if err := os.WriteFile(filepath.Join(dir, prevLogName), []byte("time=2026-01-02T03:04:05.000Z level=INFO msg=still running\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !DetectLastRunCrashed(dir) {
		t.Fatal("want crash when mortar.prev.log does not end with the clean-shutdown line")
	}
}

func TestDiagnosticsSectionLogTails(t *testing.T) {
	home := "/home/me"
	dir := t.TempDir()
	prev := home + "/.local/share/mortar started\nprev-line\n"
	crash := "fatal at " + home + "/.cache/x\ncrash-line\n"
	if err := os.WriteFile(filepath.Join(dir, prevLogName), []byte(prev), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, crashLogName), []byte(crash), 0o600); err != nil {
		t.Fatal(err)
	}
	report := doctor.Report{Checks: []doctor.Check{{Status: "ok", Detail: "fine"}}}
	got := diagnosticsSection(report, home, dir)
	if !strings.Contains(got, "**"+prevLogName+"**") || !strings.Contains(got, "**"+crashLogName+"**") {
		t.Fatalf("missing log sections: %q", got)
	}
	if !strings.Contains(got, "prev-line") || !strings.Contains(got, "crash-line") {
		t.Fatalf("missing log tails: %q", got)
	}
	if strings.Contains(got, home) {
		t.Fatalf("home not hidden: %q", got)
	}
	if !strings.Contains(got, "~/.local/share/mortar") {
		t.Fatalf("hidden home missing: %q", got)
	}

	absent := diagnosticsSection(report, home, t.TempDir())
	if strings.Contains(absent, prevLogName) || strings.Contains(absent, crashLogName) {
		t.Fatalf("absent files still listed: %q", absent)
	}

	var huge strings.Builder
	for range 80 {
		huge.WriteString(strings.Repeat("z", 200) + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, prevLogName), []byte(huge.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, crashLogName), []byte(huge.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	got = diagnosticsSection(report, home, dir)
	if len(got) > maxDiagnostics+64 {
		t.Fatalf("over budget: %d bytes", len(got))
	}
}

func TestLastRunCrashedReportsOncePerRun(t *testing.T) {
	lastRunCrashed.Store(true)
	s := &Service{}
	if !s.LastRunCrashed() || s.LastRunCrashed() {
		t.Fatal("want true once, then false")
	}
}

func TestDetectLastRunCrashedTruncatedResetsSeen(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, crashLogName), []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, crashSeenName), []byte("9999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if DetectLastRunCrashed(dir) {
		t.Fatal("a truncated crash.log must not count as a new crash")
	}
	got, err := fsx.ReadFile(filepath.Join(dir, crashSeenName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != "5" {
		t.Fatalf("seen after truncate = %q", got)
	}
}
