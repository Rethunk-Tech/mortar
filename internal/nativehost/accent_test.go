package nativehost

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestAccentColorFollowsMortarSettings(t *testing.T) {
	testfs.DataHome(t)
	if got := accentColor(); got != "#D6B17A" {
		t.Fatalf("no settings: %s", got)
	}
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"global":{"accent":"moss"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := accentColor(); got != "#93B86A" {
		t.Fatalf("moss: %s", got)
	}
}
