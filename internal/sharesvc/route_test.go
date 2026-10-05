package sharesvc

import "testing"

func TestParseModRoute(t *testing.T) {
	tests := []struct {
		value string
		game  string
		modID int
		ok    bool
	}{
		{value: "mortar://stardew/mod/123", game: "stardew", modID: 123, ok: true},
		{value: "mortar://stardew/mod/1", game: "stardew", modID: 1, ok: true},
		{value: "mortar://stardewvalley/mod/123", game: "stardew", modID: 123, ok: true},
		{value: "mortar://unknown/mod/123"},
		{value: "mortar://stardew/mod/0"},
		{value: "mortar://stardew/mod/01"},
		{value: "mortar://stardew/mod/+1"},
		{value: "mortar://stardew/mod/123/"},
		{value: "mortar://stardew/mod/123?source=extension"},
		{value: "mortar://stardew/p/123"},
	}
	for _, test := range tests {
		got, ok := parseModRoute(test.value)
		if ok != test.ok || (ok && (got.game != test.game || got.modID != test.modID)) {
			t.Errorf("parseModRoute(%q) = %+v, %v", test.value, got, ok)
		}
	}
}
