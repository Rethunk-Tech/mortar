package nexus

import (
	"slices"
	"testing"
)

func TestChangelogBetweenFromCannedBody(t *testing.T) {
	all := []Changelog{
		{Version: "2.1.0", Notes: []string{"not in this update"}},
		{Version: "2.0.0", Notes: []string{"breaking save format", "new crops"}},
		{Version: "1.5.0", Notes: []string{"mid"}},
		{Version: "1.0.0", Notes: []string{"first release"}},
	}
	got := ChangelogBetween(all, "1.0.0", "2.0.0")
	var versions []string
	for _, e := range got {
		versions = append(versions, e.Version)
	}
	if want := []string{"2.0.0", "1.5.0"}; !slices.Equal(versions, want) {
		t.Fatalf("versions = %v, want %v (notes=%+v)", versions, want, got)
	}
	if len(got) != 2 || !slices.Equal(got[0].Notes, []string{"breaking save format", "new crops"}) {
		t.Fatalf("2.0.0 notes = %+v", got)
	}
}

func TestChangelogBetweenEmpty(t *testing.T) {
	if got := ChangelogBetween(nil, "1.0.0", "2.0.0"); len(got) != 0 {
		t.Fatalf("empty = %+v", got)
	}
}
