package nexus

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-nexusmods"
)

func TestRateLimitWording(t *testing.T) {
	reset := time.Date(2026, 9, 30, 13, 0, 0, 0, time.UTC)
	if got := (&RateLimitError{Reset: reset}).Error(); got != "nexus rate limit reached until 2026-09-30T13:00:00Z" {
		t.Fatalf("text = %q", got)
	}
	if got := (&RateLimitError{}).Error(); got != "nexus rate limit reached" {
		t.Fatalf("zero reset printed: %q", got)
	}
}

// A 429 from the module reads as Mortar's sentence and still matches the module's type for the queue's backoff.
func TestWordedRateLimitFromTheModule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	c := nexusmods.New("Mortar", "9.9")
	c.BaseURL = srv.URL
	_, err := c.WithKey("k").Validate(t.Context())
	err = Worded(err)
	var re *RateLimitError
	if !errors.As(err, &re) || time.Until(re.Reset) < time.Minute || !strings.HasPrefix(err.Error(), "nexus rate limit reached until ") {
		t.Fatalf("429 = %v", err)
	}
	var limit *httpkit.RateLimitError
	if !errors.As(err, &limit) || !errors.Is(err, httpkit.ErrRateLimited) {
		t.Fatalf("the module's type is lost: %v", err)
	}
	plain := fmt.Errorf("other: %w", errors.ErrUnsupported)
	if !errors.Is(Worded(plain), plain) || Worded(nil) != nil {
		t.Fatal("an error that is not a rate limit changed")
	}
}

func TestUnauthorizedWording(t *testing.T) {
	if nexusmods.ErrUnauthorized.Error() != "nexus rejected the API key" {
		t.Fatalf("text = %q", nexusmods.ErrUnauthorized)
	}
}
