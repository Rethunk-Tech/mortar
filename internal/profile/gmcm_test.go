package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gmcm"
)

func TestSetGmcmOptionIsPendingHistoryAndUndoable(t *testing.T) {
	e, p := undoFixture(t)
	dir, err := e.ProfileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	capture, err := fsx.ReadFile(filepath.Join("..", "gmcm", "testdata", "gmcm-capture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "gmcm"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteFile(gmcm.CapturePath(dir, "smapi:demo.Mod"), capture, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}

	pending, err := e.SetGmcmOption("stardew", p.ID, "smapi:demo.Mod", "", 2, "5")
	if err != nil || len(pending.Edits) != 1 || pending.Edits[0].Name != "Count" {
		t.Fatalf("pending = %+v, %v", pending, err)
	}
	if _, err := e.SetGmcmOption("stardew", p.ID, "smapi:demo.Mod", "", 2, "7"); err == nil {
		t.Fatal("a value above the option's maximum was accepted")
	}
	onDisk, err := gmcm.ReadPending(dir, "smapi:demo.Mod")
	if err != nil || len(onDisk.Edits) != 1 || gmcm.Text(onDisk.Edits[0].Value) != "5" {
		t.Fatalf("pending file = %+v, %v", onDisk, err)
	}
	evs, err := e.History("stardew", p.ID)
	if err != nil || !strings.Contains(evs[0].Label, "Count") {
		t.Fatalf("history head = %+v, %v", evs[0], err)
	}

	if _, err := e.Revert("stardew", p.ID, before[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(gmcm.PendingPath(dir, "smapi:demo.Mod")); !os.IsNotExist(err) {
		t.Fatalf("undo left the pending edit: %v", err)
	}

	if _, err := e.SetGmcmOption("stardew", p.ID, "smapi:demo.Mod", "", 2, "5"); err != nil {
		t.Fatal(err)
	}
	if pending, err := e.SetGmcmOption("stardew", p.ID, "smapi:demo.Mod", "", 2, "3"); err != nil || len(pending.Edits) != 0 {
		t.Fatalf("setting the captured value back = %+v, %v", pending, err)
	}
	if _, err := os.Stat(gmcm.PendingPath(dir, "smapi:demo.Mod")); !os.IsNotExist(err) {
		t.Fatalf("no edit left, yet the pending file stays: %v", err)
	}
}
