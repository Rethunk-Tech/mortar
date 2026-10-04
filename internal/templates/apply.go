package templates

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/bundles"
	"github.com/Rethunk-AI/mortar/internal/gamesettings"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

// Preview is what applying a template to a profile would change.
type Preview struct {
	Add             []string `json:"add"`
	AlreadyHave     []string `json:"alreadyHave"`
	VersionDiffers  []string `json:"versionDiffers"`
	SettingsChanges []string `json:"settingsChanges"`
	// Missing are the mods in Add the store does not hold; applying them needs a download.
	Missing []string `json:"missing"`
}

func (s *Service) find(game, name string) (Template, error) {
	s.mu.Lock()
	list, err := s.read(game)
	s.mu.Unlock()
	if err != nil {
		return Template{}, err
	}
	i := indexOf(list, name)
	if i < 0 {
		return Template{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("template %q was not found", name))
	}
	return list[i], nil
}

// split sorts the template's mods against what the profile holds: same entry is already there, same mod id under
// another entry is a different version and is left alone, anything else is added.
func split(p profile.Profile, mods []bundles.Mod) (add, have, differs []bundles.Mod) {
	keys := map[string]bool{}
	ids := map[string]bool{}
	for _, e := range p.Entries {
		keys[e.Key] = true
		for _, m := range e.Mods {
			ids[manifest.FoldID(m.UniqueID)] = true
		}
	}
	for _, m := range mods {
		switch {
		case keys[m.EntryKey]:
			have = append(have, m)
		case ids[manifest.FoldID(m.UniqueID)]:
			differs = append(differs, m)
		default:
			add = append(add, m)
		}
	}
	return add, have, differs
}

func names(mods []bundles.Mod) []string {
	out := make([]string, 0, len(mods))
	for _, m := range mods {
		out = append(out, m.Name)
	}
	return out
}

func (s *Service) profileOf(game, id string) (profile.Profile, error) {
	all, err := s.d.Profiles.List(game)
	if err != nil {
		return profile.Profile{}, err
	}
	for _, p := range all {
		if p.ID == id {
			return p, nil
		}
	}
	return profile.Profile{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("profile %q was not found", id))
}

func settingsDiff(have, want gamesettings.Settings) []string {
	var out []string
	for _, c := range []struct {
		label string
		same  bool
	}{
		{"window mode", ptrEq(have.WindowMode, want.WindowMode)},
		{"display", ptrEq(have.DisplayIndex, want.DisplayIndex)},
		{"resolution", ptrEq(have.PreferredResolutionX, want.PreferredResolutionX) && ptrEq(have.PreferredResolutionY, want.PreferredResolutionY)},
		{"fullscreen resolution", ptrEq(have.FullscreenResolutionX, want.FullscreenResolutionX) && ptrEq(have.FullscreenResolutionY, want.FullscreenResolutionY)},
		{"zoom level", ptrEq(have.ZoomLevel, want.ZoomLevel)},
		{"UI scale", ptrEq(have.UIScale, want.UIScale)},
		{"start muted", ptrEq(have.StartMuted, want.StartMuted)},
		{"music volume", ptrEq(have.MusicVolumeLevel, want.MusicVolumeLevel)},
		{"sound volume", ptrEq(have.SoundVolumeLevel, want.SoundVolumeLevel)},
	} {
		if !c.same {
			out = append(out, c.label)
		}
	}
	return out
}

func ptrEq[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// PreviewApplyTemplate reports what ApplyTemplate would do to the profile without changing anything.
func (s *Service) PreviewApplyTemplate(game, templateName, profileID string) (Preview, error) {
	t, err := s.find(game, templateName)
	if err != nil {
		return Preview{}, err
	}
	p, err := s.profileOf(game, profileID)
	if err != nil {
		return Preview{}, err
	}
	add, have, differs := split(p, t.Bundle)
	out := Preview{Add: names(add), AlreadyHave: names(have), VersionDiffers: names(differs), SettingsChanges: []string{}, Missing: []string{}}
	for _, m := range add {
		if !s.d.Profiles.HasStoreItem(game, m.EntryKey) {
			out.Missing = append(out.Missing, m.Name)
		}
	}
	cur, err := s.d.GameSettings(game, profileID)
	if err != nil {
		return Preview{}, err
	}
	out.SettingsChanges = append(out.SettingsChanges, settingsDiff(cur, t.GameSettings)...)
	if opts, err := s.d.Profiles.LaunchOptions(game, profileID); err != nil {
		return Preview{}, err
	} else if t.LaunchOptions != "" && strings.TrimSpace(opts) != strings.TrimSpace(t.LaunchOptions) {
		out.SettingsChanges = append(out.SettingsChanges, "launch options")
	}
	return out, nil
}

// ApplyTemplate merges the template into an existing profile: its mods are added (the profile's own are never
// removed) as one history event, then its game settings and launch options replace the profile's. Settings are not
// part of history, so undoing the event reverts the mods only.
func (s *Service) ApplyTemplate(game, templateName, profileID string) (bundles.ApplyResult, error) {
	t, err := s.find(game, templateName)
	if err != nil {
		return bundles.ApplyResult{}, err
	}
	p, err := s.profileOf(game, profileID)
	if err != nil {
		return bundles.ApplyResult{}, err
	}
	add, _, _ := split(p, t.Bundle)
	return s.fill(game, profileID, t, add)
}

func (s *Service) fill(game, id string, t Template, mods []bundles.Mod) (bundles.ApplyResult, error) {
	res, err := s.d.Bundles.ApplyMods(game, "template", mods, id)
	if err != nil {
		return bundles.ApplyResult{}, err
	}
	if err := s.d.SetGameSettings(game, id, t.GameSettings); err != nil {
		return bundles.ApplyResult{}, err
	}
	if t.LaunchOptions != "" {
		if res.Profile, err = s.d.Profiles.SetLaunchOptions(game, id, t.LaunchOptions); err != nil {
			return bundles.ApplyResult{}, err
		}
	}
	return res, nil
}
