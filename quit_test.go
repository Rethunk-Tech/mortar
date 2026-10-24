package main

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchdogExitsOnceAfterGraceAndWaitsForInstalls(t *testing.T) {
	prev := watchdogPoll
	watchdogPoll = time.Millisecond
	t.Cleanup(func() { watchdogPoll = prev })
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
	case <-time.After(50 * time.Millisecond):
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
	case <-time.After(30 * time.Millisecond):
	}
}

func TestOneClickOnCloseDecides(t *testing.T) {
	cases := []struct {
		name                             string
		keepInTray, play, quitting, busy bool
		want                             closeAction
	}{
		{"tray on hides", true, false, false, false, closeHide},
		{"tray on hides even while busy", true, false, false, true, closeHide},
		{"tray off and idle quits at once", false, false, false, false, closeQuit},
		{"tray off and busy asks", false, false, false, true, closeAsk},
		{"play mode never hides", true, true, false, false, closeQuit},
		{"a quit already under way lets every later click through", false, false, true, true, closeAllow},
		{"also with the tray on", true, false, true, false, closeAllow},
	}
	for _, c := range cases {
		if got := decideClose(c.keepInTray, c.play, c.quitting, c.busy); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}
