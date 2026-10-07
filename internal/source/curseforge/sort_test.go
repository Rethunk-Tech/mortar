package curseforge

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestEverySortTheDriverListsReachesTheAPI(t *testing.T) {
	want := map[string]int{
		source.SortDownloads: 6, source.SortEndorsements: 2, source.SortUpdated: 3,
		source.SortNewest: 11, source.SortName: 4,
	}
	for _, s := range (Driver{}).Sorts() {
		if got := sortField(s); got != want[s] {
			t.Errorf("sortField(%q) = %d, want %d", s, got, want[s])
		}
	}
}
