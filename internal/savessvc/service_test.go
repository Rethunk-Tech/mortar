package savessvc

import (
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestDismissHidesModForThatSaveOnly(t *testing.T) {
	testfs.DataHome(t)
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{settings: store}
	for range 2 {
		if err := s.Dismiss("Farm_1", "smapi:Author.Mod"); err != nil {
			t.Fatal(err)
		}
	}
	used := []mod.ID{"smapi:author.mod", "smapi:other.mod"}
	d := store.Get().Dismissed
	if got := saves.Lacking(used, nil, asIDs(d["Farm_1"])); !reflect.DeepEqual(got, []saves.Lack{{ID: "smapi:other.mod"}}) {
		t.Fatalf("Farm_1 lacks %+v", got)
	}
	if got := saves.Lacking(used, nil, asIDs(d["Farm_2"])); len(got) != 2 {
		t.Fatalf("Farm_2 lacks %+v", got)
	}
	if len(d["Farm_1"]) != 1 {
		t.Fatalf("dismissed twice: %v", d["Farm_1"])
	}
	if err := s.RestoreDismissed("Farm_1", "smapi:Author.Mod"); err != nil {
		t.Fatal(err)
	}
	d = store.Get().Dismissed
	if got := saves.Lacking(used, nil, asIDs(d["Farm_1"])); !reflect.DeepEqual(got, []saves.Lack{{ID: "smapi:author.mod"}, {ID: "smapi:other.mod"}}) {
		t.Fatalf("Farm_1 after restore lacks %+v", got)
	}
}
