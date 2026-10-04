package datasvc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/testenv/testfs"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/testenv"
)

func TestRemoveItemsRefusesAReferencedKey(t *testing.T) {
	testfs.DataHome(t)
	items, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "Farm")
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	itemDir := filepath.Join(root, "store", "stardew", "nexus-1-1")
	if err := os.MkdirAll(itemDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(itemDir, "m.bin"), []byte("mod"), 0o600); err != nil {
		t.Fatal(err)
	}
	profPath := filepath.Join(root, "profiles", "stardew", p.ID, "profile.json")
	setEntry(t, profPath, "nexus-1-1")
	svc := NewService(items, profiles, nil)
	err = svc.RemoveItems("stardew", []string{"nexus-1-1"})
	if !errors.Is(err, errInUse) {
		t.Fatalf("in use: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "Farm") {
		t.Fatalf("error must name the profile, got %v", err)
	}
}
