package profile

import (
	"reflect"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestHistoryDiffAddedRemovedVersionEnabled(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"B/manifest.json": manifestJSON("Me.B")})
	e.item(t, "local-a2", map[string]string{"A/manifest.json": `{"Name":"Alpha","Author":"me","Version":"2.0.0","UniqueID":"Me.A"}`})
	p := addFarmMod(t, e)
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	events, err := e.History("stardew", p.ID)
	if err != nil || len(events) < 2 {
		t.Fatalf("history: %+v %v", events, err)
	}
	first, withB := events[len(events)-1], events[0]
	got, err := e.HistoryDiff("stardew", p.ID, first.ID, withB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Added) != 1 {
		t.Fatalf("added = %+v", got.Added)
	}
	if _, err := e.RemoveEntry("stardew", p.ID, "local-b"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.UpdateEntry("stardew", p.ID, "local-a", "local-a2"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "local-a2", "smapi:Me.A", false); err != nil {
		t.Fatal(err)
	}
	all, err := e.History("stardew", p.ID)
	if err != nil || len(all) < 2 {
		t.Fatalf("later history: %+v %v", all, err)
	}
	diff, err := e.HistoryDiff("stardew", p.ID, withB.ID, all[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Versions) != 1 || diff.Versions[0].Old == diff.Versions[0].New {
		t.Fatalf("versions = %+v", diff.Versions)
	}
	if len(diff.Enabled) != 1 || diff.Enabled[0].New {
		t.Fatalf("enabled = %+v", diff.Enabled)
	}
	if len(diff.Removed) != 1 {
		t.Fatalf("removed = %+v", diff.Removed)
	}
}

func TestHistoryDiffConfigFiles(t *testing.T) {
	t.Parallel()
	before := []Entry{{Key: "a", Mods: []Component{{ID: "smapi:Me.A", Name: "Alpha", Version: "1"}}}}
	after := []Entry{{Key: "a", Mods: []Component{{ID: "smapi:Me.A", Name: "Alpha", Version: "1"}}}}
	cfgA := map[string]map[string][]byte{"smapi:me.a": {"config.json": []byte(`{"x":1}`)}}
	cfgB := map[string]map[string][]byte{"smapi:me.a": {"config.json": []byte(`{"x":2}`)}}
	got := DiffSnapshots("a", "b", before, after, cfgA, cfgB)
	if len(got.Configs) != 1 || !reflect.DeepEqual(got.Configs[0].Files, []string{"config.json"}) {
		t.Fatalf("configs = %+v", got.Configs)
	}
	if len(got.Items) != 1 || got.Items[0].Kind != diffConfig {
		t.Fatalf("items = %+v", got.Items)
	}
}

func TestRevertHistoryItemKinds(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("Me.A"), "A/config.json": `{"x":1}`})
	e.item(t, "local-b", map[string]string{"B/manifest.json": manifestJSON("Me.B")})
	e.item(t, "local-a2", map[string]string{"A/manifest.json": `{"Name":"Alpha","Author":"me","Version":"2.0.0","UniqueID":"Me.A"}`, "A/config.json": `{"x":1}`})
	p := addFarmMod(t, e)
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	added, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.RevertHistoryItem("stardew", p.ID, added[0].ID, "Me.B"); err != nil {
		t.Fatal(err)
	}
	cur, err := e.read("stardew", p.ID)
	if err != nil || len(cur.Entries) != 1 {
		t.Fatalf("after revert add: %+v %v", cur.Entries, err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "local-a", "smapi:Me.A", false); err != nil {
		t.Fatal(err)
	}
	evs, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.RevertHistoryItem("stardew", p.ID, evs[0].ID, "Me.A"); err != nil {
		t.Fatal(err)
	}
	cur, err = e.read("stardew", p.ID)
	if err != nil || len(cur.Entries) != 1 || hasID(cur.Entries[0].Disabled, "smapi:Me.A") {
		t.Fatalf("toggle revert: %+v %v", cur.Entries, err)
	}
	if _, err := e.UpdateEntry("stardew", p.ID, "local-a", "local-a2"); err != nil {
		t.Fatal(err)
	}
	evs, err = e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.RevertHistoryItem("stardew", p.ID, evs[0].ID, "Me.A"); err != nil {
		t.Fatal(err)
	}
	cur, err = e.read("stardew", p.ID)
	if err != nil || len(cur.Entries) != 1 || cur.Entries[0].Key != "local-a" {
		t.Fatalf("update revert: %+v %v", cur.Entries, err)
	}
}

func TestChangesSinceLastRun(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"B/manifest.json": manifestJSON("Me.B")})
	p := addFarmMod(t, e)
	mid := time.Now().UTC()
	// Event times are whole seconds, so the next event must land in a later second than mid.
	time.Sleep(time.Until(mid.Truncate(time.Second).Add(time.Second)))
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	got, err := e.ChangesSince("stardew", p.ID, mid)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Added) != 1 {
		t.Fatalf("changes since run = %+v", got)
	}
	none, err := e.ChangesSince("stardew", p.ID, time.Time{})
	if err != nil || len(none.Items) != 0 {
		t.Fatalf("no run = %+v %v", none, err)
	}
}

func TestKnownGoodMarkAndRestore(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"B/manifest.json": manifestJSON("Me.B")})
	p := addFarmMod(t, e)
	if _, err := e.RestoreKnownGood("stardew", p.ID); usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("restore with nothing marked: %v, want a not-found user error", err)
	}
	if _, err := e.MarkKnownGood("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	got, err := e.RestoreKnownGood("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Key != "local-a" {
		t.Fatalf("restored = %+v", got.Entries)
	}
	marks, err := e.KnownGood("stardew", p.ID)
	if err != nil || len(marks) == 0 {
		t.Fatalf("markers = %+v %v", marks, err)
	}
}
