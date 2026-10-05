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

func TestBusyAndStopFollowTheInstall(t *testing.T) {
	svc, _ := sweepEnv(t)
	svc.procDir = t.TempDir()
	svc.set(Status{Game: "stardew", State: Launching, Profile: "p", Install: "abc"})
	if !svc.BusyInstall("abc") || svc.BusyInstall("other") || !svc.AnyBusy() || !svc.Busy("stardew") {
		t.Fatal("busy must follow the install the launch runs, and stay true for the game")
	}
	if err := svc.StopInstall("other"); err == nil {
		t.Fatal("stopped an install that is not running")
	}
	svc.set(Status{Game: "stardew", State: Idle})
	if svc.BusyInstall("abc") || svc.AnyBusy() {
		t.Fatal("an idle game kept its install busy")
	}
}
