package profile

import (
	"slices"
	"testing"
)

func everywhereEnv(t *testing.T) (env, Profile, Profile) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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

	got, err := e.PreviewEverywhere("stardew", "me.a")
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
	e, a, b := everywhereEnv(t)
	got, err := e.UpdateEverywhere("stardew", "me.a", "a-2")
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
	}
}
