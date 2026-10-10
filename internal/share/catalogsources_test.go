package share

import (
	"errors"
	"testing"
)

// A link's own sourceKeys are the sender's claim: a game that has no CurseForge, Nexus or Thunderstore source in the
// receiver's catalog does not gain one by the link saying so.
func TestCheckSharedTakesSourcesFromTheCatalogNotTheLink(t *testing.T) {
	t.Parallel()
	claim := map[string]string{"curseforge": "1", "nexus": "n", "thunderstore": "t"}
	for _, ref := range []Ref{{CurseForge: 7, FileID: 9}, {ModID: 5, FileID: 9}} {
		if err := checkShared(Shared{Game: "riskofrain2", Name: "x", SourceKeys: claim, Entries: []Ref{ref}}); !errors.Is(err, ErrMalformed) {
			t.Errorf("%+v for a game without that source: %v", ref, err)
		}
	}
	if err := checkShared(Shared{Game: "lethal-company", Name: "x", SourceKeys: claim, Entries: []Ref{{Package: "Ns-Mod", Version: "1.0.0"}}}); err != nil {
		t.Errorf("a source the catalog lists: %v", err)
	}
}
