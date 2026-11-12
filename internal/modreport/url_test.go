package modreport

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func TestGitHubIssueURLEncodesAndTruncatesBody(t *testing.T) {
	long := strings.Repeat("x", maxGitHubBody+500)
	got, cut := GitHubIssueURL("Pathoschild/ContentPatcher", "Error in CP", long)
	if !cut {
		t.Fatal("long body not reported cut")
	}
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

func TestGitHubIssueURLCutsAtARuneBoundary(t *testing.T) {
	body := strings.Repeat("a", maxGitHubBody-1) + "é"
	got, _ := GitHubIssueURL("o/r", "t", body)
	if strings.Contains(got, "%EF%BF%BD") {
		t.Fatal("the cut split a rune")
	}
}

func TestGitHubIssueURLFitsTheOpener(t *testing.T) {
	body := "Mod: CatLib\n" + strings.Repeat("é x", 3000)
	got, cut := GitHubIssueURL("Pathoschild/ContentPatcher", "Error in CP", body)
	if len(got) == 0 || len(got) > maxURL || !cut {
		t.Fatalf("len %d cut %v", len(got), cut)
	}
	short, cut := GitHubIssueURL("a/b", "t", "short body")
	if cut || !strings.HasSuffix(short, "title=t&body=short+body") {
		t.Fatalf("short: %q cut %v", short, cut)
	}
}

func TestGitHubIssueURLRefusesANonRepo(t *testing.T) {
	for _, repo := range []string{"", "..", "../x/y", "a/../b", "a/b/c", "a/b?x=1", "a/b#c", "a b/c", "./."} {
		if got, _ := GitHubIssueURL(repo, "t", "b"); got != "" {
			t.Errorf("%q gave %q", repo, got)
		}
	}
}

func TestBuildNamesTheLoaderAndFallsBackToThePage(t *testing.T) {
	res := Build(Input{ModName: "Carry On", ModVersion: "1", Game: "vintagestory", SMAPIVersion: "1.2", LoaderName: "BepInEx", PageURL: "https://mods.vintagestory.at/carryon"})
	if !res.Page || res.GitHub || res.Nexus || res.URL != "https://mods.vintagestory.at/carryon" {
		t.Fatalf("result: %+v", res)
	}
	if !strings.Contains(res.Text, "BepInEx: 1.2") || !strings.Contains(res.Text, "(none captured)") {
		t.Fatalf("text: %q", res.Text)
	}
	if got := Build(Input{ModName: "X", SMAPIVersion: "1"}).Text; !strings.Contains(got, "Loader: 1") || !strings.HasPrefix(got, "Mod: X\n") {
		t.Fatalf("text: %q", got)
	}
}

func TestBuildTakesErrorLinesFromEntries(t *testing.T) {
	log := "[12:00:00 ERROR SomeMod] first\n   second line\n[12:00:01 INFO  SomeMod] ignored\n"
	res := Build(Input{ModName: "Some Mod", LogName: "SomeMod", Entries: launch.ParseLog(log), GitHubRepo: "o/r"})
	if !strings.Contains(res.Text, "[12:00:00 ERROR SomeMod] first\n   second line\n") || strings.Contains(res.Text, "ignored") {
		t.Fatalf("text: %q", res.Text)
	}
}
