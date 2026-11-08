package control

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestParseSinceTime(t *testing.T) {
	t.Parallel()
	utc := time.Date(2026, 3, 1, 12, 30, 5, 0, time.UTC)
	for _, tc := range []struct {
		raw     string
		want    time.Time
		wantErr bool
	}{
		{"2026-03-01T12:30:05Z", utc, false},
		{"2026-03-01T14:30:05+02:00", utc, false},
		{"2026-03-01T12:30:05.123456789Z", utc.Add(123456789 * time.Nanosecond), false},
		{"", time.Time{}, false},
		{"yesterday", time.Time{}, true},
		{"2h", time.Time{}, true},
		{"2026-13-45", time.Time{}, true},
	} {
		got, err := profile.ParseSince(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseSince(%q) succeeded, want error", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSince(%q) = %v, %v", tc.raw, got, err)
			continue
		}
		if !got.Equal(tc.want) {
			t.Errorf("ParseSince(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
	// A time without a zone keeps its wall clock, whichever zone the parser gives it.
	for raw, layout := range map[string]string{"2026-03-01 12:30:05": "2006-01-02 15:04:05", "2026-03-01 12:30": "2006-01-02 15:04"} {
		got, err := profile.ParseSince(raw)
		if err != nil || got.IsZero() || got.Format(layout) != raw {
			t.Errorf("ParseSince(%q) = %v, %v", raw, got, err)
		}
	}
}
