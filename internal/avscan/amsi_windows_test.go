//go:build windows

package avscan

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// amsiSample is Microsoft's documented AMSI test string, which an AMSI provider reports as malware, assembled so no
// file in the repo holds it whole.
func amsiSample() string {
	return strings.Join([]string{"AMSI Test Sample: ", "7e72c3ce-861b-4339-8740-0ac1484c1386"}, "")
}

// TestAMSIFlagsTheMicrosoftTestSample needs a real antivirus registered as the AMSI provider (Defender in the test VM); it
// skips where there is none.
func TestAMSIFlagsTheMicrosoftTestSample(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "clean.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if hit, found, err := (amsi{}).Scan(context.Background(), dir); err != nil || found {
		if errors.Is(err, ErrNoScanner) {
			t.Skip("no AMSI provider on this machine")
		}
		t.Fatalf("a clean folder was flagged: %+v %v %v", hit, found, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample.txt"), []byte(amsiSample()), 0o600); err != nil {
		t.Skipf("the antivirus removed the test file as it was written: %v", err)
	}
	hit, found, err := (amsi{}).Scan(context.Background(), dir)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("the antivirus removed the test file before AMSI saw it")
	}
	if err == nil && !found && defenderRealtimeOff(t) {
		t.Skip("Defender real-time protection is off, so no provider flags the sample")
	}
	if err != nil || !found || hit.File != "sample.txt" || hit.Scanner != "AMSI" {
		t.Fatalf("the AMSI test sample was not flagged: %+v %v %v", hit, found, err)
	}
}

// defenderRealtimeOff reports that Defender answered and says real-time protection is off, as on a hosted CI runner; a
// failed query counts as on, so the test still fails where a provider should flag the sample.
func defenderRealtimeOff(t *testing.T) bool {
	out, err := exec.CommandContext(t.Context(), "powershell", "-NoProfile", "-NonInteractive", "-Command",
		"(Get-MpComputerStatus).RealTimeProtectionEnabled").Output()
	return err == nil && strings.EqualFold(strings.TrimSpace(string(out)), "false")
}
