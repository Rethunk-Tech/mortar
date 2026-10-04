package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

func TestLoadAfterEditConflictIsShownNotCounted(t *testing.T) {
	a := testdataPack(t, "edit_a")
	b := testdataPack(t, "edit_b")
	a.Name = "Edit A"
	a.LoadAfter = []string{b.UniqueID}
	a.Dependencies = []manifest.Dependency{{UniqueID: b.UniqueID, Required: false}}
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{a, b})
	if len(got.AssetConflicts) != 1 {
		t.Fatalf("got %+v", got.AssetConflicts)
	}
	c := got.AssetConflicts[0]
	if c.Kind != "edit" || !c.Cosmetic || c.WinnerName != "Edit A wins" {
		t.Fatalf("got %+v", c)
	}
	if got.Count() != 0 {
		t.Fatalf("counted %d: %+v", got.Count(), got)
	}
}
