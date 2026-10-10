package problems

import (
	"bufio"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// KindGameSetting is the LoadFailure kind of a setting in the game's own files that its mods need switched on.
const KindGameSetting = "game-setting"

// settingFailures reads an ini-style file's `key = value` lines (sections, blank lines and ; or # comments ignored)
// and reports each required setting the file holds with another value. A key the file lacks is the game's default and
// is not reported.
func settingFailures(content string, required []components.RequiredSetting) []LoadFailure {
	have := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == ';' || line[0] == '#' || line[0] == '[' {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			have[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
		}
	}
	var out []LoadFailure
	for _, r := range required {
		if v, ok := have[strings.ToLower(r.Key)]; ok && !strings.EqualFold(v, r.Value) {
			out = append(out, LoadFailure{Plugin: r.Key, Kind: KindGameSetting, Message: r.Message})
		}
	}
	return out
}

// gameSettingFailures checks the game's required settings in the files its catalog entry points at. A file that cannot
// be read is not a finding, and nothing is ever written.
func (s *Service) gameSettingFailures(gameID string) []LoadFailure {
	info, ok := components.Game(gameID)
	if !ok || len(info.RequiredSettings) == 0 || s.settings == nil {
		return nil
	}
	var out []LoadFailure
	set := s.settings.Get()
	for role, rs := range groupByPath(info.RequiredSettings) {
		// In edit mode Mortar writes the options file's values into the profile's copy at launch, so the player's own
		// file is not a finding.
		if role == "options" && settings.ResolveAt(set, "gameSettingsMode", settings.Scope{Game: gameID}, nil) == settings.GameSettingsEdit {
			continue
		}
		path, err := game.PathFor(s.home, s.settings.Get(), gameID, "", role)
		if err != nil {
			continue
		}
		b, err := fsx.ReadFile(path)
		if err != nil {
			continue
		}
		out = append(out, settingFailures(string(b), rs)...)
	}
	return out
}

func groupByPath(in []components.RequiredSetting) map[string][]components.RequiredSetting {
	out := map[string][]components.RequiredSetting{}
	for _, r := range in {
		out[r.Path] = append(out[r.Path], r)
	}
	return out
}
