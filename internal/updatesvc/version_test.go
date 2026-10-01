package updatesvc

import (
	"path/filepath"
	"testing"
)

func TestUpgraded(t *testing.T) {
	if upgraded("1.1.0", "") {
		t.Fatal("first run is not an upgrade")
	}
	if !upgraded("1.1.0", "1.0.0") {
		t.Fatal("expected newer")
	}
	if upgraded("1.0.0", "1.1.0") {
		t.Fatal("downgrade is not an upgrade")
	}
}

func TestLastRunRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := writeLastRun(dir, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	got, err := readLastRun(dir)
	if err != nil || got != "1.2.3" {
		t.Fatalf("read %q, %v", got, err)
	}
	if filepath.Base(lastRunPath(dir)) != lastRunFile {
		t.Fatalf("unexpected file name %q", lastRunFile)
	}
}
