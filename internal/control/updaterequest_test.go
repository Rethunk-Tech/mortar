package control

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/problems"
)

func TestUpdateRequestTakesTheUpdatesSource(t *testing.T) {
	t.Parallel()
	ts := updateRequest("lethal-company", "p", problems.Update{Key: "k", Version: "1.1.0", Package: "Ns-Mod"})
	if ts.Package != "Ns-Mod" || ts.Version != "1.1.0" || ts.ModID != 0 || ts.Latest {
		t.Fatalf("thunderstore = %+v", ts)
	}
	mr := updateRequest("g", "p", problems.Update{Version: "2.0", Package: "abc", PackageSource: "modrinth", PackageVersion: "v9"})
	if mr.Source != "modrinth" || mr.Version != "v9" {
		t.Fatalf("modrinth = %+v", mr)
	}
	nx := updateRequest("stardew", "p", problems.Update{Version: "3", NexusID: 7, FileID: 9, ID: "smapi:A", GitHubFallback: "o/r"})
	if nx.ModID != 7 || nx.FileID != 9 || !nx.Latest || nx.FallbackRepo != "o/r" || nx.Package != "" {
		t.Fatalf("nexus = %+v", nx)
	}
}
