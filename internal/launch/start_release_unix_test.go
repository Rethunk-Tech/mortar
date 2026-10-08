//go:build !windows

package launch

import (
	"path/filepath"
	"testing"
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
