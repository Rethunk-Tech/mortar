package savessvc

import (
	"reflect"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/saves"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

func TestDismissHidesModForThatSaveOnly(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{settings: store}
	for range 2 {
		if err := s.Dismiss("Farm_1", "Author.Mod"); err != nil {
			t.Fatal(err)
		}
	}
	used := []string{"author.mod", "other.mod"}
	d := store.Get().Dismissed
	if got := saves.Lacking(used, nil, d["Farm_1"]); !reflect.DeepEqual(got, []saves.Lack{{UniqueID: "other.mod"}}) {
		t.Fatalf("Farm_1 lacks %+v", got)
	}
	if got := saves.Lacking(used, nil, d["Farm_2"]); len(got) != 2 {
		t.Fatalf("Farm_2 lacks %+v", got)
	}
	if len(d["Farm_1"]) != 1 {
		t.Fatalf("dismissed twice: %v", d["Farm_1"])
	}
	if err := s.RestoreDismissed("Farm_1", "Author.Mod"); err != nil {
		t.Fatal(err)
	}
	d = store.Get().Dismissed
	if got := saves.Lacking(used, nil, d["Farm_1"]); !reflect.DeepEqual(got, []saves.Lack{{UniqueID: "author.mod"}, {UniqueID: "other.mod"}}) {
		t.Fatalf("Farm_1 after restore lacks %+v", got)
	}
}
