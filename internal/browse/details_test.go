package browse

import (
	"context"
	"sync/atomic"
	"testing"

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
	if _, err := s.Details(t.Context(), "g", "nexus", "1"); err == nil {
		t.Fatal("a source the game does not list answered")
	}
}
