package tools

import (
	"context"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

type fakeCatalog struct {
	present map[string]bool
	ran     []string
}

func (f *fakeCatalog) GameToolPresent(_, _ string, t components.GameTool) bool {
	return f.present[t.ID]
}

func (f *fakeCatalog) RunGameTool(_ context.Context, _, _ string, t components.GameTool) error {
	f.ran = append(f.ran, t.ID)
	return nil
}

func TestACatalogToolIsListedWhenItsFilesAreThereAndLaunchedThroughTheCatalog(t *testing.T) {
	t.Parallel()
	cat := &fakeCatalog{present: map[string]bool{"redeploy": true, "cybercat": true}}
	s := &Service{store: &Store{root: t.TempDir()}, Catalog: cat}
	got, err := s.List("cyberpunk-2077")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, tool := range got {
		if !tool.Builtin {
			t.Fatalf("%s is not marked as the game's own", tool.ID)
		}
		ids = append(ids, tool.ID)
	}
	if !slices.Equal(ids, []string{"game:redeploy", "game:cybercat"}) {
		t.Fatalf("listed = %v", ids)
	}
	if err := s.Launch("cyberpunk-2077", "p", "game:redeploy"); err != nil || !slices.Equal(cat.ran, []string{"redeploy"}) {
		t.Fatalf("ran = %v, %v", cat.ran, err)
	}
	if err := s.Launch("cyberpunk-2077", "p", "game:nope"); err == nil {
		t.Fatal("an unknown catalog tool started")
	}
}
