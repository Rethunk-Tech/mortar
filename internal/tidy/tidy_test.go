package tidy

import "testing"

func TestTakeSumsCountsOnceAndDropsEmptyEntries(t *testing.T) {
	var c Collector
	c.Add("tempFolders", "Removed leftover temporary folders", "store", "", 0)
	c.Add("tempFolders", "Removed leftover temporary folders", "store", "", 0, ".tmp-1", ".tmp-2")
	c.Add("leftoverModFolder", "Removed a leftover mod folder", "profiles", "", 5)
	got := c.Take()
	if got.Total != 7 || len(got.Entries) != 2 || got.Entries[0].Count != 2 || got.Entries[1].Names == nil {
		t.Fatalf("report = %+v", got)
	}
	if again := c.Take(); again.Total != 0 || again.Entries == nil || len(again.Entries) != 0 {
		t.Fatalf("second take = %+v", again)
	}
}

func TestClosedCollectorDropsLaterRepairs(t *testing.T) {
	var c Collector
	c.Add("a", "A", "store", "", 1, "x")
	if c.Pending() != 1 {
		t.Fatalf("pending = %d", c.Pending())
	}
	c.Close()
	c.Add("b", "B", "store", "", 1, "y")
	got := c.Take()
	if got.Total != 1 || len(got.Entries) != 1 || got.Entries[0].Code != "a" {
		t.Fatalf("after close = %+v", got)
	}
}

func TestWhereNamesTheProfileAndMergesTheSamePlace(t *testing.T) {
	var c Collector
	c.Add("x", "X", "profiles", "Main", 1, "a")
	c.Add("x", "X", "profiles", "Main", 1, "b")
	c.Add("y", "Y", "dataFolder", "", 2)
	s := Service{Report: &c}
	got := s.Take()
	if got.Total != 4 || len(got.Entries) != 2 {
		t.Fatalf("report = %+v", got)
	}
	if got.Entries[0].Where != "profile Main" || got.Entries[0].Count != 2 {
		t.Fatalf("profile merge = %+v", got.Entries[0])
	}
	if got.Entries[1].Where != "data folder" {
		t.Fatalf("data folder = %+v", got.Entries[1])
	}
}
