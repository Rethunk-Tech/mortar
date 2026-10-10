package sharesvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/share"
)

// The recipient's own catalog decides which sources a game has, not the link: Lethal Company lists no CurseForge
// source, so a CurseForge file in its link cannot be downloaded however the link's sourceKeys read.
func TestACurseForgeFileForAGameWithNoCurseForgeSourceIsNotOfferedForDownload(t *testing.T) {
	t.Parallel()
	r := &resolver{game: "lethal-company", curseforgeUnavailable: func() string { return "" }, storedKeys: map[string]bool{}}
	if m := r.curseforge(t.Context(), share.Ref{CurseForge: 7, FileID: 9}); m.State == StateDownload {
		t.Fatalf("offered for download: %+v", m)
	}
}

func TestEverySourceComesFromTheReceiversCatalogForTheLinksGame(t *testing.T) {
	t.Parallel()
	r := &resolver{game: "repo", storedKeys: map[string]bool{}, curseforgeUnavailable: func() string { return "" }}
	mods, _ := r.resolve(t.Context(), []share.Ref{
		{GitHub: "o/r@v1/a.zip"},
		{Itch: "someone/cool-mod"},
		{CurseForge: 7, FileID: 9},
		{Package: "Ns-Mod", Version: "1.0.0"},
	})
	for _, m := range mods[:3] {
		if m.State != StateUnavailable || m.Reason != ReasonNoSource {
			t.Errorf("%s from a source the game's catalog lacks: %+v", m.Site, m)
		}
	}
	if mods[3].Reason == ReasonNoSource {
		t.Errorf("repo lists thunderstore: %+v", mods[3])
	}
}
