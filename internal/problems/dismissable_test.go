package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

func TestHideDismissedSoftOnly(t *testing.T) {
	in := []framework.AssetConflict{
		{Kind: "load", Target: "a"},
		{Kind: "edit", Target: "b"},
	}
	got, dismissed := hideDismissed(in, []string{dismissToken("edit", "b"), dismissToken("load", "a")})
	if len(got) != 0 || len(dismissed) != 2 {
		t.Fatalf("got %+v dismissed %+v", got, dismissed)
	}
}

func TestDismissedLoadFailureReturnsWithTheNextRun(t *testing.T) {
	f := LoadFailure{Kind: "missing-dependency", Plugin: "Fixture Plugin 1.2.3", Message: "missing dependencies: x"}
	other := LoadFailure{Kind: "load-exception", Plugin: "Other 1.0.0", Message: "boom"}
	tokens := []string{loadFailureToken("run-1", f)}
	got, dismissed := hideDismissedLoadFailures([]LoadFailure{f, other}, "run-1", tokens)
	if len(got) != 1 || got[0].Plugin != "Other 1.0.0" {
		t.Fatalf("same run: %+v", got)
	}
	if len(dismissed) != 1 || dismissed[0].LoadFailure == nil || dismissed[0].LoadFailure.Plugin != f.Plugin || dismissed[0].Token != tokens[0] {
		t.Fatalf("the hidden row cannot be restored: %+v", dismissed)
	}
	if got, _ := hideDismissedLoadFailures([]LoadFailure{f, other}, "run-2", tokens); len(got) != 2 {
		t.Fatalf("next run: %+v", got)
	}
}

func TestDismissableRowsIncludeTheRunLogFailures(t *testing.T) {
	r := Result{LoadFailures: []LoadFailure{
		{Kind: "missing-dependency", Plugin: "Fixture Plugin 1.2.3", Message: "missing dependencies: x"},
		{Kind: KindGameVersion, Plugin: "bepinex5"},
		{Kind: KindGameSetting, Plugin: "cfg"},
	}}
	rows := DismissableRows(r)
	if len(rows) != 1 || rows[0].Kind != "loadFailure" || rows[0].Failure.Plugin != "Fixture Plugin 1.2.3" {
		t.Fatalf("rows = %+v", rows)
	}
}
