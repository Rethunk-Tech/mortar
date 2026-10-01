package sharesvc

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/migrate"
)

func TestExternalLocalModsKeepProfileStateForImport(t *testing.T) {
	mods := externalLocalMods([]migrate.ModPreview{
		{UniqueID: "Example.Mod", Name: "Example Mod", Version: "1.2.3", Enabled: false, SourcePath: "/mods/example"},
	})
	if len(mods) != 1 {
		t.Fatalf("mods = %#v", mods)
	}
	if mods[0].Site != SiteLocal || mods[0].State != StateDownload || mods[0].Key != "external:0" {
		t.Fatalf("mod = %#v", mods[0])
	}
	if len(mods[0].UniqueIDs) != 1 || mods[0].UniqueIDs[0] != "Example.Mod" {
		t.Fatalf("unique IDs = %#v", mods[0].UniqueIDs)
	}
}
