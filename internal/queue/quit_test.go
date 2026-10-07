package queue

import (
	"testing"
	"time"
)

func TestWaitBoundedReturnsWhenWorkersStop(t *testing.T) {
	if !waitBounded(func() { time.Sleep(5 * time.Millisecond) }, time.Second) {
		t.Fatal("a wait that finishes in time was reported as abandoned")
	}
}

func TestWaitBoundedAbandonsAStuckWorker(t *testing.T) {
	stuck := make(chan struct{})
	defer close(stuck)
	start := time.Now()
	if waitBounded(func() { <-stuck }, 20*time.Millisecond) {
		t.Fatal("a stuck wait was reported as finished")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("gave up after %s, want about 20ms", time.Since(start))
	}
}
