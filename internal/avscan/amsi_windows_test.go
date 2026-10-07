//go:build windows

package avscan

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestAMSIFlagsTheEicarString needs a real antivirus registered as the AMSI provider (Defender in the test VM); it
// skips where there is none. The string is assembled here so no file in the repo holds it whole.
func TestAMSIFlagsTheEicarString(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(dir, "eicar.com"), []byte(eicar()), 0o600); err != nil {
		t.Skipf("the antivirus removed the test file as it was written: %v", err)
	}
	hit, found, err := (amsi{}).Scan(context.Background(), dir)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("the antivirus removed the test file before AMSI saw it")
	}
	if err != nil || !found || hit.File != "eicar.com" || hit.Scanner != "AMSI" {
		t.Fatalf("EICAR was not flagged: %+v %v %v", hit, found, err)
	}
}
