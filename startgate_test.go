package main

import "testing"

// Before startup no app exists to dispatch on, so run must hold the work rather than call into Wails.
func TestStartGateHoldsWorkUntilStartup(t *testing.T) {
	g := &startGate{}
	ran := false
	g.run(func() { ran = true })
	if ran || len(g.early) != 1 {
		t.Fatalf("ran = %v, held = %d before startup", ran, len(g.early))
	}
}
