package source

import (
	"context"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/github"
)

type cutoffKey struct{}

// CutoffLayout is how a profile's "mod versions up to" date is written.
const CutoffLayout = "2006-01-02"

// ValidCutoff reports whether s is a date in CutoffLayout.
func ValidCutoff(s string) bool {
	_, err := time.Parse(CutoffLayout, s)
	return err == nil
}

// ParseCutoff reads a YYYY-MM-DD date as the end of that day where the player is, so a version released on the date
// still counts and one from the next local day does not. An empty or unreadable value is no cutoff.
func ParseCutoff(s string) time.Time {
	t, err := time.ParseInLocation(CutoffLayout, s, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t.AddDate(0, 0, 1)
}

// WithCutoff makes version picks under ctx skip versions released after cutoff; a zero cutoff changes nothing.
func WithCutoff(ctx context.Context, cutoff time.Time) context.Context {
	if cutoff.IsZero() {
		return ctx
	}
	return context.WithValue(ctx, cutoffKey{}, cutoff)
}

// CutoffOf is the cutoff WithCutoff set, zero when none.
func CutoffOf(ctx context.Context) time.Time {
	t, _ := ctx.Value(cutoffKey{}).(time.Time)
	return t
}

// Within keeps the items released before cutoff. An item whose release date is unknown (ok false), like every item
// when cutoff is zero, is kept: a source that gives no dates is left unfiltered.
func Within[T any](items []T, cutoff time.Time, released func(T) (t time.Time, ok bool)) []T {
	if cutoff.IsZero() {
		return items
	}
	out := make([]T, 0, len(items))
	for _, it := range items {
		if t, ok := released(it); !ok || t.Before(cutoff) {
			out = append(out, it)
		}
	}
	return out
}

// ReleasesWithin is Within for GitHub releases, dated by their publish time.
func ReleasesWithin(all []github.Release, cutoff time.Time) []github.Release {
	return Within(all, cutoff, func(r github.Release) (time.Time, bool) {
		t, err := time.Parse(time.RFC3339, r.Published)
		return t, err == nil
	})
}
