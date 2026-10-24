package control

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/controlwire"
)

func TestStaleDiscoveryTempsAreRemovedButARunningWriteIsKept(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, controlwire.FileName)
	old := path + ".123.tmp"
	fresh := path + ".456.tmp"
	other := filepath.Join(dir, "settings.json.789.tmp")
	for _, f := range []string{old, fresh, other} {
		if err := os.WriteFile(f, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	for _, f := range []string{old, other} {
		if err := os.Chtimes(f, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	removeStaleTemps(path, now)
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("an hour-old temp of control.json stayed")
	}
	for _, f := range []string{fresh, other} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("%s was removed: a fresh write or another file's temp must stay", filepath.Base(f))
		}
	}
}
