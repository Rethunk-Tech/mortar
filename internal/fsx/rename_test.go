package fsx

import (
	"errors"
	"testing"
	"time"
)

func TestRetryRidesOutTransientFailure(t *testing.T) {
	busy := errors.New("busy")
	calls := 0
	err := retry(func() error {
		calls++
		if calls < 4 {
			return busy
		}
		return nil
	}, func(err error) bool { return err == busy }, time.Second)
	if err != nil || calls != 4 {
		t.Fatalf("err=%v calls=%d, want success on the 4th try", err, calls)
	}
}

func TestRetryStopsOnPermanentFailureAndAfterWindow(t *testing.T) {
	perm := errors.New("perm")
	calls := 0
	if err := retry(func() error { calls++; return perm }, func(error) bool { return false }, time.Second); err != perm || calls != 1 {
		t.Fatalf("permanent error retried: err=%v calls=%d", err, calls)
	}
	start := time.Now()
	if err := retry(func() error { return perm }, func(error) bool { return true }, 50*time.Millisecond); err != perm {
		t.Fatalf("err=%v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("retry outlived its window")
	}
}
