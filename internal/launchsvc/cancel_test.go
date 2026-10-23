//go:build !windows

package launchsvc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

func TestACancelledLaunchEndsIdleNotFailed(t *testing.T) {
	svc, p := startEnv(t)
	sl := svc.selectedSlot(game.Find("stardew"))
	svc.Starter = game.Starter{
		Runner:   func(string, string, ...string) (<-chan error, error) { return make(chan error), nil },
		LookPath: func(string) (string, error) { return "", errors.New("none") },
	}
	ctx, cancel := context.WithCancel(context.Background())
	buf := &launch.Buffer{}
	svc.mu.Lock()
	svc.logs[keyOf(sl)], svc.stop[keyOf(sl)] = session{buf: buf, profile: p.ID, started: time.Now()}, cancel
	svc.mu.Unlock()
	svc.set(Status{Game: "stardew", Install: installOf(sl), State: Launching, Profile: p.ID})
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.run(ctx, sl, p.ID, launchRun{plan: &launchplan.Plan{Exe: "/bin/game"}, env: game.StartEnv{Direct: true}}, buf)
	}()
	if err := svc.CancelLaunch(context.Background(), "stardew"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the launch kept waiting after it was cancelled")
	}
	if st := svc.current(sl); st.State != Idle || st.Error != "" {
		t.Fatalf("status after a cancel = %+v", st)
	}
	if why := svc.LaunchFailure("stardew", installOf(sl)); why != "" {
		t.Fatalf("a cancelled launch was recorded as failing: %q", why)
	}
	if err := svc.CancelLaunch(context.Background(), "stardew"); err == nil {
		t.Fatal("cancelling a launch that is not starting succeeded")
	}
}
