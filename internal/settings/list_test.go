package settings

import (
	"slices"
	"testing"
)

func TestListColumnsKeepEveryColumnTheListOffers(t *testing.T) {
	for _, id := range []string{"size", "startup", "released", "kind", "pluginVersion"} {
		if got := sanitizeListColumns([]string{"on", "name", id}); !slices.Contains(got, id) {
			t.Errorf("saving %q kept %v", id, got)
		}
	}
}
