package share

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func refIDs(refs []Ref) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Identity()
	}
	return out
}

func TestPlaceRefs(t *testing.T) {
	a, b, c, d := Ref{ModID: 1, FileID: 1}, Ref{ModID: 2, FileID: 2}, Ref{ModID: 3, FileID: 3}, Ref{ModID: 4, FileID: 4}

	got := placeRefs([]Ref{b, c}, []Ref{a, b, c}, []Ref{a})
	if !slices.Equal(refIDs(got), []string{a.Identity(), b.Identity(), c.Identity()}) {
		t.Fatalf("add at 0: %v", refIDs(got))
	}

	got = placeRefs([]Ref{a, c}, []Ref{a, b, d, c}, []Ref{d})
	if !slices.Equal(refIDs(got), []string{a.Identity(), d.Identity(), c.Identity()}) {
		t.Fatalf("missing predecessor: %v", refIDs(got))
	}

	kept := Ref{ModID: 1, FileID: 1, Note: "keep"}
	got = placeRefs([]Ref{kept, b}, []Ref{a, b}, []Ref{a})
	var notes []string
	for _, r := range got {
		if r.Identity() == a.Identity() {
			notes = append(notes, r.Note)
		}
	}
	if !slices.Contains(notes, "keep") {
		t.Fatalf("have entry was changed: notes=%v got=%v", notes, refIDs(got))
	}
}

func TestSameExceptArrivalComparesConfigsBothWays(t *testing.T) {
	cfgA := Config{ID: "smapi:Me.A", Path: "config.json", Data: []byte(`{"v":2}`)}
	cfgB := Config{ID: "smapi:Me.B", Path: "config.json", Data: []byte(`{"b":1}`)}
	base := Preview{
		Name: "Main", Entries: []Ref{{ModID: 1, FileID: 1}, {ModID: 2, FileID: 2}},
		IDs: []mod.ID{"smapi:Me.A", "smapi:Me.B"}, Configs: []Config{cfgA, cfgB},
	}
	missing := map[string]bool{"n:2:2": true}
	have := func(configs ...Config) Preview {
		return Preview{Name: "Main", Entries: []Ref{{ModID: 1, FileID: 1}}, IDs: []mod.ID{"smapi:Me.A"}, Configs: configs}
	}
	tests := []struct {
		name string
		have Preview
		same bool
	}{
		{"only the unarrived mod's config is absent", have(cfgA), true},
		{"the player deleted an arrived mod's config", have(), false},
		{"the player added a config", have(cfgA, Config{ID: "smapi:Me.A", Path: "config/x.json", Data: []byte(`{}`)}), false},
	}
	for _, tt := range tests {
		if got := SameExceptArrival(base, tt.have, missing); got != tt.same {
			t.Errorf("%s: same = %v, want %v", tt.name, got, tt.same)
		}
	}
}
