package share

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

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
