package queue

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryRecordsAFinishedDownload(t *testing.T) {
	f := newFixture(t)
	f.start()
	r := req(10)
	r.BatchID = "batch-1"
	if _, err := f.s.Add(t.Context(), []Request{r}); err != nil {
		t.Fatal(err)
	}
	f.wait("done", f.item(StateDone))
	got := f.s.History()
	if len(got) != 1 {
		t.Fatalf("history %+v", got)
	}
	e := got[0]
	if e.Name != "Alpha" || e.Version != "1.0" || e.Source != "nexus" || e.Profile != "p1" || e.BatchID != "batch-1" || e.Outcome != StateDone {
		t.Fatalf("entry %+v", e)
	}
	if e.Started == 0 || e.Finished == 0 {
		t.Fatalf("times %+v", e)
	}
}

func TestHistoryRecordsSkipAndClear(t *testing.T) {
	f := newFixture(t)
	f.s.Pause()
	if _, err := f.s.Add(t.Context(), []Request{req(10)}); err != nil {
		t.Fatal(err)
	}
	id := f.s.State().Items[0].ID
	f.s.Skip(id)
	got := f.s.History()
	if len(got) != 1 || got[0].Outcome != StateSkipped {
		t.Fatalf("history %+v", got)
	}
	f.s.ClearHistory()
	if len(f.s.History()) != 0 {
		t.Fatal("history was not cleared")
	}
	if _, err := os.Stat(filepath.Join(f.dir, historyFile)); err != nil {
		t.Fatal(err)
	}
}

func TestHistoryKeepsOnlyTheLastThousand(t *testing.T) {
	f := newFixture(t)
	it := &Item{Name: "n", Version: "1", Profile: "p1", started: f.now()}
	f.s.writeHistory(make([]HistoryEntry, historyLimit))
	for range 3 {
		f.s.recordHistory(it, StateDone)
	}
	got := f.s.History()
	if len(got) != historyLimit {
		t.Fatalf("len %d", len(got))
	}
}

func TestHistoryFileIsJSON(t *testing.T) {
	f := newFixture(t)
	f.s.recordHistory(&Item{Name: "n", Version: "2", Profile: "p", SizeKB: 4, started: time.Unix(10, 0)}, StateFailed)
	b, err := os.ReadFile(filepath.Join(f.dir, historyFile))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 || b[0] != '[' {
		t.Fatalf("not a JSON array: %s", b)
	}
}

func TestRetryAllFailed(t *testing.T) {
	f := newFixture(t)
	f.s.Pause()
	if _, err := f.s.Add(t.Context(), []Request{req(30)}); err != nil {
		t.Fatal(err)
	}
	base := HistoryEntry{Game: "stardew", Profile: "p1", ModID: 1, Kind: "nexus"}
	mk := func(file int, outcome string) HistoryEntry {
		e := base
		e.FileID, e.Outcome = file, outcome
		return e
	}
	gh := HistoryEntry{Game: "stardew", Profile: "p1", Repo: "o/r", Tag: "v1", Asset: "a.zip", Outcome: StateFailed}
	bad := mk(0, StateFailed)
	bad.Game = ""
	f.s.writeHistory([]HistoryEntry{
		mk(10, StateFailed), mk(10, StateFailed), // repeated failure: one retry
		mk(11, StateFailed), mk(11, StateDone), // succeeded since
		mk(12, StateDone), mk(12, StateFailed), // newest failed
		mk(30, StateFailed), // already queued
		mk(13, StateSkipped),
		gh, bad,
	})
	got, err := f.s.RetryAllFailed(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got.Requeued != 3 || got.Skipped["superseded"] != 2 || got.Skipped["queued"] != 1 || got.Skipped["incomplete"] != 1 {
		t.Fatalf("result %+v", got)
	}
	if n := len(f.s.State().Items); n != 4 {
		t.Fatalf("%d items queued", n)
	}
	again, err := f.s.RetryAllFailed(t.Context())
	if err != nil || again.Requeued != 0 || again.Skipped["queued"] != 4 {
		t.Fatalf("second run %+v, %v", again, err)
	}
}
