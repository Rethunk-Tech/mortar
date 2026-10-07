package nexussvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

func TestPeekDetailsFallsBackToBatchedPage(t *testing.T) {
	c := &meta.Client{CacheDir: t.TempDir()}
	if _, ok := PeekDetails(c, "stardewvalley", 1); ok {
		t.Fatal("empty cache should miss")
	}
	meta.Put(c, PageName("stardewvalley", 1), Details{Page: nexus.Page{Version: "1.0"}, Partial: true})
	if d, ok := PeekDetails(c, "stardewvalley", 1); !ok || d.Page.Version != "1.0" {
		t.Fatalf("page fallback = %+v, %v", d, ok)
	}
	meta.Put(c, DetailsName("stardewvalley", 1), Details{Page: nexus.Page{Version: "2.0"}})
	if d, _ := PeekDetails(c, "stardewvalley", 1); d.Page.Version != "2.0" {
		t.Fatalf("full details should win: %+v", d)
	}
}
