package browse

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

type detailFake struct{ calls *atomic.Int32 }

func (detailFake) ID() string              { return "detailfake" }
func (detailFake) Name() string            { return "Detail fake" }
func (detailFake) Modes() []source.Acquire { return nil }
func (f detailFake) Details(_ context.Context, key, id, _ string) (source.Details, error) {
	f.calls.Add(1)
	return source.Details{Description: key + ":" + id}, nil
}

func TestDetailsAsksTheSourceOncePerHit(t *testing.T) {
	var calls atomic.Int32
	source.Register(detailFake{&calls})
	c := components.NewClient(nil)
	c.SetManifest(components.Manifest{Games: []components.GameInfo{{
		ID: "g", Name: "G", Marker: "g.exe", Sources: []components.GameSource{{ID: "detailfake", Key: "gk"}},
	}}})
	components.Use(c)
	t.Cleanup(func() { components.Use(nil) })

	s := &Service{}
	for range 2 {
		got, err := s.Details(t.Context(), "g", "detailfake", "a")
		if err != nil || got.Description != "gk:a" {
			t.Fatalf("details = %+v, %v", got, err)
		}
	}
	if _, err := s.Details(t.Context(), "g", "detailfake", "b"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("source asked %d times, want once per hit", calls.Load())
	}
	// A card reopened after the TTL asks again, and the answers that expired with it are dropped.
	details.Lock()
	for k, e := range details.m {
		e.until = time.Now().Add(-time.Second)
		details.m[k] = e
	}
	details.Unlock()
	if _, err := s.Details(t.Context(), "g", "detailfake", "a"); err != nil || calls.Load() != 3 {
		t.Fatalf("expired details: source asked %d times, want 3 (%v)", calls.Load(), err)
	}
	details.Lock()
	n := len(details.m)
	details.Unlock()
	if n != 1 {
		t.Fatalf("%d details kept, want only the fresh one", n)
	}
	if _, err := s.Details(t.Context(), "g", "nexus", "1"); err == nil {
		t.Fatal("a source the game does not list answered")
	}
}
