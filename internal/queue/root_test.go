package queue

import (
	"context"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestNeedsRootAnswerAndFail(t *testing.T) {
	ask := &profile.RemapAsk{Key: "store-key", Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 2}}
	s := &Service{
		d: Deps{
			Dir: t.TempDir(),
			Now: func() time.Time { return time.Unix(1, 0).UTC() },
		},
		kick:    make(chan struct{}, 1),
		cancels: map[string]context.CancelFunc{},
	}
	s.items = []*Item{{
		ID: "a", Game: "stardew", Profile: "p", State: StateNeedsRoot, staged: "store-key", Remap: ask,
	}}

	s.AnswerRoot("a", "ModFolder")
	if it := s.find("a"); it.State != StateQueued || it.chosenRoot != "ModFolder" || it.staged != "store-key" {
		t.Fatalf("after AnswerRoot: %+v", it)
	}

	s.items[0].State, s.items[0].chosenRoot = StateNeedsRoot, ""
	s.FailRoot("a")
	if it := s.find("a"); it.State != StateFailed || it.Error == "" || it.staged != "" || it.Remap != nil {
		t.Fatalf("after FailRoot: %+v", it)
	}
}
