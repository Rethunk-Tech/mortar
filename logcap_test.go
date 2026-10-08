package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/support"
)

func TestCappedWriterStopsAtLimitWithOneNotice(t *testing.T) {
	var buf bytes.Buffer
	c := &cappedWriter{w: &buf, left: 10}
	for _, s := range []string{"12345", "6789", "abcdef", "more"} {
		if n, err := c.Write([]byte(s)); err != nil || n != len(s) {
			t.Fatalf("write %q: %d %v", s, n, err)
		}
	}
	got := buf.String()
	if !strings.HasPrefix(got, "123456789") || strings.Count(got, "size limit") != 1 || strings.Contains(got, "abcdef") {
		t.Fatalf("got %q", got)
	}
}

func TestCapCrashLogTruncatesWhenSeenCoversOverLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crash.log")
	if err := os.WriteFile(path, []byte("crash-bytes-here"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "crash.seen"), []byte("16\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	capCrashLogAt(dir, 10)
	info, err := os.Stat(path)
	if err != nil || info.Size() != 0 {
		t.Fatalf("size %v %v", info, err)
	}
	seen, err := fsx.ReadFile(filepath.Join(dir, "crash.seen"))
	if err != nil || strings.TrimSpace(string(seen)) != "0" {
		t.Fatalf("seen %q %v", seen, err)
	}
}

func TestCapCrashLogLeavesFileWhenSeenIsBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crash.log")
	if err := os.WriteFile(path, []byte("crash-bytes-here"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "crash.seen"), []byte("4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	capCrashLogAt(dir, 10)
	got, err := fsx.ReadFile(path)
	if err != nil || string(got) != "crash-bytes-here" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestCappedLogReadsAsCleanShutdown(t *testing.T) {
	dir := t.TempDir()
	f, err := fsx.Create(filepath.Join(dir, "mortar.prev.log"))
	if err != nil {
		t.Fatal(err)
	}
	lg := slog.New(slog.NewTextHandler(&cappedWriter{w: f, left: maxLogBytes}, nil))
	pad := strings.Repeat("x", 200)
	for range maxLogBytes/250 + 10 {
		lg.Info("control: install stardew p", "pad", pad)
	}
	lg.Info("shutdown", "clean", true)
	_ = f.Close()
	if support.DetectLastRunCrashed(dir) {
		t.Errorf("clean shutdown after %d MB of log was reported as a crash", maxLogBytes>>20)
	}
}

// A crash that pushes crash.log over the cap is reported with its trace still there; the next start empties it.
func TestCheckCrashLogKeepsANewTraceUntilItWasReported(t *testing.T) {
	dir := t.TempDir()
	crash := filepath.Join(dir, "crash.log")
	if err := os.WriteFile(crash, make([]byte, 64), 0o600); err != nil {
		t.Fatal(err)
	}
	emptied, crashed := checkCrashLogAt(dir, 32)
	if emptied || !crashed {
		t.Fatalf("first start: emptied %v, crashed %v; want the new trace kept and reported", emptied, crashed)
	}
	if info, _ := os.Stat(crash); info.Size() != 64 {
		t.Fatalf("crash.log is %d bytes at the report, want the whole trace", info.Size())
	}
	if emptied, _ := checkCrashLogAt(dir, 32); !emptied {
		t.Fatal("the next start kept a reported crash.log over the cap")
	}
	if info, _ := os.Stat(crash); info.Size() != 0 {
		t.Fatalf("crash.log is %d bytes after the cap", info.Size())
	}
	if _, crashed := checkCrashLogAt(dir, 32); crashed {
		t.Fatal("an emptied crash.log was reported as a new crash")
	}
}
