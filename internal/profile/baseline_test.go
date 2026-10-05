package profile

import (
	"reflect"
	"testing"
)

func TestRevertToBaselineRestoresEntriesAndEnabledStates(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	p := addFarmMod(t, e)
	before, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := e.Baseline("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, before.Entries[0].Key, "smapi:Me.A", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-b", Source{Kind: KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	got, err := e.Revert("stardew", p.ID, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Entries, before.Entries) {
		t.Fatalf("entries = %+v, want %+v", got.Entries, before.Entries)
	}
}
