//go:build !windows

package launchsvc

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAStartIsLaunchingFromAcceptanceAndATestLaunchReportsItsFailure(t *testing.T) {
	svc, p := startEnv(t)
	release := make(chan struct{})
	svc.EnsureLoader = func(context.Context, string, bool) error {
		<-release
		return errors.New("the game did not start in time")
	}
	if err := svc.Start(context.Background(), "stardew", p.ID, false); err != nil {
		t.Fatal(err)
	}
	if st, err := svc.Status("stardew"); err != nil || st.State != Launching {
		t.Fatalf("status right after Start = %+v, %v", st, err)
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for svc.Running("stardew", p.ID) {
		if time.Now().After(deadline) {
			t.Fatal("start never ended")
		}
		time.Sleep(10 * time.Millisecond)
	}

	svc.EnsureLoader = func(context.Context, string, bool) error { return errors.New("the game did not start in time") }
	res, err := svc.TestLaunch(context.Background(), "stardew", p.ID, "")
	if err == nil || res.ReachedTitle {
		t.Fatalf("a failed start reported %+v, %v", res, err)
	}
}
