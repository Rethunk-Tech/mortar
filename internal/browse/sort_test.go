package browse

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// GitHub's top repository has stars but no download count; ranked by position it would land beside each other
// source's most downloaded mod, so a missing count never outranks a known one.
func TestSortMergedDownloadsNeverRanksAMissingCountAboveAKnownOne(t *testing.T) {
	items := []Item{
		{Source: "nexus", ID: "1", Downloads: 5000},
		{Source: "github", ID: "a/big", Stars: 90000},
		{Source: "thunderstore", ID: "T-1", Downloads: 90000},
		{Source: "github", ID: "a/small", Stars: 3},
		{Source: "modrinth", ID: "m", Downloads: 12},
	}
	sortMerged(items, source.SortDownloads)
	want := []string{"T-1", "1", "m", "a/big", "a/small"}
	for i, id := range want {
		if items[i].ID != id {
			t.Fatalf("order = %v, want %v", ids(items), want)
		}
	}
}

func TestSortMergedUpdatedPlacesUnparsableDatesLast(t *testing.T) {
	items := []Item{
		{ID: "old", Updated: "2020-01-01T00:00:00Z"},
		{ID: "none"},
		{ID: "new", Updated: "2026-03-01T10:00:00.123Z"},
	}
	sortMerged(items, source.SortUpdated)
	if got := ids(items); got[0] != "new" || got[1] != "old" || got[2] != "none" {
		t.Fatalf("order = %v", got)
	}
}

func TestSortMergedLeavesOtherSortsInSourceOrder(t *testing.T) {
	items := []Item{{ID: "a", Downloads: 1}, {ID: "b", Downloads: 9}}
	sortMerged(items, "")
	sortMerged(items, source.SortName)
	if items[0].ID != "a" {
		t.Fatalf("order = %v", ids(items))
	}
}

func ids(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}
