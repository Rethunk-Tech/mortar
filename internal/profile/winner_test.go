package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

func TestSetWinnerRewritesManifestIdempotent(t *testing.T) {
	t.Parallel()
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
	got, err := e.SetWinner("stardew", p.ID, "win", "smapi:Me.Win", "smapi:Me.Lose", true)
	if err != nil {
		t.Fatal(err)
	}
	if !hasLoadAfter(got, "win", "Me.Lose") {
		t.Fatalf("LoadAfter = %+v", got.Entries)
	}
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "win"), "Me.Lose", true)
	again, err := e.SetWinner("stardew", p.ID, "win", "smapi:Me.Win", "smapi:Me.Lose", true)
	if err != nil {
		t.Fatal(err)
	}
	if !hasLoadAfter(again, "win", "Me.Lose") {
		t.Fatalf("second add lost LoadAfter: %+v", again.Entries)
	}
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "win"), "Me.Lose", true)
	cleared, err := e.SetWinner("stardew", p.ID, "win", "smapi:Me.Win", "smapi:Me.Lose", false)
	if err != nil {
		t.Fatal(err)
	}
	if hasLoadAfter(cleared, "win", "Me.Lose") {
		t.Fatalf("undo left LoadAfter: %+v", cleared.Entries)
	}
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "win"), "Me.Lose", false)
	onceMore, err := e.SetWinner("stardew", p.ID, "win", "smapi:Me.Win", "smapi:Me.Lose", false)
	if err != nil {
		t.Fatal(err)
	}
	if hasLoadAfter(onceMore, "win", "Me.Lose") {
		t.Fatalf("second undo restored LoadAfter: %+v", onceMore.Entries)
	}
}

