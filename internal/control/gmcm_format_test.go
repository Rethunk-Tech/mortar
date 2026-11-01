package control

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/gmcm"
)

func TestFormatMenuListsPageSlashIndexNameValue(t *testing.T) {
	t.Parallel()
	menu := gmcm.Capture{Pages: []gmcm.Page{
		{ID: "main", Options: []gmcm.Option{{Index: 0, Name: "Speed", Value: 1.5}, {Index: 2, Name: "Hard mode", Value: true}}},
		{ID: "extra"},
	}}
	want := []string{"main/0 Speed 1.5", "main/2 Hard mode true"}
	if got := formatMenu(menu); !slices.Equal(got, want) {
		t.Fatalf("formatMenu = %q, want %q", got, want)
	}
	if got := formatMenu(gmcm.Capture{}); got != nil {
		t.Fatalf("an empty capture lists %q", got)
	}
}
