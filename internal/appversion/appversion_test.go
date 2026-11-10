package appversion

import (
	"os"
	"strings"
	"testing"
)

func TestFromConfigReadsInfoVersion(t *testing.T) {
	got, err := FromConfig([]byte("version: '3'\n\ninfo:\n  productName: \"Mortar\"\n  version: \"1.2.3\" # comment\n"))
	if err != nil || got != "1.2.3" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := FromConfig([]byte("version: '3'\n")); err == nil {
		t.Fatal("no info.version should be an error")
	}
}

func TestRepoConfigHasAVersion(t *testing.T) {
	b, err := os.ReadFile("../../build/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FromConfig(b); err != nil {
		t.Fatal(err)
	}
}

func TestABuildPastItsReleaseReadsTheNextRelease(t *testing.T) {
	if got := NextMinor("1.4.2"); got != "1.5.0" {
		t.Errorf("NextMinor = %q", got)
	}
	for _, bad := range []string{"dev", "1.x.2", "1.2.x"} {
		if got := NextMinor(bad); got != bad {
			t.Errorf("NextMinor(%q) = %q, want it unchanged", bad, got)
		}
	}
	t.Cleanup(func() { pastRelease = "" })
	for stamp, want := range map[string]string{"": "1.4.2", "0": "1.4.2", "4": "1.5.0"} {
		pastRelease = stamp
		if got := ForCatalog("1.4.2"); got != want {
			t.Errorf("past %q: ForCatalog = %q, want %q", stamp, got, want)
		}
	}
}

// UserAgent names Mortar to a site, and the unversioned name for a build that carries no version.
func TestUserAgentNamesTheBuild(t *testing.T) {
	if got := UserAgent("1.2.3"); got != "Mortar/1.2.3" {
		t.Errorf("UserAgent = %q", got)
	}
	if got := UserAgent(""); got != "Mortar" {
		t.Errorf("UserAgent of no version = %q", got)
	}
}

// 0.3.8 wrote backups under this name; a different one would copy a second backup over the first edit's original.
func TestSteamBackupSuffixIsWhatEarlierReleasesWrote(t *testing.T) {
	if SteamBackupSuffix != ".mortar.bak" {
		t.Fatalf("SteamBackupSuffix = %q", SteamBackupSuffix)
	}
}

// The build names this exact binary: a clean tree is its commit, and a tree without a stamp (or with uncommitted
// changes, where every local build shares the commit) adds the executable's size and time so a cache it wrote is
// known to be stale. Commit is the short form of the same revision, flagged when the tree was modified.
func TestBuildAndCommitNameTheVCSStamp(t *testing.T) {
	rev, modified := vcs()
	build := Build()
	switch {
	case rev != "" && modified == "false":
		if build != rev {
			t.Errorf("Build = %q, want the clean revision %q", build, rev)
		}
	case rev != "":
		if !strings.HasPrefix(build, rev+"+") {
			t.Errorf("Build = %q, want the modified revision %q with the executable's stamp", build, rev)
		}
	case !strings.HasPrefix(build, "+"):
		t.Errorf("Build = %q, want the executable's stamp for a build with no revision", build)
	}
	want := rev
	if len(want) > 8 {
		want = want[:8]
	}
	if rev != "" && modified == "true" {
		want += "-modified"
	}
	if got := Commit(); got != want {
		t.Errorf("Commit = %q, want %q", got, want)
	}
}
