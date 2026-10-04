//go:build !windows

package launchsvc

import "testing"

func TestMeasureNextLaunchCanBeCancelled(t *testing.T) {
	svc, p := startEnv(t)
	if err := svc.MeasureNextLaunch("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if pending, err := svc.MeasureNextLaunchPending("stardew", p.ID); err != nil || !pending {
		t.Fatalf("pending after asking = %v, %v", pending, err)
	}
	if err := svc.CancelMeasureNextLaunch("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if pending, err := svc.MeasureNextLaunchPending("stardew", p.ID); err != nil || pending {
		t.Fatalf("pending after cancelling = %v, %v", pending, err)
	}
	if err := svc.CancelMeasureNextLaunch("stardew", p.ID); err != nil {
		t.Fatalf("cancelling twice: %v", err)
	}
}
