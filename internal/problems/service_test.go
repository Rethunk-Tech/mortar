package problems

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestShareCheckRunsOnce(t *testing.T) {
	s := &Service{checks: map[string]*problemCall{}}
	var mu sync.Mutex
	var calls int
	start := make(chan struct{})
	release := make(chan struct{})
	check := func() Result {
		mu.Lock()
		calls++
		mu.Unlock()
		close(start)
		<-release
		return Result{Unknown: true}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		result, err := s.shareCheck(context.Background(), "stardew/profile", check)
		if err != nil {
			t.Errorf("shareCheck: %v", err)
		}
		if !result.Unknown {
			t.Error("shareCheck returned the wrong result")
		}
	}()
	<-start
	go func() {
		defer wg.Done()
		result, err := s.shareCheck(context.Background(), "stardew/profile", check)
		if err != nil {
			t.Errorf("shareCheck: %v", err)
		}
		if !result.Unknown {
			t.Error("shareCheck returned the wrong result")
		}
	}()
	s.mu.Lock()
	call := s.checks["stardew/profile"]
	s.mu.Unlock()
	<-call.joined
	close(release)
	wg.Wait()
	if calls != 1 {
		t.Fatalf("check called %d times, want 1", calls)
	}
}

func TestUnknownResultExpires(t *testing.T) {
	now := time.Now()
	c := cached{fingerprint: "fp", until: now.Add(unknownResultTTL)}
	if !c.fresh("fp", now) || c.fresh("fp", now.Add(unknownResultTTL+time.Second)) || c.fresh("other", now) {
		t.Fatal("unknown results must be reused briefly, then retried, and never across fingerprints")
	}
	if !(cached{fingerprint: "fp"}).fresh("fp", now.Add(24*time.Hour)) {
		t.Fatal("complete results stay until the fingerprint changes")
	}
}
