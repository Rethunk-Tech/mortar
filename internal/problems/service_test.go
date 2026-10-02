package problems

import (
	"context"
	"sync"
	"testing"
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
	for range 2 {
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
	}
	<-start
	close(release)
	wg.Wait()
	if calls != 1 {
		t.Fatalf("check called %d times, want 1", calls)
	}
}
