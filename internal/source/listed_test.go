package source

import (
	"strconv"
	"testing"
)

func TestListedStopsGrowingPastTheCap(t *testing.T) {
	t.Parallel()
	var s ListedStore
	first := VersionRef{ID: "1", Version: "v0"}
	var last VersionRef
	for i := range listedCap + 50 {
		last = VersionRef{ID: strconv.Itoa(i + 1), Version: "v" + strconv.Itoa(i)}
		s.Put(last, []string{"dep"})
	}
	if s.len() != listedCap {
		t.Fatalf("len = %d, want %d", s.len(), listedCap)
	}
	if _, ok := s.Get(first); ok {
		t.Fatal("oldest version still listed")
	}
	if _, ok := s.Get(last); !ok {
		t.Fatal("newest version missing")
	}
}

func TestListedGetAllSkipsUnknownRefs(t *testing.T) {
	var s ListedStore
	a, b := VersionRef{ID: "a", Version: "1"}, VersionRef{ID: "b", Version: "1"}
	s.Put(a, []string{"x"})
	if got := s.GetAll([]VersionRef{a, b}); len(got) != 1 || got[a][0] != "x" {
		t.Fatalf("GetAll = %v", got)
	}
}
