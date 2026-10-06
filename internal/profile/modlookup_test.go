package profile

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestEntryEnabledAndFindMod(t *testing.T) {
	t.Parallel()
	p := Profile{Entries: []Entry{
		{Key: "a", Mods: []Component{{ID: "smapi:Au.One"}, {ID: ""}}, Disabled: []mod.ID{"smapi: au.one "}},
		{Key: "b", Mods: []Component{{ID: "smapi:Au.Two"}}},
	}}
	a := p.Entries[0]
	if a.Enabled("smapi:AU.ONE") {
		t.Error("a disabled ID matches regardless of case and stray spaces")
	}
	if !a.Enabled("") {
		t.Error("a mod without an ID cannot be switched off on its own")
	}
	if !p.Entries[1].Enabled("smapi:Au.Two") {
		t.Error("a mod absent from Disabled is on")
	}
	if e, m, ok := p.FindMod("", "smapi:au.two"); !ok || e.Key != "b" || m.ID != "smapi:Au.Two" {
		t.Errorf("FindMod any entry = %v %v %v", e.Key, m.ID, ok)
	}
	if _, _, ok := p.FindMod("a", "smapi:Au.Two"); ok {
		t.Error("FindMod with a key must stay in that entry")
	}
}

func TestEntryJSONKeepsComponentIDs(t *testing.T) {
	t.Parallel()
	in := Entry{
		Key: "a", Mods: []Component{{ID: "smapi:Au.One", Needs: []mod.ID{"smapi:Au.Two"}, ContentPackFor: "smapi:Au.Fw", LoadAfter: []mod.ID{"smapi:Au.Two"}}},
		Disabled: []mod.ID{"smapi:Au.One"},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"id":"smapi:Au.One"`, `"needs":["smapi:Au.Two"]`, `"disabled":["smapi:Au.One"]`, `"loadAfter":["smapi:Au.Two"]`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("%s missing from %s", want, raw)
		}
	}
	var out Entry
	if err := json.Unmarshal(raw, &out); err != nil || !reflect.DeepEqual(in, out) {
		t.Errorf("round trip = %+v, %v", out, err)
	}
}
