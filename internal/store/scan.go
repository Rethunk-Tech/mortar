package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/avscan"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// scanTimeout bounds one item's scan; a scan that has not finished by then is an error, not a detection.
const scanTimeout = 2 * time.Minute

// DetectedError is an item the antivirus flagged before it entered the store.
type DetectedError struct {
	Game string `json:"game"`
	Key  string `json:"key"`
	avscan.Detection
}

// Detail is what the GUI reads off the error (usererr.Detailer): which store item, and what was flagged in it.
func (e *DetectedError) Detail() any { return e }

func (e *DetectedError) Error() string {
	if e.File == "" {
		return fmt.Sprintf("the antivirus (%s) reports %s", e.Scanner, e.Name)
	}
	return fmt.Sprintf("the antivirus (%s) reports %s in %s", e.Scanner, e.Name, e.File)
}

// scanning is the store's antivirus hook: the scanner chosen at the time of each scan, where a failed scan is
// reported, and the items the player chose to install despite a detection.
type scanning struct {
	mu      sync.Mutex
	scanner func() avscan.Scanner
	failed  func(game, key string, err error)
	allowed map[string]bool
}

// SetScanner makes every item that enters the store pass through the scanner pick returns, which is asked each time so
// a changed setting applies at once. failed hears about a scan that could not finish; the item installs anyway.
func (s *Store) SetScanner(pick func() avscan.Scanner, failed func(game, key string, err error)) {
	s.scan.mu.Lock()
	defer s.scan.mu.Unlock()
	s.scan.scanner, s.scan.failed = pick, failed
}

// AllowUnscanned lets the next install of key through without a scan, once: the player's answer to a detection.
func (s *Store) AllowUnscanned(game, key string) {
	s.scan.mu.Lock()
	defer s.scan.mu.Unlock()
	if s.scan.allowed == nil {
		s.scan.allowed = map[string]bool{}
	}
	s.scan.allowed[game+"\x00"+key] = true
}

// checkExtracted scans the folder an item was extracted into. A detection is returned as a malware error; a scan that
// fails or finds no scanner lets the item in. Loader bundles are Mortar's own downloads and are not scanned.
func (s *Store) checkExtracted(ctx context.Context, game, key, dir string) error {
	s.scan.mu.Lock()
	pick, failed := s.scan.scanner, s.scan.failed
	allowed := s.scan.allowed[game+"\x00"+key]
	delete(s.scan.allowed, game+"\x00"+key)
	s.scan.mu.Unlock()
	if pick == nil || allowed {
		return nil
	}
	if _, _, loaderBundle := LoaderOf(key); loaderBundle {
		return nil
	}
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, scanTimeout)
	defer cancel()
	hit, found, err := pick().Scan(ctx, dir)
	switch {
	case err == nil && !found:
		return nil
	case ctx.Err() != nil && parent.Err() != nil:
		return parent.Err()
	case err != nil:
		if failed != nil && !isNoScanner(err) {
			failed(game, key, err)
		}
		return nil
	}
	return usererr.Wrap(usererr.Malware, &DetectedError{Game: game, Key: key, Detection: hit})
}

func isNoScanner(err error) bool { return errors.Is(err, avscan.ErrNoScanner) }
