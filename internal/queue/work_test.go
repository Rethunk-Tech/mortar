package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestNextScansRunningProfilesWithoutTheQueueLock(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	s := &Service{
		d: Deps{
			Now:     func() time.Time { return time.Unix(1, 0).UTC() },
			Premium: func() bool { return false },
			Running: func(string, string) bool {
				select {
				case <-entered:
				default:
					close(entered)
				}
				<-release
				return true
			},
		},
		kick:    make(chan struct{}, 1),
		cancels: map[string]context.CancelFunc{},
		items:   []*Item{{ID: "a", Game: "stardew", Profile: "p", State: StateQueued, Repo: "o/r", Tag: "1", Asset: "a.zip"}},
	}
	done := make(chan struct{})
	go func() {
		s.step(context.Background())
		close(done)
	}()
	<-entered
	start := time.Now()
	_ = s.State()
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("State blocked while Running scanned processes")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("step did not finish")
	}
}

func TestASourceBusyAnswerRequeuesTheItemAndPausesTheQueue(t *testing.T) {
	now := time.Unix(1000, 0).UTC()
	s := &Service{
		d:     Deps{Now: func() time.Time { return now }},
		kick:  make(chan struct{}, 1),
		items: []*Item{{ID: "a", State: StateDownloading}},
	}
	s.settle("a", fmt.Errorf("modrinth: %w", &source.BusyError{Source: "Modrinth"}))
	if it := s.find("a"); it.State != StateQueued || !s.until.Equal(now.Add(defaultBackoff)) {
		t.Fatalf("item %+v until %v", it, s.until)
	}
}
