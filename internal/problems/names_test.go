package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

func TestDependencyNamesUseTheNexusPage(t *testing.T) {
	m := fakeMeta{
		refs:  map[string][]meta.Ref{"Pathoschild.ContentPatcher": {{Site: "GitHub", ID: 1}, {Site: "Nexus", ID: 1915}}},
		pages: map[int]meta.Page{1915: {Name: "Content Patcher"}},
	}
	got := DependencyNames(context.Background(), m, []string{"smapi:Pathoschild.ContentPatcher", "No.Such"})
	if len(got) != 1 || got["smapi:Pathoschild.ContentPatcher"] != "Content Patcher" {
		t.Fatalf("names = %v", got)
	}
}
