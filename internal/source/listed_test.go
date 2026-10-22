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
