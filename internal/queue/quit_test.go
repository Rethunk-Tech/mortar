package queue

import (
	"testing"
	"time"
)

func newQuitService(t *testing.T, states ...string) *Service {
	t.Helper()
	s, err := New(Deps{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	for i, state := range states {
		s.items = append(s.items, &Item{ID: string(rune('a' + i)), State: state})
	}
	return s
}

func stopWaitReturns(s *Service, wait func(), limit time.Duration) <-chan struct{} {
	returned := make(chan struct{})
	go func() {
		s.StopWait(wait, limit)
		close(returned)
	}()
	return returned
}

func TestStopWaitReturnsWhenWorkersStop(t *testing.T) {
	s := newQuitService(t)
	select {
	case <-stopWaitReturns(s, func() { time.Sleep(5 * time.Millisecond) }, time.Second):
	case <-time.After(2 * time.Second):
		t.Fatal("StopWait did not return after the workers stopped")
	}
}

func TestStopWaitAbandonsAStuckDownload(t *testing.T) {
	s := newQuitService(t, StateDownloading)
	stuck := make(chan struct{})
	defer close(stuck)
	select {
	case <-stopWaitReturns(s, func() { <-stuck }, 20*time.Millisecond):
	case <-time.After(2 * time.Second):
		t.Fatal("StopWait kept waiting for a worker that is only downloading")
	}
}

func TestStopWaitDoesNotAbandonAnInstall(t *testing.T) {
	s := newQuitService(t, StateInstalling)
	finish := make(chan struct{})
	returned := stopWaitReturns(s, func() { <-finish }, 20*time.Millisecond)
	select {
	case <-returned:
		t.Fatal("StopWait gave up on an install under way")
	case <-time.After(10 * installPoll):
	}
	close(finish)
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("StopWait did not return once the install finished")
	}
}
