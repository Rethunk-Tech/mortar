package profile

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSetGroupEnabledOneHistoryEvent(t *testing.T) {
	t.Parallel()
	e := envWithFolderAB(t)
	p := abProfile(t, e)
	var err error
	ka, kb := p.Entries[0].Key, p.Entries[1].Key
	if p, err = e.CreateGroup("stardew", p.ID, "Core"); err != nil {
		t.Fatal(err)
	}
	if p, err = e.AddToGroup("stardew", p.ID, "Core", ka); err != nil {
		t.Fatal(err)
	}
	if p, err = e.AddToGroup("stardew", p.ID, "Core", kb); err != nil {
		t.Fatal(err)
	}
	before, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.SetGroupEnabled("stardew", p.ID, "Core", false)
	if err != nil {
		t.Fatal(err)
	}
	after, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("history events = %d, want %d", len(after), len(before)+1)
	}
	for _, en := range p.Entries {
		if en.Key == ka || en.Key == kb {
			if len(en.Disabled) == 0 {
				t.Fatalf("entry %s still enabled", en.Key)
			}
		}
	}
}

func TestRemoveEntryDropsGroupKeys(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"A/manifest.json": manifestJSON("Me.A")})
	p := mustCreate(t, e, "P")
	var err error
	if p, err = e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	key := p.Entries[0].Key
	if p, err = e.AddToGroup("stardew", p.ID, "Core", key); err != nil {
		t.Fatal(err)
	}
	p, err = e.RemoveEntry("stardew", p.ID, key)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(p.Groups, func(g Group) bool { return slices.Contains(g.Keys, key) }) {
		t.Fatalf("groups still name %s: %+v", key, p.Groups)
	}
}

func TestFreeGroupNameCountsUpAndStaysWithinTheLimit(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("g", maxGroupName)
	p := Profile{Groups: []Group{{Name: "Core"}, {Name: "core (2)"}, {Name: long}}}
	if got := p.freeGroupName("Fresh"); got != "Fresh" {
		t.Fatalf("free name = %q", got)
	}
	if got := p.freeGroupName("CORE"); got != "CORE (3)" {
		t.Fatalf("taken name = %q", got)
	}
	if got := p.freeGroupName(long); utf8.RuneCountInString(got) != maxGroupName || !strings.HasSuffix(got, " (2)") {
		t.Fatalf("long name = %q", got)
	}
}

func TestNestedGroupsToggleRenameAndDeleteTogether(t *testing.T) {
	t.Parallel()
	e := envWithFolderAB(t)
	p := abProfile(t, e)
	var err error
	ka, kb := p.Entries[0].Key, p.Entries[1].Key
	// Visuals has no group of its own: it exists as the parent of two.
	if _, err = e.AddToGroup("stardew", p.ID, "Visuals/Hats", ka); err != nil {
		t.Fatal(err)
	}
	if p, err = e.AddToGroup("stardew", p.ID, "Visuals/Skins", kb); err != nil {
		t.Fatal(err)
	}
	if _, err = e.CreateGroup("stardew", p.ID, "Visuals/"); err == nil {
		t.Fatal("an empty level must be refused")
	}
	if p, err = e.SetGroupEnabled("stardew", p.ID, "Visuals", false); err != nil {
		t.Fatal(err)
	}
	for _, en := range p.Entries {
		if len(en.Disabled) == 0 {
			t.Fatalf("entry %s still enabled after its parent group went off", en.Key)
		}
	}
	if p, err = e.RenameGroup("stardew", p.ID, "Visuals", "Looks/Cosmetic"); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, g := range p.Groups {
		names = append(names, g.Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"Looks/Cosmetic/Hats", "Looks/Cosmetic/Skins"}) {
		t.Fatalf("renamed tree = %v", names)
	}
	if _, err = e.RenameGroup("stardew", p.ID, "Looks", "Looks/Inner"); err == nil {
		t.Fatal("a group must not move inside itself")
	}
	if p, err = e.DeleteGroup("stardew", p.ID, "Looks"); err != nil || len(p.Groups) != 0 {
		t.Fatalf("delete tree: %+v %v", p.Groups, err)
	}
}
