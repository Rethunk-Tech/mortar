package problems

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

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

type slowMeta struct {
	fakeMeta
	inFlight, peak atomic.Int32
}

func (s *slowMeta) Page(ctx context.Context, id int) (meta.Page, error) {
	n := s.inFlight.Add(1)
	defer s.inFlight.Add(-1)
	for p := s.peak.Load(); n > p && !s.peak.CompareAndSwap(p, n); p = s.peak.Load() {
	}
	time.Sleep(20 * time.Millisecond)
	return s.fakeMeta.Page(ctx, id)
}

func TestDependencyNamesLookUpInParallelWithinTheBound(t *testing.T) {
	m := &slowMeta{fakeMeta: fakeMeta{refs: map[string][]meta.Ref{}, pages: map[int]meta.Page{}}}
	var ids []string
	for i := 1; i <= 24; i++ {
		id := fmt.Sprintf("smapi:Mod.N%d", i)
		ids = append(ids, id)
		m.refs[fmt.Sprintf("Mod.N%d", i)] = []meta.Ref{{Site: "Nexus", ID: i}}
		m.pages[i] = meta.Page{Name: fmt.Sprintf("Name %d", i)}
	}
	start := time.Now()
	got := DependencyNames(context.Background(), m, ids)
	if len(got) != 24 || got[ids[3]] != "Name 4" {
		t.Fatalf("names = %v", got)
	}
	if peak := m.peak.Load(); peak < 2 || peak > nameLookups {
		t.Fatalf("peak concurrency %d, want 2..%d", peak, nameLookups)
	}
	if d := time.Since(start); d > 300*time.Millisecond {
		t.Fatalf("took %v: serial would be 480ms", d)
	}
}
