package sharesvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestApplySharedGroups(t *testing.T) {
	s, _ := newService(t, false)
	items, _ := testenv.Stores(t)
	p := mineProfile(t, s)
	keyA, keyB := storeMod(t, items, "Me.A"), storeMod(t, items, "Me.B")
	for _, k := range []string{keyA, keyB} {
		if _, err := s.d.Profiles.AddEntry("stardew", p.ID, k, profile.Source{Kind: profile.KindLocal, Name: k}); err != nil {
			t.Fatal(err)
		}
	}
	groups := func(t *testing.T) map[string][]string {
		t.Helper()
		cur, err := s.d.Profiles.Get("stardew", p.ID)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string][]string{}
		for _, g := range cur.Groups {
			out[g.Name] = g.Keys
		}
		return out
	}

	if err := s.applySharedGroups("stardew", p.ID, nil); err != nil {
		t.Fatalf("no groups: %v", err)
	}
	if err := s.applySharedGroups("stardew", p.ID, []share.FileGroup{
		{Name: "Gone", Refs: []share.Ref{{Local: "absent"}}},
		{Name: "Pair", Refs: []share.Ref{{Local: keyA}, {Local: keyB}}},
	}); err != nil {
		t.Fatal(err)
	}
	got := groups(t)
	if keys, ok := got["Gone"]; !ok || len(keys) != 0 {
		t.Errorf("a group resolving to nothing is created empty: %v", got)
	}
	if len(got["Pair"]) != 2 {
		t.Errorf("Pair keys = %v, want both entries", got["Pair"])
	}

	if err := s.applySharedGroups("stardew", p.ID, []share.FileGroup{{Name: "gone"}}); err != nil {
		t.Errorf("a failed create is fine when the group exists under another case: %v", err)
	}
	if err := s.applySharedGroups("stardew", p.ID, []share.FileGroup{{Name: " "}}); err == nil {
		t.Error("a create failing with no such group must return the error")
	}
	if err := s.applySharedGroups("stardew", "missing", []share.FileGroup{{Name: "X"}}); err == nil {
		t.Error("an unknown profile must fail")
	}
}

func TestHasGroupIgnoresCase(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Groups: []profile.Group{{Name: "Core"}}}
	if !hasGroup(p, "CORE") || hasGroup(p, "Other") {
		t.Fatal("hasGroup must match names without case and nothing else")
	}
}
