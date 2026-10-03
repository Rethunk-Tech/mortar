package modreport

import (
	"strings"
	"testing"
)

func TestGitHubIssueURLEncodesAndTruncatesBody(t *testing.T) {
	long := strings.Repeat("x", maxGitHubBody+500)
	got := GitHubIssueURL("Pathoschild/ContentPatcher", "Error in CP", long)
	if !strings.HasPrefix(got, "https://github.com/Pathoschild/ContentPatcher/issues/new?") {
		t.Fatalf("prefix: %q", got)
	}
	if !strings.Contains(got, "title=Error+in+CP") {
		t.Fatalf("title: %q", got)
	}
	if strings.Contains(got, strings.Repeat("x", maxGitHubBody+1)) {
		t.Fatal("body not truncated in URL")
	}
}

func TestNexusBugsURL(t *testing.T) {
	got := NexusBugsURL("stardewvalley", 1915)
	want := "https://www.nexusmods.com/stardewvalley/mods/1915?tab=bugs"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildGitHubPrefersRepo(t *testing.T) {
	res := Build(Input{
		ModName: "Content Patcher", ModVersion: "2.0.0",
		GameVersion: "1.6.15", SMAPIVersion: "4.1.10", MortarVersion: "0.1.0",
		ErrorLines: []string{"[12:00 ERROR CP] boom"},
		GitHubRepo: "Pathoschild/ContentPatcher",
	})
	if !res.GitHub || !strings.Contains(res.URL, "github.com/Pathoschild/ContentPatcher/issues/new") {
		t.Fatalf("url: %q", res.URL)
	}
	if !strings.Contains(res.Text, "Mod: Content Patcher 2.0.0") {
		t.Fatalf("text: %q", res.Text)
	}
}
