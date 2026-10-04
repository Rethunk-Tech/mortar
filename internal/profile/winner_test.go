package profile

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

func TestSetWinnerRewritesManifestIdempotent(t *testing.T) {
	e := newEnv(t)
	e.item(t, "win", map[string]string{"manifest.json": manifestJSON("Me.Win")})
	e.item(t, "lose", map[string]string{"manifest.json": manifestJSON("Me.Lose")})
	p := mustCreate(t, e, "Farm")
	if _, err := e.AddEntry("stardew", p.ID, "win", Source{Kind: KindLocal, Name: "win.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "lose", Source{Kind: KindLocal, Name: "lose.zip"}); err != nil {
		t.Fatal(err)
	}
	got, err := e.SetWinner("stardew", p.ID, "win", "Me.Lose", true)
	if err != nil {
		t.Fatal(err)
	}
	if !hasLoadAfter(got, "win", "Me.Lose") {
		t.Fatalf("LoadAfter = %+v", got.Entries)
	}
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "win"), "Me.Lose", true)
	again, err := e.SetWinner("stardew", p.ID, "win", "Me.Lose", true)
	if err != nil {
		t.Fatal(err)
	}
	if !hasLoadAfter(again, "win", "Me.Lose") {
		t.Fatalf("second add lost LoadAfter: %+v", again.Entries)
	}
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "win"), "Me.Lose", true)
	cleared, err := e.SetWinner("stardew", p.ID, "win", "Me.Lose", false)
	if err != nil {
		t.Fatal(err)
	}
	if hasLoadAfter(cleared, "win", "Me.Lose") {
		t.Fatalf("undo left LoadAfter: %+v", cleared.Entries)
	}
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "win"), "Me.Lose", false)
	onceMore, err := e.SetWinner("stardew", p.ID, "win", "Me.Lose", false)
	if err != nil {
		t.Fatal(err)
	}
	if hasLoadAfter(onceMore, "win", "Me.Lose") {
		t.Fatalf("second undo restored LoadAfter: %+v", onceMore.Entries)
	}
}

func TestLoadAfterReappliedAfterUpdate(t *testing.T) {
	e := newEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p := mustCreate(t, e, "P")
	e.item(t, "a-1", map[string]string{"A/manifest.json": manifestJSON("me.a")})
	e.item(t, "a-2", map[string]string{"A/manifest.json": manifestJSON("me.a")})
	e.item(t, "b-1", map[string]string{"B/manifest.json": manifestJSON("me.b")})
	if _, err := e.AddEntry("stardew", p.ID, "a-1", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "b-1", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetWinner("stardew", p.ID, "a-1", "me.b", true); err != nil {
		t.Fatal(err)
	}
	got, err := e.UpdateEntry("stardew", p.ID, "a-1", "a-2")
	if err != nil {
		t.Fatal(err)
	}
	if !hasLoadAfter(got, "a-2", "me.b") {
		t.Fatalf("update dropped LoadAfter: %+v", got.Entries)
	}
	assertOptionalDep(t, read(t, filepath.Join(e.mods(p.ID), "a-2", "A", "manifest.json")), "me.b", true)
}

func hasLoadAfter(p Profile, key, loser string) bool {
	for _, e := range p.Entries {
		if e.Key != key {
			continue
		}
		for _, id := range e.LoadAfter {
			if SameID(id, loser) {
				return true
			}
		}
	}
	return false
}

func winnerManifest(t *testing.T, e env, profileID, key string) string {
	t.Helper()
	return read(t, filepath.Join(e.mods(profileID), key, "manifest.json"))
}

func assertOptionalDep(t *testing.T, raw, uniqueID string, want bool) {
	t.Helper()
	m, err := manifest.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	got := false
	for _, d := range m.Dependencies {
		if SameID(d.UniqueID, uniqueID) && !d.Required {
			got = true
			break
		}
	}
	if got != want {
		t.Fatalf("optional %s = %v, want %v; body %s", uniqueID, got, want, strings.TrimSpace(raw))
	}
}

func TestRewriteManifestDepsJSONC(t *testing.T) {
	raw := []byte("\xef\xbb\xbf{\n  // comment\n  \"Name\": \"Win\",\n  \"UniqueID\": \"Me.Win\",\n}\n")
	out, err := rewriteManifestDeps(raw, []string{"Me.Lose"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertOptionalDep(t, string(out), "Me.Lose", true)
}
