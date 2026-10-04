package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/support"
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
	f, err := os.Create(filepath.Join(dir, "mortar.prev.log"))
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
