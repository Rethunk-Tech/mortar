package nexussvc

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

func TestPeekDetailsPrefersTheNewerFetch(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	c := &meta.Client{CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	meta.Put(c, DetailsName("g", 1), Details{Category: "old"})
	now = now.Add(time.Hour)
	meta.Put(c, PageName("g", 1), Details{Category: "new", Partial: true})
	if d, ok := PeekDetails(c, "g", 1); !ok || d.Category != "new" {
		t.Fatalf("got %+v ok=%v", d, ok)
	}
}
