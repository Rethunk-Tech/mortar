//go:build !windows

package launchsvc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestAStartIsLaunchingFromAcceptanceAndATestLaunchReportsItsFailure(t *testing.T) {
	svc, p := startEnv(t)
	release := make(chan struct{})
	svc.EnsureLoader = func(context.Context, string, string, bool) error {
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

	svc.EnsureLoader = func(context.Context, string, string, bool) error { return errors.New("the game did not start in time") }
	res, err := svc.TestLaunch(context.Background(), "stardew", p.ID, "")
	if err == nil || res.ReachedTitle {
		t.Fatalf("a failed start reported %+v, %v", res, err)
	}
}

func TestATestLaunchWithFieldsIgnoresTheDefaultPreset(t *testing.T) {
	svc, p := startEnv(t)
	if _, err := svc.profiles.SetLaunchPresets("stardew", p.ID, []profile.LaunchPreset{{ID: "d", Name: "Debug", LaunchOptions: "--from-preset"}}, "d"); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), specKey{}, profile.LaunchSpec{Options: "--shown"})
	spec, err := svc.launchSpec(ctx, "stardew", p.ID, "")
	if err != nil || spec.Options != "--shown" {
		t.Fatalf("spec = %+v, %v", spec, err)
	}
	if spec, err = svc.launchSpec(context.Background(), "stardew", p.ID, ""); err != nil || spec.Options != "--from-preset" {
		t.Fatalf("without fields the default preset applies: %+v, %v", spec, err)
	}
}
