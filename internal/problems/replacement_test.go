package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/meta"
)

func TestReplacementFromSummaryNexusLink(t *testing.T) {
	m := fakeMeta{
		pages: map[int]meta.Page{
			5098: {ID: 5098, Name: "New Mod", Downloads: []meta.File{
				{ID: 1, Type: "MAIN", FileName: "new.zip", Mods: []meta.Mod{{UniqueID: "Author.NewMod", Version: "1.0"}}},
			}},
		},
	}
	got := replacementFromSummary(context.Background(), m, nil, "obsolete; use https://www.nexusmods.com/stardewvalley/mods/5098")
	if got == nil || got.PageID != 5098 || got.FileID != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestReplacementFromSummaryGitHubLink(t *testing.T) {
	got := replacementFromSummary(context.Background(), fakeMeta{}, nil, "see https://github.com/pathoschild/stardewmods")
	if got == nil || got.GitHub != "pathoschild/stardewmods" {
		t.Fatalf("got %+v", got)
	}
}

func TestBrokenIncludesAbandoned(t *testing.T) {
	m := fakeMeta{compat: map[string]meta.UpdateResult{
		"A": {Compatibility: "Abandoned", CompatibilitySummary: "unmaintained"},
	}}
	got := Check(context.Background(), m, Environment{}, []Installed{mod("a", "A", "1", true)})
	if len(got.Broken) != 1 || got.Broken[0].Status != "abandoned" || got.Broken[0].Summary != "unmaintained" {
		t.Fatalf("broken = %+v", got.Broken)
	}
}
