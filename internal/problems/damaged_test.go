package problems

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestDamagedRowsAreOnePerStoreItemWithAFewFileNames(t *testing.T) {
	mods := []Installed{
		{Key: "nexus-1-2", Manifest: manifest.Manifest{Name: "Pack"}},
		{Key: "nexus-1-2", Manifest: manifest.Manifest{Name: "Pack Extra"}},
		{Key: "local-fine", Manifest: manifest.Manifest{Name: "Fine"}},
	}
	damaged := map[string]store.Damage{"nexus-1-2": {
		Missing: []string{"a", "b", "c"}, Changed: []string{"d", "e"}, Extra: []string{"f"},
	}}
	rows := damagedRows(mods, damaged)
	if len(rows) != 1 || rows[0].Name != "Pack" || rows[0].Missing != 3 || rows[0].Extra != 1 || len(rows[0].Files) != maxDamagedFiles {
		t.Fatalf("rows = %+v", rows)
	}
}
