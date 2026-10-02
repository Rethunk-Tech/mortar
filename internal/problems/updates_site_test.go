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
