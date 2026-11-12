package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestHistoryRecordsAFinishedDownload(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.start()
	r := req(10)
	r.BatchID = "batch-1"
	f.add(r)
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
	t.Parallel()
	f := newFixture(t)
	f.s.Pause()
	f.add(req(10))
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
	t.Parallel()
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
	t.Parallel()
	f := newFixture(t)
	f.s.recordHistory(&Item{Name: "n", Version: "2", Profile: "p", SizeKB: 4, started: time.Unix(10, 0)}, StateFailed)
	f.s.FlushHistory()
	b, err := os.ReadFile(filepath.Join(f.dir, historyFile))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 || b[0] != '[' {
		t.Fatalf("not a JSON array: %s", b)
	}
}

func TestRetryAllFailed(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.s.Pause()
	f.add(req(30))
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

func TestHistoryReadsFileOnce(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.s.recordHistory(&Item{Name: "a", started: f.now()}, StateDone)
	f.s.FlushHistory()
	if err := os.Remove(f.s.historyPath()); err != nil {
		t.Fatal(err)
	}
	f.s.recordHistory(&Item{Name: "b", started: f.now()}, StateDone)
	f.s.FlushHistory()
	var onDisk []HistoryEntry
	if _, err := datadir.ReadJSON(f.s.historyPath(), &onDisk); err != nil {
		t.Fatal(err)
	}
	if len(onDisk) != 2 || onDisk[0].Name != "a" || onDisk[1].Name != "b" {
		t.Fatalf("history on disk %+v", onDisk)
	}
	got := f.s.History()
	got[0].Name = "mutated"
	if f.s.History()[0].Name != "a" {
		t.Fatal("History handed out the cached slice")
	}
}

func TestHistoryReadsAnIndentedFileAndWritesItCompact(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	old, _ := json.MarshalIndent([]HistoryEntry{{Name: "old", Outcome: StateDone}}, "", "  ")
	if err := os.WriteFile(f.s.historyPath(), old, 0o600); err != nil {
		t.Fatal(err)
	}
	f.s.recordHistory(&Item{Name: "new", started: f.now()}, StateDone)
	f.s.FlushHistory()
	if got := f.s.History(); len(got) != 2 || got[0].Name != "old" {
		t.Fatalf("history %+v", got)
	}
	b, err := os.ReadFile(f.s.historyPath())
	if err != nil || bytes.Contains(b, []byte("\n")) {
		t.Fatalf("history file not compact: %v %s", err, b)
	}
}

func TestHistoryKeepsADamagedFileAsideInsteadOfLosingIt(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if err := os.WriteFile(f.s.historyPath(), []byte(`[{"name":"old"},{"name":"newer"`), 0o600); err != nil {
		t.Fatal(err)
	}
	f.s.recordHistory(&Item{Name: "b", started: f.now()}, StateDone)
	f.s.FlushHistory()
	var onDisk []HistoryEntry
	if _, err := datadir.ReadJSON(f.s.historyPath(), &onDisk); err != nil {
		t.Fatal(err)
	}
	if len(onDisk) != 1 || onDisk[0].Name != "b" {
		t.Fatalf("history on disk %+v", onDisk)
	}
	aside, err := filepath.Glob(f.s.historyPath() + ".corrupt-*")
	if err != nil || len(aside) != 1 {
		t.Fatalf("damaged file moved aside to %v, %v", aside, err)
	}
	kept, err := os.ReadFile(aside[0])
	if err != nil || !bytes.Contains(kept, []byte(`"name":"newer"`)) {
		t.Fatalf("damaged file kept %q, %v", kept, err)
	}
}

func TestHistoryRetryKeepsThePackageFile(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.s.recordHistory(&Item{Kind: KindInstall, Package: "mod", Source: profile.KindCurseForge, PackageFile: 77}, StateFailed)
	e := f.s.History()[0]
	if got := e.request().PackageFile; got != 77 {
		t.Fatalf("retry PackageFile = %d, want 77", got)
	}
}

func TestHistoryBurstIsWrittenOnceAfterTheDelayAndAtFlush(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.s.histDelay = time.Hour
	const n = 50
	for range n {
		f.s.recordHistory(&Item{Name: "m", started: f.now()}, StateDone)
	}
	if _, err := os.Stat(f.s.historyPath()); err == nil {
		t.Fatal("a burst of finishes wrote the file before the delay or a flush")
	}
	f.s.FlushHistory()
	var onDisk []HistoryEntry
	if _, err := datadir.ReadJSON(f.s.historyPath(), &onDisk); err != nil || len(onDisk) != n {
		t.Fatalf("after flush the file holds %d entries, %v; want %d", len(onDisk), err, n)
	}
}

func TestHistoryIsWrittenOnItsOwnAfterTheDelay(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.s.histDelay = 10 * time.Millisecond
	f.s.recordHistory(&Item{Name: "m", started: f.now()}, StateDone)
	deadline := time.Now().Add(5 * time.Second)
	for {
		var onDisk []HistoryEntry
		if _, err := datadir.ReadJSON(f.s.historyPath(), &onDisk); err == nil && len(onDisk) == 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the history was never written")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Run's wait promises nothing writes the queue's files after it returns, and the history's delayed write is one of
// them: a caller whose TempDir goes on return must not have that timer fire into it afterwards.
func TestRunWaitsForTheHistoriesDelayedWrite(t *testing.T) {
	t.Parallel()
	s, err := New(Deps{Dir: t.TempDir(), Now: func() time.Time { return time.Unix(1, 0).UTC() }})
	if err != nil {
		t.Fatal(err)
	}
	s.histDelay = time.Hour
	s.recordHistory(&Item{Name: "m", started: s.d.Now()}, StateDone)
	ctx, cancel := context.WithCancel(t.Context())
	wait := Run(ctx, s, make(chan nxmsvc.Assignment))
	cancel()
	wait()
	if _, err := os.Stat(s.historyPath()); err != nil {
		t.Fatalf("wait returned with the history still pending: %v", err)
	}
}
