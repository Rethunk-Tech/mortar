package control

import (
	"errors"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
)

func profileSource(kind string, modID int, repo string) profile.Source {
	return profile.Source{Kind: kind, ModID: modID, Repo: repo}
}

func TestSourceParamsAreChecked(t *testing.T) {
	t.Parallel()
	if trackingSource("nexus") != nil || trackingSource("github") == nil || trackingSource("") == nil {
		t.Fatal("only nexus keeps tracked mods")
	}
	if _, err := (&Services{}).updateChangelog(t.Context(), Params{Source: "curseforge"}); err == nil {
		t.Fatal("a source without changelogs must be refused")
	}
	if got := sourceID(profileSource("nexus", 42, ""), "nexus"); got != "42" {
		t.Fatalf("nexus id = %q", got)
	}
	if got := sourceID(profileSource("github", 0, "a/b"), "nexus"); got != "" {
		t.Fatalf("github entry matched nexus: %q", got)
	}
}

func TestProfileSetRejectsABadFlag(t *testing.T) {
	t.Parallel()
	if _, err := (&Services{}).profileSet(Params{Key: "hidden", Value: "maybe"}, profile.Profile{}, "id"); err == nil {
		t.Fatal("a non-boolean hidden value must be refused")
	}
}

func TestALaunchIsNotRefusedWhenTheSaveGapLookupFails(t *testing.T) {
	fit, gap := adviceOnly(savessvc.Fit{LastMissing: []savessvc.Lack{{}}}, true, errors.New("could not reach the mod dataset"))
	if gap || len(fit.LastMissing) != 0 {
		t.Fatalf("a failed lookup warned: gap=%v fit=%+v", gap, fit)
	}
	fit, gap = adviceOnly(savessvc.Fit{LastMissing: []savessvc.Lack{{}}}, true, nil)
	if !gap || len(fit.LastMissing) != 1 {
		t.Fatalf("a successful lookup was dropped: gap=%v fit=%+v", gap, fit)
	}
}
