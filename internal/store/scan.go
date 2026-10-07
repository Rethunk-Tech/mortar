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
	// Removed is set when the antivirus deleted the file itself: there is nothing left to install anyway.
	Removed bool `json:"removed"`
	avscan.Detection
}

// Detail is what the GUI reads off the error (usererr.Detailer): which store item, and what was flagged in it.
func (e *DetectedError) Detail() any { return e }

func (e *DetectedError) Error() string {
	if e.Removed {
		return "Windows removed the file as malware without saying which; restore it from Windows Security's protection history if you trust it"
	}
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
	allowed map[string]Override
	// applied holds the overrides whose scan was skipped, until the install that used them succeeds or fails.
	applied map[string]Override
}

// Override is the player's "Install anyway" for one store item: the profile it goes into, the mod's name and what the
// antivirus flagged, kept for the history entry that records the override once the item has been installed.
type Override struct {
	Profile, Name, Detection string
}

// SetScanner makes every item that enters the store pass through the scanner pick returns, which is asked each time so
// a changed setting applies at once. failed hears about a scan that could not finish; the item installs anyway.
func (s *Store) SetScanner(pick func() avscan.Scanner, failed func(game, key string, err error)) {
	s.scan.mu.Lock()
	defer s.scan.mu.Unlock()
	s.scan.scanner, s.scan.failed = pick, failed
}

// AllowUnscanned lets the next install of key through without a scan, once: the player's answer to a detection.
func (s *Store) AllowUnscanned(game, key string, ov Override) {
	s.scan.mu.Lock()
	defer s.scan.mu.Unlock()
	if s.scan.allowed == nil {
		s.scan.allowed = map[string]Override{}
	}
	s.scan.allowed[game+"\x00"+key] = ov
}

// TakeOverride returns the override whose scan was skipped for key, once: the caller records it when the item is in.
func (s *Store) TakeOverride(game, key string) (Override, bool) {
	s.scan.mu.Lock()
	defer s.scan.mu.Unlock()
	ov, ok := s.scan.applied[game+"\x00"+key]
	delete(s.scan.applied, game+"\x00"+key)
	return ov, ok
}

func (s *Store) dropOverride(game, key string) {
	s.scan.mu.Lock()
	defer s.scan.mu.Unlock()
	delete(s.scan.applied, game+"\x00"+key)
}

// checkExtracted scans the folder an item was extracted into. A detection is returned as a malware error; a scan that
// fails or finds no scanner lets the item in. Loader bundles are Mortar's own downloads and are not scanned.
func (s *Store) checkExtracted(ctx context.Context, game, key, dir string) error {
	s.scan.mu.Lock()
	pick, failed := s.scan.scanner, s.scan.failed
	ov, allowed := s.scan.allowed[game+"\x00"+key]
	delete(s.scan.allowed, game+"\x00"+key)
	if allowed {
		if s.scan.applied == nil {
			s.scan.applied = map[string]Override{}
		}
		s.scan.applied[game+"\x00"+key] = ov
	}
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
