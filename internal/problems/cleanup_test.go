package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

func TestCleanupHintsFindsUnusedFrameworkNeededOnlyByDisabledMod(t *testing.T) {
	spaceCore := manifest.Manifest{
		Name:     "SpaceCore",
		UniqueID: "spacechase0.SpaceCore",
	}
	oldPack := manifest.Manifest{
		UniqueID: "Example.OldPack",
		Dependencies: []manifest.Dependency{{
			UniqueID: "spacechase0.SpaceCore",
			Required: false,
		}},
	}
	contentFramework := manifest.Manifest{
		Name:     "Content Framework",
		UniqueID: "Example.ContentFramework",
	}
	oldContentPack := manifest.Manifest{
		UniqueID:       "Example.OldContentPack",
		ContentPackFor: "Example.ContentFramework",
	}
	mods := []framework.Mod{
		{
			Key:      "spacecore",
			Enabled:  true,
			Manifest: spaceCore,
		},
		{
			Key:      "old-pack",
			Enabled:  false,
			Manifest: oldPack,
		},
		{
			Key:      "content-framework",
			Enabled:  true,
			Manifest: contentFramework,
		},
		{
			Key:      "old-content-pack",
			Enabled:  false,
			Manifest: oldContentPack,
		},
	}

	got := cleanupHints(mods, nil)
	if len(got) != 2 || got[0].Key != "content-framework" || got[1].Key != "spacecore" {
		t.Fatalf("cleanupHints() = %#v, want both unused frameworks", got)
	}
}
