package tidy

import "testing"

func TestTakeSumsCountsOnceAndDropsEmptyEntries(t *testing.T) {
	var c Collector
	c.Add("Removed leftover temporary folders", "store", 0)
	c.Add("Removed leftover temporary folders", "store/stardew", 0, ".tmp-1", ".tmp-2")
	c.Add("Repaired file names", "profiles", 5)
	got := c.Take()
	if got.Total != 7 || len(got.Entries) != 2 || got.Entries[0].Count != 2 || got.Entries[1].Names == nil {
		t.Fatalf("report = %+v", got)
	}
	if again := c.Take(); again.Total != 0 || again.Entries == nil || len(again.Entries) != 0 {
		t.Fatalf("second take = %+v", again)
	}
}
