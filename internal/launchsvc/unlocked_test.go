package launchsvc

import "testing"

func TestIdleCallsUnlocked(t *testing.T) {
	svc := NewService(t.TempDir(), nil, nil)
	var n int
	svc.Unlocked = func() { n++ }
	svc.set(Status{Game: "stardew", State: Launching, Profile: "p"})
	if n != 0 {
		t.Fatalf("Unlocked during launch = %d", n)
	}
	svc.set(Status{Game: "stardew", State: Idle})
	if n != 1 {
		t.Fatalf("Unlocked on idle = %d", n)
	}
}
