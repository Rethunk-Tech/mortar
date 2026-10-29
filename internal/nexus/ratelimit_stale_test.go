package nexus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRetryAfterBeatsAStaleBudgetReset(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			w.Header().Set("X-Rl-Daily-Remaining", "100")
			w.Header().Set("X-Rl-Hourly-Remaining", "50")
			w.Header().Set("X-Rl-Hourly-Reset", now.Add(-time.Hour).Format(time.RFC3339))
			w.Header().Set("X-Rl-Daily-Reset", now.Add(-time.Hour).Format(time.RFC3339))
			_, _ = w.Write([]byte(`{"user_id":1,"name":"x"}`))
			return
		}
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := New("9.9")
	c.BaseURL, c.CacheDir, c.Now = srv.URL, t.TempDir(), func() time.Time { return now }
	c = c.WithKey("k")
	if _, err := c.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err := c.Validate(context.Background())
	var re *RateLimitError
	if !errors.As(err, &re) {
		t.Fatalf("not a rate limit: %v", err)
	}
	if !re.Reset.After(now) {
		t.Fatalf("reset %v is not after now; Retry-After was ignored", re.Reset)
	}
}
