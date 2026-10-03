package nexus

import (
	"slices"
	"testing"
)

func TestChangelogBetweenFromCannedBody(t *testing.T) {
	raw := []byte(`{
		"1.0.0": ["first release"],
		"1.5.0": ["mid"],
		"2.0.0": ["breaking save format", "new crops"],
		"2.1.0": ["not in this update"]
	}`)
	all, err := parseChangelogs(raw, 100)
	if err != nil {
		t.Fatal(err)
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

func TestChangelogBetweenEmptyBody(t *testing.T) {
	all, err := parseChangelogs([]byte("[]"), 10)
	if err != nil || len(ChangelogBetween(all, "1.0.0", "2.0.0")) != 0 {
		t.Fatalf("empty = %+v, %v", all, err)
	}
}
