package queue

import (
	"context"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/github"
)

// sourceLimit is a source's politeness: at most cap fetches at once, and starts paced by a token bucket. Nexus has
// its own headers (the queue's rate-limit pause), so it has none.
type sourceLimit struct {
	sem    chan struct{}
	mu     sync.Mutex
	tokens float64
	burst  float64
	perSec float64
	last   time.Time
}

func newSourceLimit(concurrent int, burst, perSec float64, now time.Time) *sourceLimit {
	return &sourceLimit{sem: make(chan struct{}, concurrent), tokens: burst, burst: burst, perSec: perSec, last: now}
}

// take reports how long to wait before a start may happen; zero means a token was taken.
func (l *sourceLimit) take(now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens = min(l.burst, l.tokens+now.Sub(l.last).Seconds()*l.perSec)
	l.last = now
	if l.tokens >= 1 {
		l.tokens--
		return 0
	}
	return time.Duration((1 - l.tokens) / l.perSec * float64(time.Second))
}

func (l *sourceLimit) acquire(ctx context.Context, now func() time.Time) (func(), error) {
	select {
	case l.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	release := func() { <-l.sem }
	for {
		wait := l.take(now())
		if wait == 0 {
			return release, nil
		}
		select {
		case <-ctx.Done():
			release()
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// limitFor creates a source's limit on first use. GitHub allows far more to a logged-in gh user than to an
// anonymous address, so its limit is sized by the login seen then.
func (s *Service) limitFor(ctx context.Context, source string) *sourceLimit {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l := s.limits[source]; l != nil {
		return l
	}
	now := s.d.Now()
	var l *sourceLimit
	switch {
	case source == "github" && github.DefaultAuth.LoggedIn(ctx):
		l = newSourceLimit(6, 10, 5, now)
	default:
		l = newSourceLimit(2, 2, 1, now)
	}
	if s.limits == nil {
		s.limits = map[string]*sourceLimit{}
	}
	s.limits[source] = l
	return l
}

// sourceSlot holds a fetch to its source's limit; the returned func frees it.
func (s *Service) sourceSlot(ctx context.Context, it Item) (func(), error) {
	var source string
	switch {
	case it.Package != "":
		source = "thunderstore"
	case it.Repo != "":
		source = "github"
	default:
		return func() {}, nil
	}
	return s.limitFor(ctx, source).acquire(ctx, s.d.Now)
}
