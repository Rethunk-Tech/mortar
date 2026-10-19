package github

import (
	"io"
	"sync"
	"time"
)

// SpeedLimit reports the most bytes a second all downloads together may use; zero or nil is unlimited. It is read on
// every chunk, so a change reaches downloads already running.
var SpeedLimit func() int64

// minBurst is the bytes a download may take at once however low the limit is.
const minBurst = 16 << 10

// bucket is one token bucket shared by every download, so the limit holds across them together.
type bucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
}

var shared bucket

// reserve takes n bytes from the bucket at rate bytes a second and says how long the caller must wait before
// sending them on. A quarter second of the rate (at least minBurst) is available at once.
func (b *bucket) reserve(n, rate int64, now time.Time) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	burst := max(float64(rate)/4, minBurst)
	if b.last.IsZero() {
		b.tokens = burst
	} else {
		b.tokens = min(burst, b.tokens+now.Sub(b.last).Seconds()*float64(rate))
	}
	b.last = now
	b.tokens -= float64(n)
	if b.tokens >= 0 {
		return 0
	}
	return time.Duration(-b.tokens / float64(rate) * float64(time.Second))
}

type throttled struct {
	done <-chan struct{}
	err  func() error
	r    io.Reader
}

func (t *throttled) Read(p []byte) (int, error) {
	var rate int64
	if SpeedLimit != nil {
		rate = SpeedLimit()
	}
	if rate <= 0 {
		return t.r.Read(p)
	}
	if burst := max(rate/4, minBurst); int64(len(p)) > burst {
		p = p[:burst]
	}
	n, err := t.r.Read(p)
	if n > 0 {
		if wait := shared.reserve(int64(n), rate, time.Now()); wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-t.done:
				return n, t.err()
			}
		}
	}
	return n, err
}
