package savessvc

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backup"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// scheduleCheck is how often Mortar looks whether a scheduled save backup is due; a run blocked by the game waits at
// most this long after the game closes.
const scheduleCheck = 10 * time.Minute

// RunScheduledBackups backs up changed saves on the saveBackupHours schedule until ctx ends.
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

// scheduledTick runs one pass when the interval since the last pass has passed and the game is not running. The
// first check after start measures from the newest scheduled backup on disk, so restarting Mortar does not reset a
// daily schedule.
func (s *Service) scheduledTick(now time.Time) {
	set := s.settings.Get()
	hours, err := strconv.Atoi(settings.Resolve(set, "saveBackupHours", settings.GameStardew, nil))
	if err != nil || hours <= 0 {
		return
	}
	savesDir, backupsDir, err := s.backupDirs()
	if err != nil {
		log.Printf("scheduled save backup: %v", err)
		return
	}
	if s.lastScheduled.IsZero() {
		if s.lastScheduled, err = backup.LastScheduled(backupsDir); err != nil {
			log.Printf("scheduled save backup: %v", err)
			return
		}
	}
	if now.Sub(s.lastScheduled) < time.Duration(hours)*time.Hour {
		return
	}
	if s.gameBusy() {
		return
	}
	keep, err := strconv.Atoi(settings.Resolve(set, "saveBackupKeep", settings.GameStardew, nil))
	if err != nil || keep < 1 {
		keep = backup.DefaultKeep
	}
	run, err := backup.Scheduled(savesDir, backupsDir, keep, now)
	s.lastScheduled = now
	log.Printf("scheduled save backup: %d saved, %d unchanged, %d failed", run.Saved, run.Unchanged, run.Failed)
	if err != nil {
		log.Printf("scheduled save backup: %v", err)
	}
}
