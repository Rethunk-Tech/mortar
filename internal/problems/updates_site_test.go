package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

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
	held := HideHeld(UpdatesResult{Updates: []Update{{Key: "k", Installed: "1.4.10-stable", Version: "1.5.3-beta"}}}, nil, false, "")
	if len(held.Updates) != 0 {
		t.Fatalf("beta offered with prereleases off: %+v", held.Updates)
	}
}

func TestUnbumpedManifestInsideNewerDownloadIsNotAnUpdate(t *testing.T) {
	part := inst("k1", "Haru.DesertExpansion", "2.0.8", true)
	part.SourceVersion = "2.0.9"
	older := inst("k2", "Other.Mod", "1.0.0", true)
	older.SourceVersion = "1.0.0"
	rm := fakeMeta{compat: map[string]meta.UpdateResult{
		"Haru.DesertExpansion": {Known: true, Suggested: &meta.Update{Version: "2.0.9", URL: "https://www.nexusmods.com/stardewvalley/mods/31595"}},
		"Other.Mod":            {Known: true, Suggested: &meta.Update{Version: "1.1.0", URL: "https://www.nexusmods.com/stardewvalley/mods/9"}},
	}}
	got := CheckUpdates(context.Background(), rm, testEnv, []framework.Mod{part, older}, false).Updates
	if len(got) != 1 || got[0].ID != "smapi:Other.Mod" {
		t.Fatalf("got %+v", got)
	}
}

func TestSiteURLBuildsModDropLinkFromCatalogKey(t *testing.T) {
	if got := siteURL("stardewvalley", meta.Ref{Site: "ModDrop", ID: 5}); got != "https://www.moddrop.com/stardew-valley/mods/5" {
		t.Fatalf("got %q", got)
	}
	if got := siteURL("unlistedgame", meta.Ref{Site: "ModDrop", ID: 5}); got != "" {
		t.Fatalf("a game without a ModDrop key should give no link, got %q", got)
	}
}
