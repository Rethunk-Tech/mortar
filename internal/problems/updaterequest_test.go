package problems

import "testing"

func TestUpdateRequestTakesTheUpdatesSource(t *testing.T) {
	t.Parallel()
	ts := Update{Key: "k", Version: "1.1.0", Package: "Ns-Mod"}.Request("lethal-company", "p")
	if ts.Package != "Ns-Mod" || ts.Version != "1.1.0" || ts.ModID != 0 || ts.Game != "lethal-company" || ts.Profile != "p" {
		t.Fatalf("thunderstore = %+v", ts)
	}
	mr := Update{Version: "2.0", Package: "abc", PackageSource: "modrinth", PackageVersion: "v9"}.Request("g", "p")
	if mr.Source != "modrinth" || mr.Version != "v9" {
		t.Fatalf("modrinth = %+v", mr)
	}
	gh := Update{Version: "2.0", GitHubRepo: "o/r", NexusID: 7}.Request("g", "p")
	if gh.Repo != "o/r" || gh.ModID != 0 {
		t.Fatalf("github = %+v", gh)
	}
	nx := Update{Version: "3", NexusID: 7, FileID: 9, ID: "smapi:A", GitHubFallback: "o/r"}.Request("stardew", "p")
	if nx.ModID != 7 || nx.FileID != 9 || nx.FallbackRepo != "o/r" || nx.FallbackID != "smapi:A" || nx.Package != "" {
		t.Fatalf("nexus = %+v", nx)
	}
}
