package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

func TestNextScansRunningProfilesWithoutTheQueueLock(t *testing.T) {
	t.Parallel()
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

func TestABusySourceHoldsOnlyItsOwnItemsAndGivesUp(t *testing.T) {
	t.Parallel()
	now := time.Unix(1000, 0).UTC()
	s := &Service{
		d:    Deps{Now: func() time.Time { return now }, Premium: func() bool { return true }, Dir: t.TempDir()},
		kick: make(chan struct{}, 1),
		items: []*Item{
			{ID: "mr", State: StateDownloading, Package: "sodium", Source: "modrinth", Game: "g", Profile: "p"},
			{ID: "gh", State: StateQueued, Repo: "o/r", Game: "g", Profile: "q"},
		},
	}
	busy := func() { s.settle("mr", fmt.Errorf("modrinth: %w", &source.BusyError{Source: "Modrinth"})) }
	for i := range maxBusy {
		busy()
		if it, _, _ := s.next(map[string]bool{}); it == nil || it.ID != "gh" {
			t.Fatalf("answer %d: another source's item did not go ahead: %v", i+1, it)
		}
		if want := now.Add(defaultBackoff << i); !s.until["modrinth"].Equal(want) {
			t.Fatalf("answer %d: waits until %v, want %v", i+1, s.until["modrinth"], want)
		}
		now = s.until["modrinth"]
		s.find("mr").State = StateDownloading
	}
	busy()
	if it := s.find("mr"); it.State != StateFailed || it.Error != "Modrinth kept refusing; Retry" {
		t.Fatalf("after %d busy answers: %+v", maxBusy+1, it)
	}
	s.Retry("mr")
	reset := now.Add(10 * time.Minute)
	s.find("mr").State = StateDownloading
	s.settle("mr", &source.BusyError{Source: "Modrinth", Reset: reset})
	if it := s.find("mr"); it.State != StateQueued || it.busy != 1 || !s.until["modrinth"].Equal(reset) {
		t.Fatalf("a retried item starts its count over and keeps the source's own reset: %+v until %v", it, s.until["modrinth"])
	}
}

// A step's error that lands after the player paused or cancelled the item leaves the player's choice standing.
func TestAnErrorAfterPauseOrCancelKeepsTheItemPausedOrCancelled(t *testing.T) {
	t.Parallel()
	for _, state := range []string{StatePaused, StateCancelled} {
		s := &Service{
			d:       Deps{Now: func() time.Time { return time.Unix(1, 0).UTC() }, Dir: t.TempDir()},
			kick:    make(chan struct{}, 1),
			cancels: map[string]context.CancelFunc{},
			items:   []*Item{{ID: "a", Game: "g", Profile: "p", State: state}},
		}
		s.settle("a", fmt.Errorf("read: connection reset by peer"))
		s.settle("a", &source.BusyError{Source: "nexus"})
		if got := s.items[0].State; got != state {
			t.Fatalf("%s item became %s", state, got)
		}
	}
}

func TestWaitCapsAHugeReset(t *testing.T) {
	t.Parallel()
	now := time.Unix(1, 0).UTC()
	s := &Service{d: Deps{Now: func() time.Time { return now }}}
	s.wait("github", now.Add(50*24*time.Hour), defaultBackoff)
	if got := s.until["github"]; !got.Equal(now.Add(maxWait)) {
		t.Fatalf("until %v", got)
	}
}

// A CDN's limit on a Nexus item holds the Nexus key, whichever site's wrapper carries it.
func TestARateLimitHoldsTheItemsOwnSource(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_000, 0).UTC()
	reset := now.Add(time.Hour)
	s := &Service{
		d:       Deps{Now: func() time.Time { return now }, Dir: t.TempDir()},
		kick:    make(chan struct{}, 1),
		cancels: map[string]context.CancelFunc{},
		items:   []*Item{{ID: "a", Game: "g", Profile: "p", State: StateDownloading}},
	}
	s.settle("a", &github.RateLimitError{Reset: reset})
	if got := s.until["nexus"]; !got.Equal(reset) || !s.until["github"].IsZero() {
		t.Fatalf("until %v", s.until)
	}
}
