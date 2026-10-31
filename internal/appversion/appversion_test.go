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

func TestABuildPastItsReleaseReadsTheNextRelease(t *testing.T) {
	if got := NextMinor("1.4.2"); got != "1.5.0" {
		t.Errorf("NextMinor = %q", got)
	}
	if got := NextMinor("dev"); got != "dev" {
		t.Errorf("a version that is not x.y.z changed to %q", got)
	}
	t.Cleanup(func() { pastRelease = "" })
	for stamp, want := range map[string]string{"": "1.4.2", "0": "1.4.2", "4": "1.5.0"} {
		pastRelease = stamp
		if got := ForCatalog("1.4.2"); got != want {
			t.Errorf("past %q: ForCatalog = %q, want %q", stamp, got, want)
		}
	}
}

// 0.3.8 wrote backups under this name; a different one would copy a second backup over the first edit's original.
func TestSteamBackupSuffixIsWhatEarlierReleasesWrote(t *testing.T) {
	if SteamBackupSuffix != ".mortar.bak" {
		t.Fatalf("SteamBackupSuffix = %q", SteamBackupSuffix)
	}
}
