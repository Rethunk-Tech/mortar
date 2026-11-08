package profile

import (
	"testing"

	gh "github.com/Rethunk-Tech/go-github"
)

// Current corrects an entry's source to the store key that names the installed file: a Nexus source whose file id
// lags the key follows the key, and a GitHub source recorded before the asset was stored recovers tag and asset.
func TestCurrentFollowsNexusAndGitHubKeys(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	e.item(t, "nexus-7-2", map[string]string{"manifest.json": manifestJSON("X.N")})
	ghKey := gh.Key("me", "mod", "v2.0.0", "mod.zip")
	e.item(t, ghKey, map[string]string{"manifest.json": manifestJSON("X.G")})
	if err := e.items.Describe("stardew", ghKey, KindGitHub, "mod", "v2.0.0", "mod.zip"); err != nil {
		t.Fatal(err)
	}
	p, err := e.AddEntry("stardew", p.ID, "nexus-7-2", Source{Kind: KindNexus, ModID: 7, FileID: 1, Digest: "sha256:x"})
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.AddEntry("stardew", p.ID, ghKey, Source{Kind: KindGitHub, Repo: "me/mod"})
	if err != nil {
		t.Fatal(err)
	}
	cur := e.Current("stardew", p)
	if len(cur.Entries) != 2 {
		t.Fatalf("entries = %+v", cur.Entries)
	}
	nexus, github := cur.Entries[0].Source, cur.Entries[1].Source
	if nexus.ModID != 7 || nexus.FileID != 2 || nexus.Digest != "" {
		t.Fatalf("nexus source = %+v, want mod 7 file 2 with the stale digest cleared", nexus)
	}
	if github.Tag != "v2.0.0" || github.Asset != "mod.zip" || github.Digest != "" {
		t.Fatalf("github source = %+v, want tag v2.0.0 and asset mod.zip", github)
	}
}
