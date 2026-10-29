package github

import (
	"context"
	"errors"
	"net/http"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-httpkit/download"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/bytefmt"
)

// Limiter is the bucket every download shares, so the speed limit holds across them together; main sets its Rate
// (zero or nil is unlimited) and it is read on every chunk.
var Limiter = &download.Limiter{}

// errNoResume is what a player sees when a server keeps answering with a range that cannot resume.
var errNoResume = errors.New("the download server sent a range Mortar could not resume")

// tooLargeError reads as the cap and matches safefs.ErrTooLarge.
type tooLargeError string

func (e tooLargeError) Error() string { return string(e) }
func (tooLargeError) Unwrap() error   { return safefs.ErrTooLarge }

// Download streams url to dest, appending with Range when a partial and the server allow it, and failing when the
// body exceeds limit bytes (which must be positive).
func Download(ctx context.Context, hc *http.Client, url, dest string, limit int64, progress download.Progress) error {
	err := download.File(ctx, hc, url, dest, download.Options{
		Limit:     limit,
		Progress:  progress,
		Limiter:   Limiter,
		Authorize: DefaultAuth.Apply,
		Check:     gh.Observe,
	})
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return &RateLimitError{Reset: rl.Reset}
	}
	switch {
	case errors.Is(err, safefs.ErrTooLarge):
		return tooLargeError("larger than " + bytefmt.Format(limit))
	case errors.Is(err, download.ErrNoResume):
		return errNoResume
	}
	return err
}
