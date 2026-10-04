package templates

import (
	"fmt"
	"maps"
	"slices"
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
	out.SettingsChanges = append(out.SettingsChanges, launchDiff(p, t.LaunchConfig)...)
	return out, nil
}

// Undo is what UndoApplyTemplate needs to put the profile back: the history event holding its mods before the
// apply, and the game settings and launch options it replaced.
type Undo struct {
	EventID       string                `json:"eventId"`
	ModsChanged   bool                  `json:"modsChanged"`
	GameSettings  gamesettings.Settings `json:"gameSettings"`
	LaunchOptions string                `json:"launchOptions"`
	Launch        LaunchConfig          `json:"launch"`
}

// Applied is the result of ApplyTemplate with the means to undo it.
type Applied struct {
	bundles.ApplyResult
	Undo Undo `json:"undo"`
}

// ApplyTemplate merges the template into an existing profile: its mods are added (the profile's own are never
// removed) as one history event of their own, then its game settings and launch options replace the profile's.
// The result carries what UndoApplyTemplate needs to restore all of it.
func (s *Service) ApplyTemplate(game, templateName, profileID string) (Applied, error) {
	t, err := s.find(game, templateName)
	if err != nil {
		return Applied{}, err
	}
	p, err := s.profileOf(game, profileID)
	if err != nil {
		return Applied{}, err
	}
	undo := Undo{}
	if undo.GameSettings, err = s.d.GameSettings(game, profileID); err != nil {
		return Applied{}, err
	}
	if undo.LaunchOptions, err = s.d.Profiles.LaunchOptions(game, profileID); err != nil {
		return Applied{}, err
	}
	undo.Launch = launchOf(p)
	if undo.EventID, err = s.d.Profiles.Baseline(game, profileID); err != nil {
		return Applied{}, err
	}
	add, _, _ := split(p, t.Bundle)
	res, err := s.fill(game, profileID, t, add)
	if err != nil {
		return Applied{}, err
	}
	undo.ModsChanged = res.Added > 0
	return Applied{ApplyResult: res, Undo: undo}, nil
}

// UndoApplyTemplate reverts the mods an ApplyTemplate added and restores the game settings and launch options it
// replaced.
func (s *Service) UndoApplyTemplate(game, profileID string, u Undo) (profile.Profile, error) {
	if u.ModsChanged {
		if _, err := s.d.Profiles.Revert(game, profileID, u.EventID); err != nil {
			return profile.Profile{}, err
		}
	}
	if err := s.d.SetGameSettings(game, profileID, u.GameSettings); err != nil {
		return profile.Profile{}, err
	}
	if _, err := s.d.Profiles.SetLaunchOptions(game, profileID, u.LaunchOptions); err != nil {
		return profile.Profile{}, err
	}
	if _, err := s.d.Profiles.SetLaunchSettings(game, profileID, u.Launch.LaunchPrefix, u.Launch.LaunchEnv); err != nil {
		return profile.Profile{}, err
	}
	if _, err := s.d.Profiles.SetOverrides(game, profileID, u.Launch.Overrides); err != nil {
		return profile.Profile{}, err
	}
	return s.d.Profiles.SetLaunchPresets(game, profileID, u.Launch.LaunchPresets, u.Launch.DefaultLaunchPreset)
}

func launchOf(p profile.Profile) LaunchConfig {
	return LaunchConfig{
		LaunchPrefix: p.LaunchPrefix, LaunchEnv: p.LaunchEnv, Overrides: maps.Clone(p.Overrides),
		LaunchPresets: slices.Clone(p.LaunchPresets), DefaultLaunchPreset: p.DefaultLaunchPreset,
	}
}

func disabledMods(p profile.Profile) []string {
	out := []string{}
	for _, e := range p.Entries {
		if !e.Source.Bundled() {
			out = append(out, e.Disabled...)
		}
	}
	return out
}

// launchDiff names the launch settings the template would replace; a part the template leaves empty is not applied.
func launchDiff(have profile.Profile, want LaunchConfig) []string {
	var out []string
	for _, c := range []struct {
		label   string
		applies bool
		differs bool
	}{
		{"launch prefix", want.LaunchPrefix != "", have.LaunchPrefix != want.LaunchPrefix},
		{"launch environment", want.LaunchEnv != "", have.LaunchEnv != want.LaunchEnv},
		{"setting overrides", len(want.Overrides) > 0, !maps.Equal(have.Overrides, want.Overrides)},
		{"launch presets", len(want.LaunchPresets) > 0, !slices.Equal(have.LaunchPresets, want.LaunchPresets) || have.DefaultLaunchPreset != want.DefaultLaunchPreset},
	} {
		if c.applies && c.differs {
			out = append(out, c.label)
		}
	}
	return out
}

func (s *Service) fill(game, id string, t Template, mods []bundles.Mod) (bundles.ApplyResult, error) {
	mods = slices.Clone(mods)
	for i := range mods {
		mods[i].Source = mods[i].Source.WithDisabled(t.Disabled)
	}
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
	if t.LaunchPrefix != "" || t.LaunchEnv != "" {
		if res.Profile, err = s.d.Profiles.SetLaunchSettings(game, id, t.LaunchPrefix, t.LaunchEnv); err != nil {
			return bundles.ApplyResult{}, err
		}
	}
	if len(t.Overrides) > 0 {
		if res.Profile, err = s.d.Profiles.SetOverrides(game, id, t.Overrides); err != nil {
			return bundles.ApplyResult{}, err
		}
	}
	if len(t.LaunchPresets) > 0 {
		if res.Profile, err = s.d.Profiles.SetLaunchPresets(game, id, t.LaunchPresets, t.DefaultLaunchPreset); err != nil {
			return bundles.ApplyResult{}, err
		}
	}
	return res, nil
}
