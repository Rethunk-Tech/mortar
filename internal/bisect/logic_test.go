package bisect

import (
	"context"
	"errors"
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
	if len(tested) != 3 || len(tested[0]) != 0 {
		t.Fatalf("runs = %v, want every mod on first, then two halves", tested)
	}
}

func TestFindKeepsALibraryOnWhileAnythingThatNeedsItIsOn(t *testing.T) {
	mods := []Mod{{ID: "lib"}}
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		mods = append(mods, Mod{ID: id, Dependencies: []string{"lib"}})
	}
	got, err := find(context.Background(), mods, func(_ context.Context, off []string) (bool, error) {
		if slices.Contains(off, "lib") {
			for _, m := range mods[1:] {
				if !slices.Contains(off, m.ID) {
					t.Fatalf("lib off while %s, which needs it, is on: %v", m.ID, off)
				}
			}
		}
		return !slices.Contains(off, "f"), nil
	}, nil)
	if err != nil || len(got) != 1 || got[0].ID != "f" {
		t.Fatalf("found %v, %v; want f", got, err)
	}
}

func TestFindBlamesNobodyWhenEveryModOnRunsClean(t *testing.T) {
	_, err := find(context.Background(), []Mod{{ID: "a"}, {ID: "b"}}, func(context.Context, []string) (bool, error) {
		return false, nil
	}, nil)
	if !errors.Is(err, ErrNoCrash) {
		t.Fatalf("err = %v, want ErrNoCrash", err)
	}
}
