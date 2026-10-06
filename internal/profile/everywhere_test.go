package profile

import (
	"slices"
	"testing"
)

func everywhereEnv(t *testing.T) (env, Profile, Profile) {
	t.Helper()
	e := newEnv(t)
	a := mustCreate(t, e, "A")
	b := mustCreate(t, e, "B")
	var err error
	m := manifestJSON("me.a")
	e.item(t, "a-1", map[string]string{"A/manifest.json": m})
	e.item(t, "a-2", map[string]string{"A/manifest.json": m})
	src := Source{Kind: KindLocal, Name: "a.zip"}
	if _, err := e.AddEntry("stardew", a.ID, "a-1", src); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", b.ID, "a-1", src); err != nil {
		t.Fatal(err)
	}
	a, err = e.read("stardew", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err = e.read("stardew", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	return e, a, b
}

func TestPreviewEverywhereExcludesPinnedSkippedLocked(t *testing.T) {
	t.Parallel()
	e, a, b := everywhereEnv(t)
	c := mustCreate(t, e, "C")
	if _, err := e.AddEntry("stardew", c.ID, "a-1", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetPinned("stardew", a.ID, "a-1", true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetSkipVersion("stardew", b.ID, "a-1", "2.0.0"); err != nil {
		t.Fatal(err)
	}
	e.Running = func(game, id string) bool { return game == "stardew" && id == c.ID }

	got, err := e.PreviewEverywhere("stardew", "smapi:me.a")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Affected) != 0 {
		t.Fatalf("affected %+v, want none", got.Affected)
	}
	reasons := map[string]string{}
	for _, s := range got.Skipped {
		reasons[s.ProfileID] = s.Reason
	}
	if reasons[a.ID] != skipPinned || reasons[b.ID] != skipVersion || reasons[c.ID] != skipLocked {
		t.Fatalf("skipped %+v", got.Skipped)
	}
}

func TestUpdateEverywhereTwoProfilesOneStoreItem(t *testing.T) {
	t.Parallel()
	e, a, b := everywhereEnv(t)
	got, err := e.UpdateEverywhere("stardew", "smapi:me.a", "a-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Updated) != 2 {
		t.Fatalf("updated %+v", got.Updated)
	}
	ids := []string{got.Updated[0].ProfileID, got.Updated[1].ProfileID}
	if !slices.Contains(ids, a.ID) || !slices.Contains(ids, b.ID) {
		t.Fatalf("updated ids %v", ids)
	}
	change := map[string]string{}
	for _, hit := range got.Updated {
		change[hit.ProfileID] = hit.Change
	}
	for _, id := range []string{a.ID, b.ID} {
		p, err := e.read("stardew", id)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Entries) != 1 || p.Entries[0].Key != "a-2" {
			t.Fatalf("%s entries %+v", id, p.Entries)
		}
		events, err := e.History("stardew", id)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(events, func(ev HistoryEvent) bool { return ev.Kind == historyUpdated }) {
			t.Fatalf("%s history %+v", id, events)
		}
		if change[id] == "" || change[id] != events[0].ID {
			t.Fatalf("%s change %q, newest event %q", id, change[id], events[0].ID)
		}
	}
}

func TestLatestStoreKeyOrdersBySourceVersion(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	pkg := func(key, source, name, version string, files map[string]string) {
		e.item(t, key, files)
		if err := e.items.Describe("stardew", key, source, name, version); err != nil {
			t.Fatal(err)
		}
	}
	plugin := map[string]string{"BepInEx/plugins/x.dll": "x"}
	// Keys sort opposite to versions, so a string comparison of keys picks the wrong one.
	pkg("pkg-c", KindThunderstore, "Ns-Mod", "1.9.0", plugin)
	pkg("pkg-b", KindThunderstore, "Ns-Mod", "1.10.0", plugin)
	pkg("pkg-a", KindThunderstore, "Ns-Mod", "1.2.0", plugin)
	pkg("pkg-z", KindThunderstore, "Ns-Other", "9.0.0", plugin)
	pkg("github-a", KindGitHub, "me/mod", "v2.0.0", plugin)
	pkg("github-b", KindGitHub, "me/mod", "v10.0.0", plugin)
	man := func(v string) map[string]string {
		return map[string]string{"A/manifest.json": `{"Name":"A","Author":"me","Version":"` + v + `","UniqueID":"me.a"}`}
	}
	e.item(t, "local-1", man("1.9.0"))
	e.item(t, "local-2", man("1.10.0"))
	e.item(t, "local-0", man("1.0.0"))
	e.item(t, "nexus-5-10", man("1.0.0"))
	e.item(t, "nexus-5-12", man("1.1.0"))
	e.item(t, "nexus-5-20", map[string]string{"B/manifest.json": manifestJSON("me.b")})
	e.item(t, "nexus-6-30", man("1.0.5"))

	for _, c := range []struct{ old, id, want string }{
		{"pkg-c", "Ns-Mod", "pkg-b"},
		{"github-a", "me/mod", "github-b"},
		{"local-1", "smapi:me.a", "local-2"},
		{"local-1", "local-1", "local-2"},
		{"nexus-5-10", "smapi:me.a", "nexus-5-12"},
	} {
		got, err := e.latestStoreKey("stardew", c.old, c.id)
		if err != nil || got != c.want {
			t.Errorf("latest of %s = %q, %v; want %s", c.old, got, err, c.want)
		}
	}
	for _, old := range []string{"pkg-b", "local-2", "nexus-5-12", "pkg-z"} {
		if got, err := e.latestStoreKey("stardew", old, ""); err == nil {
			t.Errorf("latest of %s = %q, want none", old, got)
		}
	}
}
