package modrinth

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestEverySortTheDriverListsReachesTheAPI(t *testing.T) {
	want := map[string]string{
		source.SortDownloads: "downloads", source.SortEndorsements: "follows",
		source.SortUpdated: "updated", source.SortNewest: "newest",
	}
	for _, s := range (Driver{}).Sorts() {
		if got := sortIndex(s); got != want[s] {
			t.Errorf("sortIndex(%q) = %q, want %q", s, got, want[s])
		}
	}
	if sortIndex(source.SortName) != "relevance" {
		t.Error("name has no Modrinth index")
	}
}
