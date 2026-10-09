package shortcut_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/shortcut"
)

// serve runs a control channel whose status for the game steps through states, one per call, then stays on the last.
func serve(t *testing.T, states ...launchsvc.State) (dir string, asked func() []control.Params) {
	t.Helper()
	dir = t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	var requests []control.Params
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = control.Serve(ctx, dir, "test", func(_ context.Context, method string, p control.Params) (any, error) {
			mu.Lock()
			defer mu.Unlock()
			switch method {
			case shortcut.PlayRequestMethod:
				requests = append(requests, p)
				return shortcut.Request{Game: p.Game, Profile: p.Profile}, nil
			case "status":
				st := states[0]
				if len(states) > 1 {
					states = states[1:]
				}
				return launchsvc.Status{Game: p.Game, State: st}, nil
			}
			return nil, errors.New("unexpected " + method)
		})
	}()
	t.Cleanup(func() { cancel(); <-done })
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, ok := controlwire.Live(dir); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("control channel never came up")
		}
		time.Sleep(10 * time.Millisecond)
	}
	return dir, func() []control.Params {
		mu.Lock()
		defer mu.Unlock()
		return append([]control.Params(nil), requests...)
	}
}

func TestFollowForwardsTheRequestAndLastsAsLongAsTheRun(t *testing.T) {
	dir, asked := serve(t, launchsvc.Idle, launchsvc.Launching, launchsvc.Running, launchsvc.Running, launchsvc.Idle)
	r := shortcut.Request{Game: "stardew", Profile: "p1"}
	if err := shortcut.Follow(dir, r, time.Millisecond, time.Minute); err != nil {
		t.Fatal(err)
	}
	if got := asked(); len(got) != 1 || got[0].Game != "stardew" || got[0].Profile != "p1" {
		t.Fatalf("forwarded %+v", got)
	}
}

func TestFollowGivesUpOnAGameThatNeverStarts(t *testing.T) {
	dir, _ := serve(t, launchsvc.Idle)
	err := shortcut.Follow(dir, shortcut.Request{Game: "stardew", Profile: "p1"}, time.Millisecond, 20*time.Millisecond)
	if !errors.Is(err, shortcut.ErrNeverStarted) {
		t.Fatalf("got %v", err)
	}
}
