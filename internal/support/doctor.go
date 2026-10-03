package support

import (
	"encoding/json"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/doctor"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// Doctor runs the same environment checks as `mortar doctor`.
func (s *Service) Doctor() (doctor.Report, error) {
	dir, err := s.dataDir()
	if err != nil {
		return doctor.Report{}, err
	}
	cur := settings.Defaults()
	if b, readErr := fsx.ReadFile(filepath.Join(dir, "settings.json")); readErr == nil {
		_ = json.Unmarshal(b, &cur)
	}
	games, err := listedGames(s.home, cur)
	if err != nil {
		return doctor.Report{}, err
	}
	env := map[string]problems.Environment{}
	for _, g := range games {
		if s.env != nil {
			env[g.ID] = s.env(g.ID)
		}
	}
	live := doctor.FromLive(doctor.Live{
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

func listedGames(home string, cur settings.Settings) ([]game.GameInfo, error) {
	var out []game.GameInfo
	if g := game.Find("stardew"); g != nil {
		info := game.GameInfo{ID: g.ID(), Name: g.Name(), AppID: g.SteamAppID(), Loader: g.LoaderName(), Available: true}
		dir, store, all, err := game.Resolve(home, cur, g.ID())
		if err != nil {
			return nil, err
		}
		info.Installed, info.InstallDir, info.Store = dir != "", dir, store
		if all != nil {
			info.Installs = all
		}
		out = append(out, info)
	}
	out = append(out, game.GameInfo{ID: "lethal", Name: "Lethal Company", AppID: "1966720", Loader: "BepInEx 5"})
	return out, nil
}
