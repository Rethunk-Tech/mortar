package profile

import (
	"sync/atomic"
	"testing"
	"time"
)

// A request during a run asks for exactly one more run, and wait returns only after the last one.
func TestBackgroundSyncCoalescesAndWaits(t *testing.T) {
	t.Parallel()
	var b bgSync
	var runs atomic.Int32
	release := make(chan struct{})
	run := func() {
		if runs.Add(1) == 1 {
			<-release
		}
	}
	b.request("g/p", run)
	for range 3 {
		b.request("g/p", run)
	}
	waited := make(chan struct{})
	go func() {
		b.wait("g/p")
		close(waited)
	}()
	select {
	case <-waited:
		t.Fatal("wait returned while a run was under way")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	<-waited
	if n := runs.Load(); n != 2 {
		t.Fatalf("runs = %d, want 2", n)
	}
}
