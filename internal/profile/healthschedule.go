package profile

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
)

const (
	healthCheckFile = "health-check.json"
	// healthCheckEvery is how often each profile is checked in the background.
	healthCheckEvery = 7 * 24 * time.Hour
	// healthTick is how often RunHealthChecks looks for a due check, and how long it waits for startup to settle
	// when no problem check says so first.
	healthTick = 10 * time.Minute
	// HealthEvent is emitted with a HealthNotice after every health check.
	HealthEvent = "profile:health"
)

type healthCheck struct {
	At       time.Time `json:"at"`
	Findings int       `json:"findings"`
	Build    string    `json:"build"`
}

// HealthNotice says how many findings a profile's latest health check had.
type HealthNotice struct {
	Game     string `json:"game"`
	Profile  string `json:"profile"`
	Findings int    `json:"findings"`
}

func healthCheckPath(dir string) string { return filepath.Join(dir, healthCheckFile) }

// readHealthCheck reads the last check, as never made when another build made it: that build's logic may have found
// what this one does not, so its count is neither shown nor trusted to postpone the next check.
func readHealthCheck(dir string) (healthCheck, bool) {
	b, err := fsx.ReadFile(healthCheckPath(dir))
	if err != nil {
		return healthCheck{}, false
	}
	var c healthCheck
	if json.Unmarshal(b, &c) != nil || c.Build != appversion.Build() {
		return healthCheck{}, false
	}
	return c, true
}

// gameWide reports whether a finding is about the game's store or launch journals, not one profile.
func gameWide(f HealthFinding) bool { return f.Kind == HealthUnused || f.Kind == HealthJournal }

// recordHealth stores a check's findings: the profile's own in its folder, the game-wide ones once in the game's.
// The game-wide count shows on one profile (the first), so a stale store item is not a badge on every profile.
func (s *Service) recordHealth(game, id string, findings []HealthFinding, now time.Time) {
	own, wide := 0, 0
	for _, f := range findings {
		if gameWide(f) {
			wide++
		} else {
			own++
		}
	}
	dir, err := s.store.ProfileDir(game, id)
	if err != nil {
		return
	}
	if err := datadir.WriteJSON(healthCheckPath(dir), healthCheck{At: now, Findings: own, Build: appversion.Build()}); err != nil {
		log.Printf("health check: %v", err)
	}
	if gdir, err := s.store.gameDir(game); err == nil {
		if err := datadir.WriteJSON(healthCheckPath(gdir), healthCheck{At: now, Findings: wide, Build: appversion.Build()}); err != nil {
			log.Printf("health check: %v", err)
		}
	}
	if s.Emit != nil {
		shown := own
		if s.badgeCarrier(game) == id {
			shown += wide
		}
		s.Emit(HealthEvent, HealthNotice{Game: game, Profile: id, Findings: shown})
	}
}

// badgeCarrier is the profile whose badge includes the game-wide findings.
func (s *Service) badgeCarrier(game string) string {
	list, err := s.store.listOK(game)
	if err != nil || len(list) == 0 {
		return ""
	}
	return list[0].ID
}

// HealthBadges maps each of the game's profiles whose latest health check found something to how many findings.
func (s *Service) HealthBadges(game string) (map[string]int, error) {
	list, err := s.store.listOK(game)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, p := range list {
		dir, err := s.store.ProfileDir(game, p.ID)
		if err != nil {
			continue
		}
		if c, ok := readHealthCheck(dir); ok && c.Findings > 0 {
			out[p.ID] = c.Findings
		}
	}
	if gdir, err := s.store.gameDir(game); err == nil && len(list) > 0 {
		if c, ok := readHealthCheck(gdir); ok && c.Findings > 0 {
			out[list[0].ID] += c.Findings
		}
	}
	return out, nil
}

// FlagHealth makes the next background pass check every profile of these games, whatever their last check.
//
//wails:ignore
func (s *Service) FlagHealth(games ...string) {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if s.healthDue == nil {
		s.healthDue = map[string]bool{}
	}
	for _, g := range games {
		s.healthDue[g] = true
	}
}

// RunHealthChecks checks each profile quietly once a week until ctx ends. It does nothing before settled closes (or
// healthTick passes), so it adds no work to startup.
//
//wails:ignore
func (s *Service) RunHealthChecks(ctx context.Context, settled <-chan struct{}) {
	wait := time.NewTimer(healthTick)
	defer wait.Stop()
	select {
	case <-ctx.Done():
		return
	case <-settled:
	case <-wait.C:
	}
	tick := time.NewTicker(healthTick)
	defer tick.Stop()
	s.healthPass(ctx, time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			s.healthPass(ctx, now)
		}
	}
}

// healthPass checks, one at a time, the profiles whose last check is a week old or whose game was flagged, leaving
// every game that is running alone; a flag stays until its game's profiles have all been checked.
func (s *Service) healthPass(ctx context.Context, now time.Time) {
	for _, g := range game.Implemented() {
		if s.store.AnyRunning(g) || (s.store.GameRunning != nil && s.store.GameRunning(g)) {
			continue
		}
		s.healthMu.Lock()
		flagged := s.healthDue[g]
		s.healthMu.Unlock()
		list, err := s.store.listOK(g)
		if err != nil {
			continue
		}
		for _, p := range list {
			if ctx.Err() != nil {
				return
			}
			dir, err := s.store.ProfileDir(g, p.ID)
			if err != nil {
				continue
			}
			if c, ok := readHealthCheck(dir); !flagged && ok && now.Sub(c.At) < healthCheckEvery {
				continue
			}
			if _, err := s.ProfileHealth(g, p.ID); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Printf("health check %s/%s: %v", g, p.ID, err)
			}
		}
		if flagged {
			s.healthMu.Lock()
			delete(s.healthDue, g)
			s.healthMu.Unlock()
		}
	}
}
