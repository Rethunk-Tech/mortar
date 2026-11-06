package browse

import (
	"slices"
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
	slices.SortStableFunc(items, byMergedSort(source.SortDownloads))
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
	slices.SortStableFunc(items, byMergedSort(source.SortUpdated))
	if got := ids(items); got[0] != "new" || got[1] != "old" || got[2] != "none" {
		t.Fatalf("order = %v", got)
	}
}

// A sort outside MergedSorts is not ranked across sources: their pages interleave, each in its own order, instead of
// being ordered by download count.
func TestASortOutsideMergedSortsKeepsEachSourcesOwnOrder(t *testing.T) {
	a := &rankSource{id: "oa", downloads: []int{1, 2, 3}, sorts: []string{source.SortDownloads, source.SortName}}
	b := &rankSource{id: "ob", downloads: []int{900, 800}, sorts: []string{source.SortDownloads, source.SortName}}
	game := rankGame(t, a, b)
	page, err := (&Client{}).searchAll(t.Context(), game, "", 1, Filter{Sort: source.SortName})
	if err != nil {
		t.Fatal(err)
	}
	if got := downloadsOf(page.Items); !slices.Equal(got, []int{1, 900, 2, 800, 3}) {
		t.Fatalf("name sort order = %v, want the sources interleaved, not ranked by downloads", got)
	}
}

func ids(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}