func TestLoadAfterReappliedAfterUpdate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
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
	if _, err := e.SetWinner("stardew", p.ID, "a-1", "smapi:me.a", "smapi:me.b", true); err != nil {
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
		for _, c := range e.Mods {
			if slices.ContainsFunc(c.LoadAfter, func(id mod.ID) bool { return mod.Equal(id, mod.SMAPI(loser)) }) {
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
		if d.ModID() == mod.SMAPI(uniqueID) && !d.Required {
			got = true
			break
		}
	}
	if got != want {
		t.Fatalf("optional %s = %v, want %v; body %s", uniqueID, got, want, strings.TrimSpace(raw))
	}
}

func TestRewriteManifestDepsJSONC(t *testing.T) {
	t.Parallel()
	raw := []byte("\xef\xbb\xbf{\n  // comment\n  \"Name\": \"Win\",\n  \"UniqueID\": \"Me.Win\",\n}\n")
	out, err := manifest.RewriteDependencies(raw, []mod.ID{"smapi:Me.Lose"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertOptionalDep(t, string(out), "Me.Lose", true)
}

func TestSetWinnerNeverWritesASelfDependencyOrCycle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "pack", map[string]string{
		"Main/manifest.json": manifestJSON("Me.Main"),
		"Npc/manifest.json":  `{"Name":"Npc","Author":"me","Version":"1.0.0","UniqueID":"Me.Npc","Dependencies":[{"UniqueID":"Me.Main"}]}`,
	})
	e.item(t, "other", map[string]string{"manifest.json": manifestJSON("Me.Other")})
	p := mustCreate(t, e, "Farm")
	for _, k := range []string{"pack", "other"} {
		if _, err := e.AddEntry("stardew", p.ID, k, Source{Kind: KindLocal, Name: k + ".zip"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, loser := range []string{"smapi:Me.Npc", "smapi:Me.Other"} {
		if _, err := e.SetWinner("stardew", p.ID, "pack", "smapi:Me.Main", mod.ID(loser), true); err != nil {
			t.Fatal(err)
		}
	}
	main := winnerManifest(t, e, p.ID, "pack/Main")
	npc := winnerManifest(t, e, p.ID, "pack/Npc")
	assertOptionalDep(t, main, "Me.Npc", false)
	assertOptionalDep(t, npc, "Me.Npc", false)
	assertOptionalDep(t, main, "Me.Other", true)
}

func TestSetWinnerRefusesALoserThatNeedsTheWinner(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "win", map[string]string{"manifest.json": manifestJSON("Me.Win")})
	e.item(t, "lose", map[string]string{"manifest.json": `{"Name":"Lose","Author":"me","Version":"1.0.0","UniqueID":"Me.Lose","Dependencies":[{"UniqueID":"Me.Win","IsRequired":false}]}`})
	p := mustCreate(t, e, "Farm")
	for _, k := range []string{"win", "lose"} {
		if _, err := e.AddEntry("stardew", p.ID, k, Source{Kind: KindLocal, Name: k + ".zip"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.SetWinner("stardew", p.ID, "win", "smapi:Me.Win", "smapi:Me.Lose", true); err == nil {
		t.Fatal("made a mod win over one that needs it")
	}
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "win"), "Me.Lose", false)
}

func TestUndoWinKeepsTheAuthorsOwnDependency(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "win", map[string]string{"manifest.json": `{"Name":"Win","Author":"me","Version":"1.0.0","UniqueID":"Me.Win","Dependencies":[{"UniqueID":"Me.Lose","IsRequired":false}]}`})
	e.item(t, "lose", map[string]string{"manifest.json": manifestJSON("Me.Lose")})
	p := mustCreate(t, e, "Farm")
	for _, k := range []string{"win", "lose"} {
		if _, err := e.AddEntry("stardew", p.ID, k, Source{Kind: KindLocal, Name: k + ".zip"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, on := range []bool{true, false} {
		if _, err := e.SetWinner("stardew", p.ID, "win", "smapi:Me.Win", "smapi:Me.Lose", on); err != nil {
			t.Fatal(err)
		}
	}
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "win"), "Me.Lose", true)
}

func twoPackDownload(t *testing.T) (env, Profile) {
	t.Helper()
	e := newEnv(t)
	e.item(t, "pack", map[string]string{
		"Cape/manifest.json":    manifestJSON("Me.Cape"),
		"Annetta/manifest.json": manifestJSON("Me.Annetta"),
	})
	e.item(t, "lose", map[string]string{"manifest.json": manifestJSON("Me.Lose")})
	p := mustCreate(t, e, "Farm")
	for _, k := range []string{"pack", "lose"} {
		if _, err := e.AddEntry("stardew", p.ID, k, Source{Kind: KindLocal, Name: k + ".zip"}); err != nil {
			t.Fatal(err)
		}
	}
	return e, p
}

func TestSetWinnerOrdersOnlyTheWinningPack(t *testing.T) {
	t.Parallel()
	e, p := twoPackDownload(t)
	got, err := e.SetWinner("stardew", p.ID, "pack", "smapi:Me.Cape", "smapi:Me.Lose", true)
	if err != nil {
		t.Fatal(err)
	}
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "pack/Cape"), "Me.Lose", true)
	assertOptionalDep(t, winnerManifest(t, e, p.ID, "pack/Annetta"), "Me.Lose", false)
	for _, c := range got.Entries[entryIndex(got.Entries, "pack")].Mods {
		if want := mod.Equal(c.ID, "smapi:Me.Cape"); slices.Contains(c.LoadAfter, "smapi:Me.Lose") != want {
			t.Fatalf("%s LoadAfter = %v", c.ID, c.LoadAfter)
		}
	}
	if _, err := e.SetWinner("stardew", p.ID, "pack", "smapi:Me.Gone", "smapi:Me.Lose", true); err == nil {
		t.Fatal("won with a pack the download does not hold")
	}
}

func TestEntryLoadAfterMovesToThePacksThatListTheLoser(t *testing.T) {
	t.Parallel()
	e, p := twoPackDownload(t)
	if _, err := e.SetWinner("stardew", p.ID, "pack", "smapi:Me.Cape", "smapi:Me.Lose", true); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(filepath.Dir(e.mods(p.ID)), fileName)
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(read(t, path)), &doc); err != nil {
		t.Fatal(err)
	}
	var entries []Entry
	if err := json.Unmarshal(doc["entries"], &entries); err != nil {
		t.Fatal(err)
	}
	type legacyEntry struct {
		Entry
		LoadAfter []mod.ID `json:"loadAfter,omitempty"`
	}
	legacy := make([]legacyEntry, len(entries))
	for i, en := range entries {
		for ci := range en.Mods {
			en.Mods[ci].LoadAfter = nil
		}
		legacy[i].Entry = en
		if en.Key == "pack" {
			legacy[i].LoadAfter = []mod.ID{"smapi:Me.Lose", "smapi:Me.Nowhere"}
		}
	}
	var err error
	if doc["entries"], err = json.Marshal(legacy); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := e.load("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got.Entries[entryIndex(got.Entries, "pack")].Mods {
		want := []mod.ID(nil)
		if mod.Equal(c.ID, "smapi:Me.Cape") {
			want = []mod.ID{"smapi:Me.Lose"}
		}
		if !slices.Equal(c.LoadAfter, want) {
			t.Fatalf("%s LoadAfter = %v, want %v", c.ID, c.LoadAfter, want)
		}
	}
}
