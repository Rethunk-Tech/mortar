package problems

import "testing"

func TestNexusUpdateFollowsSuggestedSite(t *testing.T) {
	keys := []string{"Nexus:12345", "CurseForge:67890"}
	if got := nexusUpdate(keys, "https://www.nexusmods.com/stardewvalley/mods/12345"); got != 12345 {
		t.Fatalf("nexus url: %d", got)
	}
	if got := nexusUpdate(keys, "https://www.curseforge.com/stardewvalley/mods/weather-wonders"); got != 0 {
		t.Fatalf("curseforge url: %d", got)
	}
}

func TestStableSuffixIsNotPrerelease(t *testing.T) {
	for v, want := range map[string]bool{"1.4.10-stable": false, "1.5.3-beta": true, "2.0.0-Release": false, "1.0.0": false, "3.1.0-rc.1": true} {
		if got := hasPrerelease(v); got != want {
			t.Errorf("hasPrerelease(%q) = %v", v, got)
		}
	}
	held := HideHeld(UpdatesResult{Updates: []Update{{Key: "k", Installed: "1.4.10-stable", Version: "1.5.3-beta"}}}, nil, false)
	if len(held.Updates) != 0 {
		t.Fatalf("beta offered with prereleases off: %+v", held.Updates)
	}
}
