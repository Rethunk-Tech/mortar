// Package nexus holds what Mortar decides on top of go-nexusmods: which file an update means, which mirror to fetch,
// what the player has seen, the cached mod page, and the wording of a rate limit.
package nexus

import (
	"errors"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
)

// ErrQuarantined means Nexus will not allow the file to be downloaded.
var ErrQuarantined = errors.New("nexus file quarantined")

// RateLimitError is a refused or rejected call; Reset is when the exhausted window renews.
type RateLimitError struct{ Reset time.Time }

func (e *RateLimitError) Error() string {
	if e.Reset.IsZero() {
		return "nexus rate limit reached"
	}
	return "nexus rate limit reached until " + e.Reset.Format(time.RFC3339)
}

// Unwrap lets errors.As find the module's *httpkit.RateLimitError behind Nexus's wording.
func (e *RateLimitError) Unwrap() error { return &httpkit.RateLimitError{Reset: e.Reset} }

// Worded gives a rate-limit refusal from go-nexusmods Mortar's wording; any other error passes through.
func Worded(err error) error {
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return &RateLimitError{Reset: rl.Reset}
	}
	return err
}
