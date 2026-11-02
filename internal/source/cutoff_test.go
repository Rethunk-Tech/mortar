package source

import (
	"context"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/github"
)

func TestWithin(t *testing.T) {
	type rel struct {
		v string
		t string
	}
	date := func(r rel) (time.Time, bool) {
		x, err := time.Parse(time.RFC3339, r.t)
		return x, err == nil
	}
	all := []rel{{"3", "2024-03-01T10:00:00Z"}, {"2", "2024-02-10T23:59:00Z"}, {"1.5", ""}, {"1", "2024-01-01T00:00:00Z"}}
	for _, c := range []struct {
		name, cutoff string
		want         []string
	}{
		{"none", "", []string{"3", "2", "1.5", "1"}},
		{"on the day", "2024-02-10", []string{"2", "1.5", "1"}},
		{"day before", "2024-02-09", []string{"1.5", "1"}},
		{"before all", "2023-12-31", []string{"1.5"}},
		{"unreadable", "soon", []string{"3", "2", "1.5", "1"}},
	} {
		var got []string
		for _, r := range Within(all, ParseCutoff(c.cutoff), date) {
			got = append(got, r.v)
		}
		if len(got) != len(c.want) {
			t.Fatalf("%s: got %v want %v", c.name, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%s: got %v want %v", c.name, got, c.want)
			}
		}
	}
}

func TestCutoffContext(t *testing.T) {
	if !CutoffOf(WithCutoff(context.Background(), time.Time{})).IsZero() {
		t.Fatal("zero cutoff must not be set")
	}
	c := ParseCutoff("2024-02-10")
	if !CutoffOf(WithCutoff(context.Background(), c)).Equal(c) {
		t.Fatal("cutoff lost")
	}
}

func TestReleasesWithin(t *testing.T) {
	all := []github.Release{{Tag: "new", Published: "2024-03-01T10:00:00Z"}, {Tag: "old", Published: "2024-01-01T00:00:00Z"}, {Tag: "undated"}}
	var got []string
	for _, r := range ReleasesWithin(all, ParseCutoff("2024-02-01")) {
		got = append(got, r.Tag)
	}
	if len(got) != 2 || got[0] != "old" || got[1] != "undated" {
		t.Fatalf("got %v", got)
	}
}

func TestValidCutoff(t *testing.T) {
	for in, want := range map[string]bool{"2024-02-10": true, "2024-13-40": false, "yesterday": false, "": false} {
		if got := ValidCutoff(in); got != want {
			t.Errorf("ValidCutoff(%q) = %v, want %v", in, got, want)
		}
	}
}
