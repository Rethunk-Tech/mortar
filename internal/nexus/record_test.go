package nexus

import (
	"net/http"
	"testing"
	"time"
)

func TestRecordRateLimitHeaders(t *testing.T) {
	c := New("test")
	if c.Limits().Known {
		t.Fatal("expected unknown limits before a response")
	}
	h := make(http.Header)
	h.Set("X-Rl-Daily-Remaining", "42")
	h.Set("X-Rl-Daily-Limit", "2500")
	h.Set("X-Rl-Daily-Reset", "2026-10-01T00:00:00Z")
	h.Set("X-Rl-Hourly-Remaining", "7")
	h.Set("X-Rl-Hourly-Limit", "500")
	h.Set("X-Rl-Hourly-Reset", "2026-09-30T18:00:00Z")
	c.record(h)
	l := c.Limits()
	if !l.Known {
		t.Fatal("expected Known after headers")
	}
	if l.Daily.Remaining != 42 || l.Daily.Limit != 2500 {
		t.Fatalf("daily: %+v", l.Daily)
	}
	if l.Hourly.Remaining != 7 || l.Hourly.Limit != 500 {
		t.Fatalf("hourly: %+v", l.Hourly)
	}
	if !l.Daily.Reset.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("daily reset: %v", l.Daily.Reset)
	}
	if !l.Hourly.Reset.Equal(time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC)) {
		t.Fatalf("hourly reset: %v", l.Hourly.Reset)
	}
}

func TestRecordIgnoresIncompleteHeaders(t *testing.T) {
	c := New("test")
	h := make(http.Header)
	h.Set("X-Rl-Daily-Remaining", "1")
	c.record(h)
	if c.Limits().Known {
		t.Fatal("incomplete headers must not mark limits known")
	}
}
