package settings

import (
	"slices"
	"testing"
)

func TestListColumnsKeepSizeAndStartup(t *testing.T) {
	got := sanitizeListColumns([]string{"on", "name", "size", "startup"})
	if !slices.Contains(got, "size") || !slices.Contains(got, "startup") {
		t.Fatalf("columns %v", got)
	}
}
