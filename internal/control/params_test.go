package control

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func profileSource(kind string, modID int, repo string) profile.Source {
	return profile.Source{Kind: kind, ModID: modID, Repo: repo}
}

func TestSourceAndInstallParamsAreChecked(t *testing.T) {
	t.Parallel()
	if trackingSource("nexus") != nil || trackingSource("github") == nil || trackingSource("") == nil {
		t.Fatal("only nexus keeps tracked mods")
	}
	if needNoInstall(Params{}, "status") != nil || needNoInstall(Params{Install: "x"}, "status") == nil {
		t.Fatal("an install-targeted status must be refused until launchsvc has one")
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
