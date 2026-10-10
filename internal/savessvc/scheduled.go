package savessvc

import (
	"context"
	"errors"
	"log"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// scheduleCheck is how often Mortar looks whether a scheduled save backup is due; a run blocked by the game waits at
// most this long after the game closes.
const scheduleCheck = 10 * time.Minute

// scheduleRetry is the longest a failed pass (backups on an unplugged drive) waits before trying again.
const scheduleRetry = 30 * time.Minute

// RunScheduledBackups backs up changed saves on the saveBackupHours schedule until ctx ends.
//
//wails:ignore
func (s *Service) RunScheduledBackups(ctx context.Context) {
	tick := time.NewTicker(scheduleCheck)
	defer tick.Stop()
	s.scheduledTick(time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			s.scheduledTick(now)
		}
	}
}

// scheduledTick runs one pass per game when the interval since the game's last pass has passed and the game is not
// running. The first check after start measures from the newest scheduled backup on disk, so restarting Mortar does
// not reset a daily schedule.
func (s *Service) scheduledTick(now time.Time) {
	for _, id := range slices.Sorted(maps.Keys(s.scanners)) {
		s.scheduledTickFor(id, now)
	}
}

func (s *Service) scheduledTickFor(gameID string, now time.Time) {
	set := s.settings.Get()
	l, backupsDir, err := s.backupDirs(gameID)
	if err != nil {
		log.Printf("scheduled save backup: %v", err)
		return
	}
	// The game's shared saves follow the game's schedule; a profile that keeps its own saves follows its own, which
	// is the game's unless the profile overrides it.
	units := append([]profileSaves{{layout: l}}, s.ownSaves(gameID)...)
	var due []profileSaves
	for _, u := range units {
		hours, err := strconv.Atoi(settings.ResolveAt(set, "saveBackupHours", settings.Scope{Game: gameID, Profile: u.profile}, u.overrides))
		if err != nil || hours <= 0 {
			continue
		}
		u.interval = time.Duration(hours) * time.Hour
		last, err := s.lastScheduledAt(scheduleKey(gameID, u.profile), backupsDir, now)
		if err != nil {
			log.Printf("scheduled save backup: %v", err)
			return
		}
		if now.Sub(last) >= u.interval {
			due = append(due, u)
		}
	}
	if len(due) == 0 || s.gameBusy(gameID) {
		return
	}
	var run backup.Run
	var failed error
	for _, u := range due {
		keep, err := strconv.Atoi(settings.ResolveAt(set, "saveBackupKeep", settings.Scope{Game: gameID, Profile: u.profile}, u.overrides))
		if err != nil || keep < 1 {
			keep = backup.DefaultKeep
		}
		r, unitErr := backup.Scheduled(u.layout, backupsDir, u.profile, keep, now)
		run.Saved, run.Unchanged, run.Failed = run.Saved+r.Saved, run.Unchanged+r.Unchanged, run.Failed+r.Failed
		next := now
		if unitErr != nil || r.Failed > 0 {
			next = now.Add(min(u.interval, scheduleRetry) - u.interval)
		}
		s.schedMu.Lock()
		s.lastScheduled[scheduleKey(gameID, u.profile)] = next
		s.schedMu.Unlock()
		failed = errors.Join(failed, unitErr)
	}
	log.Printf("scheduled save backup: %d saved, %d unchanged, %d failed", run.Saved, run.Unchanged, run.Failed)
	if failed != nil {
		log.Printf("scheduled save backup: %v", failed)
	}
	if s.Emit != nil {
		ev := ScheduledRun{At: now.UnixMilli(), Saved: run.Saved, Unchanged: run.Unchanged, Failed: run.Failed}
		if failed != nil {
			ev.Error = failed.Error()
		}
		s.Emit(ScheduledEvent, ev)
	}
}

// scheduleKey names one schedule: the game's shared saves, or one profile's own.
func scheduleKey(gameID, profileID string) string {
	if profileID == "" {
		return gameID
	}
	return gameID + "\x00" + profileID
}

// ScheduledEvent is emitted after every scheduled backup pass with a ScheduledRun.
const ScheduledEvent = "saves:scheduled"

// ScheduledRun is what one scheduled backup pass did; Error joins the failures of the saves that did not back up.
type ScheduledRun struct {
	At        int64  `json:"at"`
	Saved     int    `json:"saved"`
	Unchanged int    `json:"unchanged"`
	Failed    int    `json:"failed"`
	Error     string `json:"error"`
}

// LastScheduledBackup is when the scheduled save backup last ran, in Unix milliseconds, or 0 when it has not.
func (s *Service) LastScheduledBackup() (int64, error) {
	var newest time.Time
	for id := range s.scanners {
		_, backupsDir, err := s.backupDirs(id)
		if err != nil {
			return 0, err
		}
		last, err := s.lastScheduledAt(id, backupsDir, time.Now())
		if err != nil {
			return 0, err
		}
		if last.After(newest) {
			newest = last
		}
	}
	if newest.IsZero() {
		return 0, nil
	}
	return newest.UnixMilli(), nil
}

// lastScheduledAt is when the last pass ran; before the first this run it is the newest scheduled backup on disk,
// so restarting Mortar does not reset a daily schedule.
func (s *Service) lastScheduledAt(gameID, backupsDir string, now time.Time) (time.Time, error) {
	s.schedMu.Lock()
	defer s.schedMu.Unlock()
	if s.lastScheduled[gameID].IsZero() {
		last, err := backup.LastScheduled(backupsDir, now)
		if err != nil {
			return time.Time{}, err
		}
		if s.lastScheduled == nil {
			s.lastScheduled = map[string]time.Time{}
		}
		s.lastScheduled[gameID] = last
	}
	return s.lastScheduled[gameID], nil
}

type profileSaves struct {
	profile   string
	layout    saves.Layout
	overrides map[string]string
	interval  time.Duration
}

// ownSaves are the saves folders of the game's profiles that keep their saves separate; the others share the folder
// the scheduled pass already backs up.
func (s *Service) ownSaves(gameID string) []profileSaves {
	if s.profiles == nil {
		return nil
	}
	all, err := s.profiles.List(gameID)
	if err != nil {
		log.Printf("scheduled save backup: %v", err)
		return nil
	}
	var out []profileSaves
	for _, p := range all {
		if !p.SeparateSaves {
			continue
		}
		sc, err := s.scannerFor(gameID, p.ID)
		if err != nil {
			log.Printf("scheduled save backup: %s: %v", p.ID, err)
			continue
		}
		out = append(out, profileSaves{profile: p.ID, layout: sc.Layout(), overrides: p.PrefOverrides()})
	}
	return out
}
