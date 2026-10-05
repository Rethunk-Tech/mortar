package support

import (
	"encoding/json"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/doctor"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// Doctor runs the same environment checks as `mortar doctor`.
func (s *Service) Doctor() (doctor.Report, error) {
	dir, err := s.dataDir()
	if err != nil {
		return doctor.Report{}, err
	}
	cur := settings.Defaults()
	if b, readErr := fsx.ReadFile(filepath.Join(dir, settings.FileName)); readErr == nil {
		_ = json.Unmarshal(b, &cur)
	}
	games, err := game.List(s.home, cur)
	if err != nil {
		return doctor.Report{}, err
	}
	env := map[string]problems.Environment{}
	for _, g := range games {
		if s.env != nil {
			env[g.ID] = s.env(g.ID)
		}
	}
	live := doctor.LiveWithNativeHosts(doctor.Live{
		Version:        s.version,
		CommandVersion: s.version,
		DataDir:        dir,
		Games:          games,
		Environment:    env,
		NxmHandled:     cur.NxmHandled,
		NxmPrevious:    cur.NxmPreviousName,
	})
	disk := doctor.Scan(dir)
	return doctor.Report{Checks: append(live.Checks, disk.Checks...)}, nil
}
