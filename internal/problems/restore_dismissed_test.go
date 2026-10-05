package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestDismissThenRestoreALoadConflict(t *testing.T) {
	testfs.DataHome(t)
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{settings: set, cache: map[string]cached{}}
	conflicts := []framework.AssetConflict{{Kind: "load", Target: "maps/greenhouse"}, {Kind: "edit", Target: "maps/desert", Cosmetic: true}}
	if err := s.DismissAssetConflict(context.Background(), "stardew", "p", "load", "maps/greenhouse"); err != nil {
		t.Fatal(err)
	}
	if err := s.DismissAssetConflict(context.Background(), "stardew", "p", "edit", "maps/desert"); err != nil {
		t.Fatal(err)
	}
	tokens := set.Get().Dismissed[dismissBucket("stardew", "p")]
	if kept, dismissed := hideDismissed(conflicts, tokens); len(kept) != 0 || len(dismissed) != 2 {
		t.Fatalf("kept %d, dismissed %d; want 0 and 2", len(kept), len(dismissed))
	}
	if err := s.RestoreDismissed(context.Background(), "stardew", "p", dismissToken("load", "maps/greenhouse")); err != nil {
		t.Fatal(err)
	}
	tokens = set.Get().Dismissed[dismissBucket("stardew", "p")]
	kept, dismissed := hideDismissed(conflicts, tokens)
	if len(kept) != 1 || kept[0].Kind != "load" || len(dismissed) != 1 {
		t.Fatalf("after restore kept %v, dismissed %d", kept, len(dismissed))
	}
}
