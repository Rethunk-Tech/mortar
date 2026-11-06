package curseforge

import (
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// A merged page reads every class from its start to the end of the page, so request count grows with the page number
// and the class count; CurseForge's key allows a modest request rate, so deep pages of a five-class game stall.
func TestMergedSearchRequestsDoNotGrowWithThePageNumber(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	d := countingDriver(t, []string{"1", "2", "3", "4", "5"}, 9000, &requests)
	if _, err := d.Search(t.Context(), source.Query{Game: "g", Key: "1", Sort: source.SortDownloads, Page: source.FirstPage + 100}); err != nil {
		t.Fatal(err)
	}
	if n := requests.Load(); n > 50 {
		t.Fatalf("page 101 of 5 classes took %d requests", n)
	}
}
