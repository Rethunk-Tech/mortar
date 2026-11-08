package vsmoddb

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestOrderBy(t *testing.T) {
	t.Parallel()
	for sort, want := range map[string]string{
		source.SortDownloads:    "downloads",
		source.SortEndorsements: "follows",
		source.SortUpdated:      "lastreleased",
		source.SortNewest:       "asset.created",
		"name":                  "",
		"":                      "",
	} {
		if got := orderBy(sort); got != want {
			t.Errorf("orderBy(%q) = %q, want %q", sort, got, want)
		}
	}
}
