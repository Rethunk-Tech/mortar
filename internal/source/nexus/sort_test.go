package nexus

import (
	"testing"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestEverySortTheDriverListsReachesTheAPI(t *testing.T) {
	want := map[string]nexusmods.ModSort{
		source.SortDownloads:    nexusmods.SortDownloads,
		source.SortEndorsements: nexusmods.SortEndorsements,
		source.SortUpdated:      nexusmods.SortUpdated,
		source.SortNewest:       nexusmods.SortCreated,
		source.SortName:         nexusmods.SortName,
	}
	for _, s := range (Driver{}).Sorts() {
		if got := modSort(s); got != want[s] {
			t.Errorf("modSort(%q) = %q, want %q", s, got, want[s])
		}
	}
}
