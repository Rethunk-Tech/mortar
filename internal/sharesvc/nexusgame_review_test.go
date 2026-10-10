package sharesvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

// A Nexus file in a link for a game whose catalog lists no Nexus source is unavailable, whatever the link's own
// sourceKeys claim; the same file for a game that lists Nexus resolves as usual.
func TestANexusFileForAGameWithNoNexusSourceIsUnavailable(t *testing.T) {
	t.Parallel()
	files := map[int][]nexus.File{100: {nf(1, "1.0", "MAIN", true)}}
	ref := []share.Ref{{ModID: 100, FileID: 1}}

	none := testResolver(files, nil)
	none.game = "repo"
	if mods, _ := none.resolve(t.Context(), ref); mods[0].State != StateUnavailable || mods[0].Reason != ReasonNoSource {
		t.Fatalf("repo lists no Nexus source: %+v", mods[0])
	}
	has := testResolver(files, nil)
	has.game = "stardew"
	if mods, _ := has.resolve(t.Context(), ref); mods[0].Reason == ReasonNoSource {
		t.Fatalf("stardew lists Nexus: %+v", mods[0])
	}
}
