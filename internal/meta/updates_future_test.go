package meta

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// An answer cached by a clock that was later set back is not fresh: the next check asks again.
func TestFutureStampedUpdateAnswerIsNotFresh(t *testing.T) {
	var down atomic.Bool
	var hits atomic.Int32
	c := server(t, &down, &hits)
	now := time.Now()
	c.Now = func() time.Time { return now }
	c.CheckUpdates(context.Background(), cpRequest)
	now = now.Add(-30 * 24 * time.Hour) // clock set back a month
	before := hits.Load()
	c.CheckUpdates(context.Background(), cpRequest)
	// A stale answer is served while a background refresh asks again; a fresh one is never asked again.
	for range 200 {
		if hits.Load() != before {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if hits.Load() == before {
		t.Errorf("future-stamped answer served as fresh")
	}
	// and Fresh/young (the shared rule) refuses it:
	if c.young(now.Add(time.Hour), updatesTTL) {
		t.Errorf("young accepts future")
	}
}
