//go:build !windows

package launch

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// A tool started from the Tools menu leaves no captured output behind once it exits.
func TestStartReleasesTheCaptureOnExit(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	exited, err := Start("/", "/bin/sh", "-c", "echo tool-output")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseOnExit(exited)
	left, _ := filepath.Glob(filepath.Join(tmp, "mortar-launch-*"))
	if len(left) != 0 || captureOf(exited) != nil {
		t.Fatalf("left %d capture files, captured entry kept: %v", len(left), captureOf(exited) != nil)
	}
}

// A capture file older than a day is swept by the next launch; a fresh one stays.
func TestNewCaptureSweepsStaleFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	stale, fresh := filepath.Join(tmp, "mortar-launch-old.log"), filepath.Join(tmp, "mortar-launch-new.log")
	for _, p := range []string{stale, fresh} {
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * staleCapture)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	c := newCapture(exec.CommandContext(t.Context(), "true"))
	defer c.release()
	if _, err := os.Stat(stale); err == nil {
		t.Fatal("stale capture file kept")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("fresh capture file removed")
	}
}
