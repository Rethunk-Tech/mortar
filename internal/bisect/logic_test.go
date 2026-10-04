package bisect

import (
	"context"
	"reflect"
	"slices"
	"testing"
)

func TestDisableClosureIncludesGroupedFrameworksAndDependents(t *testing.T) {
	mods := []Mod{
		{ID: "pack", Group: "pack"},
		{ID: "framework", Group: "pack"},
		{ID: "addon", Dependencies: []string{"pack"}},
		{ID: "unrelated"},
	}

	got := DisableClosure(mods, []string{"pack"})
	want := []string{"addon", "framework", "pack"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("disabled = %#v, want %#v", got, want)
	}
}

func TestFindUsesFakeRunnerToNarrowFailingGroup(t *testing.T) {
	mods := []Mod{
		{ID: "a"},
		{ID: "b"},
		{ID: "framework", Group: "pack"},
		{ID: "pack", Dependencies: []string{"framework"}, Group: "pack"},
		{ID: "e"},
	}
	var tested [][]string
	got, err := find(context.Background(), mods, func(_ context.Context, disabled []string) (bool, error) {
		tested = append(tested, append([]string(nil), disabled...))
		if slices.Contains(disabled, "pack") {
			return false, nil
		}
		return true, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(got))
	for _, mod := range got {
		ids = append(ids, mod.ID)
	}
	if want := []string{"framework", "pack"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("found = %#v, want %#v", ids, want)
	}
	if len(tested) != 2 {
		t.Fatalf("runner calls = %d, want 2", len(tested))
	}
}
