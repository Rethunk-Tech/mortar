package browse

import (
	"slices"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// MergedSorts are the sorts All sources offers: the ones every source that has them orders by its own API and whose
// values compare across sources. Name and newest are left out because Modrinth and GitHub cannot sort by them, so
// a merged page could not honour them.
var MergedSorts = []string{source.SortDownloads, source.SortUpdated}

// sortMerged orders the interleaved hits of every source by sort, stably, so ties keep each source's own ranking.
// Each source already returned its page in this order; this places the pages against one another. A hit with no
// value for the sort (a GitHub repository has no download count, an unparsable date) never ranks above one with it.
func sortMerged(items []Item, sort string) {
	if !slices.Contains(MergedSorts, sort) {
		return
	}
	slices.SortStableFunc(items, func(a, b Item) int {
		if sort == source.SortDownloads {
			return compareKnown(a.Downloads, a.Downloads > 0, b.Downloads, b.Downloads > 0)
		}
		ta, oka := parseTime(a.Updated)
		tb, okb := parseTime(b.Updated)
		return compareKnown(ta.UnixNano(), oka, tb.UnixNano(), okb)
	})
}

// compareKnown is descending on known values, with unknown ones after every known one and equal to each other.
func compareKnown[T int | int64](a T, aKnown bool, b T, bKnown bool) int {
	switch {
	case aKnown && bKnown:
		if a == b {
			return 0
		}
		if a > b {
			return -1
		}
		return 1
	case aKnown:
		return -1
	case bKnown:
		return 1
	}
	return 0
}

func parseTime(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, s)
	return t, err == nil
}
