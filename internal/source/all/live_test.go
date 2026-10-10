package all_test

import (
	"os"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/patreon"
)

// TestLiveSims4Sources asks each of The Sims 4's catalog sources for its first page through Mortar's own drivers. It
// uses the real services, so it runs only when MORTAR_LIVE_SOURCES is set; CurseForge also needs MORTAR_CURSEFORGE_KEY.
func TestLiveSims4Sources(t *testing.T) {
	if os.Getenv("MORTAR_LIVE_SOURCES") == "" {
		t.Skip("set MORTAR_LIVE_SOURCES=1 to query the live sources")
	}
	g, ok := components.Game("sims4")
	if !ok {
		t.Fatal("catalog has no sims4")
	}
	for _, gs := range g.Sources {
		t.Run(gs.ID, func(t *testing.T) {
			if gs.ID == "patreon" {
				id, ok := patreon.ParsePostURL("https://www.patreon.com/posts/some-sims-4-cc-12345678")
				if !ok || id != "12345678" {
					t.Fatalf("post url resolved to %q, %v", id, ok)
				}
				return
			}
			drv, ok := source.Get(gs.ID)
			if !ok {
				t.Fatalf("no driver %q", gs.ID)
			}
			srch, ok := drv.Source.(source.Searcher)
			if !ok {
				t.Fatalf("driver %q cannot search", gs.ID)
			}
			if gs.ID == "curseforge" && os.Getenv("MORTAR_CURSEFORGE_KEY") == "" {
				t.Skip("MORTAR_CURSEFORGE_KEY is not set")
			}
			page, err := srch.Search(t.Context(), source.Query{Game: "sims4", Key: gs.Key, Page: source.FirstPage, Sort: source.SortDownloads})
			if err != nil || len(page.Items) == 0 {
				t.Fatalf("%+v %v", page, err)
			}
			t.Logf("%s: total=%d items=%d first=%q second=%q", gs.ID, page.Total, len(page.Items), page.Items[0].Name, page.Items[min(1, len(page.Items)-1)].Name)
		})
	}
}
