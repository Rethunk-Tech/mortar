package problems

import (
	"bufio"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/iniedit"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// KindGameSetting is the LoadFailure kind of a setting in the game's own files that its mods need switched on.
const KindGameSetting = "game-setting"

// settingFailures reads an ini-style file's `key = value` lines (sections, blank lines and ; or # comments ignored)
// and reports each required setting the file holds with another value. A key the file lacks is the game's default and
// is not reported.
func settingFailures(content string, required []components.RequiredSetting) []LoadFailure {
	have := map[string]string{}
	section := ""
	sc := bufio.NewScanner(strings.NewReader(strings.TrimPrefix(content, "\ufeff")))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			section = strings.ToLower(strings.TrimSpace(strings.Trim(line, "[]")))
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			k = strings.ToLower(strings.TrimSpace(k))
			v = strings.TrimSpace(v)
			have[section+"\x00"+k] = v
			if _, seen := have["\x00"+k]; !seen {
				have["\x00"+k] = v
			}
		}
	}
	var out []LoadFailure
	for _, r := range required {
		// A requirement without a section matches the key in the first section that holds it.
		if v, ok := have[strings.ToLower(r.Section)+"\x00"+strings.ToLower(r.Key)]; ok && !strings.EqualFold(v, r.Value) {
			out = append(out, LoadFailure{Plugin: r.Key, Kind: KindGameSetting, Message: r.Message})
		}
	}
	return out
}

// gameSettingFailures checks the game's required settings in the files its catalog entry points at. A file that cannot
// be read is not a finding, and nothing is ever written. The profile's own gameSettingsMode decides whether Mortar
// fixes the options file at launch.
func (s *Service) gameSettingFailures(gameID, profileID string) []LoadFailure {
	info, ok := components.Game(gameID)
	if !ok || len(info.RequiredSettings) == 0 || s.settings == nil {
		return nil
	}
	var out []LoadFailure
	set := s.settings.Get()
	mode := settings.ResolveAt(set, "gameSettingsMode", settings.Scope{Game: gameID, Profile: profileID}, s.profileOverrides(gameID, profileID))
	for role, rs := range groupByPath(info.RequiredSettings) {
		path, err := game.PathFor(s.home, s.settings.Get(), gameID, "", role)
		if err != nil {
			continue
		}
		b, err := fsx.ReadFile(path)
		if err != nil {
			continue
		}
		if refused := iniedit.Check(string(b)); refused != nil {
			// Mortar leaves a file it cannot edit alone at launch, so the settings are the player's to switch on.
			for _, r := range rs {
				out = append(out, LoadFailure{Plugin: r.Key, Kind: KindGameSetting, Message: r.Message + " Mortar cannot edit this file: " + refused.Error() + "."})
			}
			continue
		}
		// In edit mode Mortar writes the values into the profile's copy at launch, so the player's own file is not a finding.
		if role == "options" && mode == settings.GameSettingsEdit {
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

func (s *Service) profileOverrides(gameID, profileID string) map[string]string {
	if s.profiles == nil || profileID == "" {
		return nil
	}
	all, err := s.profiles.List(gameID)
	if err != nil {
		return nil
	}
	for _, p := range all {
		if p.ID == profileID {
			return p.PrefOverrides()
		}
	}
	return nil
}
