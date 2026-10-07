package nexus

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestEverySortTheDriverListsReachesTheAPI(t *testing.T) {
	want := map[string]string{
		source.SortDownloads: "downloads:", source.SortEndorsements: "endorsements:", source.SortUpdated: "updatedAt:",
		source.SortNewest: "createdAt:", source.SortName: "name:",
	}
	for _, s := range (Driver{}).Sorts() {
		if got := sortClause(s); !strings.HasPrefix(got, want[s]) {
			t.Errorf("sortClause(%q) = %q, want prefix %q", s, got, want[s])
		}
	}
}
