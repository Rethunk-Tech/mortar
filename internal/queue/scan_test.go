package queue

import (
	"io/fs"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestADetectionFailsTheItemAndInstallAnywayRunsItAgain(t *testing.T) {
	f := newFixture(t)
	var flagged atomic.Bool
	flagged.Store(true)
	f.installErr = func() error {
		if flagged.Load() {
			return usererr.Wrap(usererr.Malware, &store.DetectedError{Game: "stardew", Key: "nexus-1-10", Name: "Test.Threat", File: "Mod/a.dll", Scanner: "fake"})
		}
		return nil
	}
	f.start()
	f.add(req(10))
	st := f.wait("detection", f.item(StateFailed))
	it := st.Items[0]
	if it.ErrorKind != FailMalware || it.Detection == nil || it.Detection.Name != "Test.Threat" || it.Detection.File != "Mod/a.dll" || it.Detection.Key != "nexus-1-10" || !strings.Contains(it.Error, "Test.Threat") {
		t.Fatalf("item = %+v", it)
	}
	flagged.Store(false)
	if err := f.s.InstallAnyway(it.ID); err != nil {
		t.Fatal(err)
	}
	f.wait("done after install anyway", f.item(StateDone))
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.allowed) != 1 || !strings.HasSuffix(f.allowed[0], "|nexus-1-10|"+it.Name+"|Test.Threat") {
		t.Fatalf("allowed = %v", f.allowed)
	}
	if hist := f.s.History(); len(hist) == 0 || hist[len(hist)-1].Override != "Test.Threat" {
		t.Fatalf("history = %+v", hist)
	}
}

func TestInstallAnywayRefusesAnItemWithNoDetection(t *testing.T) {
	f := newFixture(t)
	if err := f.s.InstallAnyway("nope"); err == nil {
		t.Fatal("no error")
	}
}

func TestAFileTheAntivirusRemovedFromStagingFailsAsMalwareWithNoInstallAnyway(t *testing.T) {
	f := newFixture(t)
	f.installErr = func() error {
		return &fs.PathError{Op: "open", Path: filepath.Join(f.s.downloadRoot(), "m.zip"), Err: fs.ErrNotExist}
	}
	f.start()
	f.add(req(10))
	it := f.wait("removed", f.item(StateFailed)).Items[0]
	if it.ErrorKind != FailMalware || it.Detection == nil || !it.Detection.Removed || it.Detection.Scanner != store.RealTimeScanner || it.Detection.Name != "" {
		t.Fatalf("item = %+v", it)
	}
	if err := f.s.InstallAnyway(it.ID); err == nil {
		t.Fatal("install anyway was offered for a file that is gone")
	}
}
