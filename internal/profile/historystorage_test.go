package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrimHistoryDropsOldEventsAndRecordsIt(t *testing.T) {
	t.Parallel()
	e := envWithAB(t)
	p := addFarmMod(t, e)
	for range 3 {
		e.mustSetMeA(t, p.ID, false)
		e.mustSetMeA(t, p.ID, true)
	}
	before, err := e.History("stardew", p.ID)
	if err != nil || len(before) < 6 {
		t.Fatalf("history = %d events, %v", len(before), err)
	}
	usage, err := e.TrimHistory("stardew", p.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	after, err := e.History("stardew", p.ID)
	if err != nil || len(after) != 3 || usage.Events != 3 {
		t.Fatalf("after trim: %d events, usage %+v, %v", len(after), usage, err)
	}
	if after[0].Kind != historyTrimmed || after[0].Count != len(before)-2 {
		t.Fatalf("newest event = %+v", after[0])
	}
	dir, _ := e.profileDir("stardew", p.ID)
	snaps, _ := os.ReadDir(filepath.Join(dir, snapshotsDir))
	if len(snaps) > 3 {
		t.Fatalf("%d snapshot files left for 3 events", len(snaps))
	}
	if _, err := e.TrimHistory("stardew", p.ID, 0); err == nil {
		t.Fatal("trim to zero accepted")
	}
}
