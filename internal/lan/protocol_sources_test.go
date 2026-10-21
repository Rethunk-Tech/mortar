package lan

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestTransferItemsCarryCurseForgeFilesAndSkipItchPages(t *testing.T) {
	t.Parallel()
	items := transferItems(share.Shared{Entries: []share.Ref{
		{CurseForge: 7, FileID: 9},
		{CurseForge: 7, FileID: 9},
		{Itch: "someone/cool-mod"},
	}})
	if len(items) != 1 {
		t.Fatalf("items = %+v", items)
	}
	if items[0].Key != store.PackageKey("curseforge:7", "9") || items[0].Source != profile.KindCurseForge || items[0].Package != "7" {
		t.Fatalf("item = %+v", items[0])
	}
}

func TestTransferItemsKeyParadoxModsByTheirVersion(t *testing.T) {
	t.Parallel()
	items := transferItems(share.Shared{Entries: []share.Ref{
		{Paradox: "74604", FileID: 40, Version: "1.8.0"},
		{Paradox: "74604", FileID: 40, Version: "1.8.0"},
	}})
	if len(items) != 1 {
		t.Fatalf("items = %+v", items)
	}
	if items[0].Key != store.PackageKey("paradox:74604", "1.8.0") || items[0].Source != profile.KindParadox || items[0].Package != "74604" || items[0].Version != "1.8.0" {
		t.Fatalf("item = %+v", items[0])
	}
}
