package datasvc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestRemoveItemsRefusesAReferencedKey(t *testing.T) {
	testfs.DataHome(t)
	items, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	root, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	itemDir := filepath.Join(root, "store", "stardew", "nexus-1-1")
	if err := os.MkdirAll(itemDir, 0o750); err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, itemDir, "m.bin", "mod")
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
