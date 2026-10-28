package control

import (
	"testing"
	"time"
)

func TestParseSinceTime(t *testing.T) {
	t.Parallel()
	utc := time.Date(2026, 3, 1, 12, 30, 5, 0, time.UTC)
	for _, tc := range []struct {
		raw  string
		want time.Time
	}{
		{"2026-03-01T12:30:05Z", utc},
		{"2026-03-01T14:30:05+02:00", utc},
		{"2026-03-01T12:30:05.123456789Z", utc.Add(123456789 * time.Nanosecond)},
		{"", time.Time{}},
		{"yesterday", time.Time{}},
		{"2h", time.Time{}},
		{"2026-13-45", time.Time{}},
	} {
		if got := parseSinceTime(tc.raw); !got.Equal(tc.want) {
			t.Errorf("parseSinceTime(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
	// A time without a zone keeps its wall clock, whichever zone the parser gives it.
	for raw, layout := range map[string]string{"2026-03-01 12:30:05": "2006-01-02 15:04:05", "2026-03-01 12:30": "2006-01-02 15:04"} {
		got := parseSinceTime(raw)
		if got.IsZero() || got.Format(layout) != raw {
			t.Errorf("parseSinceTime(%q) = %v", raw, got)
		}
	}
}
