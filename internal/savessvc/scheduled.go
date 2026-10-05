package savessvc

import (
	"context"
	"log"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/backup"
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
	hours, err := strconv.Atoi(settings.Resolve(set, "saveBackupHours", gameID, nil))
	if err != nil || hours <= 0 {
		return
	}
	savesDir, backupsDir, err := s.backupDirs(gameID)
	if err != nil {
		log.Printf("scheduled save backup: %v", err)
		return
	}
	last, err := s.lastScheduledAt(gameID, backupsDir, now)
	if err != nil {
		log.Printf("scheduled save backup: %v", err)
		return
	}
	if now.Sub(last) < time.Duration(hours)*time.Hour {
		return
	}
	if s.gameBusy(gameID) {
		return
	}
	keep, err := strconv.Atoi(settings.Resolve(set, "saveBackupKeep", gameID, nil))
	if err != nil || keep < 1 {
		keep = backup.DefaultKeep
	}
	run, err := backup.Scheduled(savesDir, backupsDir, keep, now)
	interval := time.Duration(hours) * time.Hour
	s.schedMu.Lock()
	s.lastScheduled[gameID] = now
	if err != nil || run.Failed > 0 {
		s.lastScheduled[gameID] = now.Add(min(interval, scheduleRetry) - interval)
	}
	s.schedMu.Unlock()
	log.Printf("scheduled save backup: %d saved, %d unchanged, %d failed", run.Saved, run.Unchanged, run.Failed)
	if err != nil {
		log.Printf("scheduled save backup: %v", err)
	}
	if s.Emit != nil {
		ev := ScheduledRun{At: now.UnixMilli(), Saved: run.Saved, Unchanged: run.Unchanged, Failed: run.Failed}
		if err != nil {
			ev.Error = err.Error()
		}
		s.Emit(ScheduledEvent, ev)
	}
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
