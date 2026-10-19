package appversion

import (
	"os"
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

func TestABuildPastItsReleaseReadsTheNextPatch(t *testing.T) {
	if got := NextPatch("0.3.7"); got != "0.3.8" {
		t.Errorf("NextPatch = %q", got)
	}
	if got := NextPatch("dev"); got != "dev" {
		t.Errorf("a version that is not x.y.z changed to %q", got)
	}
	t.Cleanup(func() { pastRelease = "" })
	for stamp, want := range map[string]string{"": "0.3.7", "0": "0.3.7", "4": "0.3.8"} {
		pastRelease = stamp
		if got := ForCatalog("0.3.7"); got != want {
			t.Errorf("past %q: ForCatalog = %q, want %q", stamp, got, want)
		}
	}
}
