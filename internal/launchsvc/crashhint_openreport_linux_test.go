//go:build linux

package launchsvc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

// OpenCrashReport hands the system file manager the folder the hint found, never a path the caller names: the stand-in
// xdg-open records its argument instead of a file manager opening on the machine running the tests.
func TestOpenCrashReportShowsTheFolderTheHintFound(t *testing.T) {
	svc, profileID, reportDir := crashDumpService(t, "mod-fault")
	opened := filepath.Join(t.TempDir(), "opened")
	bin := t.TempDir()
	script := "#!/bin/sh\nprintf '%s' \"$1\" > \"$MORTAR_TEST_OPENED.tmp\" && mv \"$MORTAR_TEST_OPENED.tmp\" \"$MORTAR_TEST_OPENED\"\n"
	if err := safefs.WriteFile(filepath.Join(bin, "xdg-open"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MORTAR_TEST_OPENED", opened)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := svc.OpenCrashReport("lethal-company", profileID); err != nil {
		t.Fatal(err)
	}
	// The file manager is started without waiting, so the test waits for it to record what it was given; the script renames a finished file into place so a half-written one is never read.
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if b, err := safefs.ReadFile(opened); err == nil {
			if got := string(b); got != reportDir {
				t.Fatalf("opened %q, want %q", got, reportDir)
			}
			return
		}
	}
	t.Fatalf("the file manager was never given %q", reportDir)
}
