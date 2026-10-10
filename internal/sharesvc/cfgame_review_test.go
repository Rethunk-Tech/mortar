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
