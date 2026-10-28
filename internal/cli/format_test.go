package cli

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestStoreReportRow(t *testing.T) {
	t.Parallel()
	it := store.Item{Key: "nexus-1-2", Name: "Alpha", Version: "1.0", Size: 1536, LastUsed: time.Date(2026, 3, 1, 23, 59, 0, 0, time.FixedZone("x", -8*3600))}
	got := storeReportRow("stardew", it, "unused")
	want := []string{"stardew", "nexus-1-2", "Alpha", "1.0", "1.5 KB", "2026-03-02", "unused"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("row = %q, want %q", got, want)
	}
}

func TestSweepFixLabel(t *testing.T) {
	t.Parallel()
	for fix, want := range map[string]string{"update": "Update", "off": "Turn off", "": "No fix yet", "other": "No fix yet"} {
		if got := sweepFixLabel(fix); got != want {
			t.Errorf("sweepFixLabel(%q) = %q, want %q", fix, got, want)
		}
	}
}

func TestWinnerText(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		kind, name, want string
	}{
		{framework.WinnerTop, "Alpha", "Alpha"},
		{framework.WinnerChosen, "Beta", "Beta"},
		{framework.WinnerLoadOrder, "Gamma", "Gamma (by load order)"},
		{framework.WinnerNeither, "", "Content Patcher applies neither"},
		{framework.WinnerPerEntry, "", "decided per entry"},
		{framework.WinnerUnclear, "", "unclear"},
		{"", "", "unclear"},
	} {
		if got := winnerText(framework.AssetConflict{WinnerKind: tc.kind, WinnerName: tc.name}); got != tc.want {
			t.Errorf("winnerText(%q) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestConfirmSkipsWithYesAndRefusesWithoutATerminal(t *testing.T) {
	if err := (&cmd{yesFlag: true}).confirm("x", "?"); err != nil {
		t.Fatal(err)
	}
	stdin := os.Stdin
	t.Cleanup(func() { os.Stdin = stdin })
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	os.Stdin = r
	if err := (&cmd{}).confirm("x", "?"); !errors.As(err, new(refusedError)) {
		t.Fatalf("confirm on a pipe = %v", err)
	}
}
