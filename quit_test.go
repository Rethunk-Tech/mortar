package main

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchdogExitsOnceAfterGraceAndWaitsForInstalls(t *testing.T) {
	var installing atomic.Bool
	installing.Store(true)
	finished := make(chan struct{}, 2)
	exited := make(chan int, 2)
	s := &QuitService{
		grace:  10 * time.Millisecond,
		busy:   installing.Load,
		finish: func() { finished <- struct{}{} },
		exit:   func(code int) { exited <- code },
	}
	s.armWatchdog()
	s.armWatchdog()
	select {
	case <-exited:
		t.Fatal("exited while an install was running")
	case <-time.After(150 * time.Millisecond):
	}
	installing.Store(false)
	select {
	case code := <-exited:
		if code != 0 || len(finished) != 1 {
			t.Fatalf("exit code %d after %d finishes", code, len(finished))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog never exited")
	}
	select {
	case <-exited:
		t.Fatal("armed twice")
	case <-time.After(100 * time.Millisecond):
	}
}
