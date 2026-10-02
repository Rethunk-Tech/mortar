package problems

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/game/stardew"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

// Service exposes the problem checks to the frontend.
type Service struct {
	home     string
	settings *settings.Store
	profiles *profile.Store
	meta     Meta
	Runs     RunReader

	mu      sync.Mutex
	cache   map[string]cached
	updates map[string]cachedUpdates
}

type cachedUpdates struct {
	fingerprint string
	at          time.Time
	result      UpdatesResult
}

// cached is a result with the fingerprint of the mods and environment it was computed for.
type cached struct {
	fingerprint string
	result      Result
}

func NewService(home string, s *settings.Store, profiles *profile.Store, m *meta.Client) *Service {
	return &Service{home: home, settings: s, profiles: profiles, meta: m, cache: map[string]cached{}, updates: map[string]cachedUpdates{}}
}

func platform() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "Mac"
	}
	return "Linux"
}

// Environment reads the versions from the game's logs; without an install they stay empty.
func (s *Service) Environment(id string) Environment {
	env := Environment{Platform: platform()}
	g := game.Find(id)
	if g == nil {
		return env
	}
	set := s.settings.Get()
	dir, err := game.InstallDir(s.home, set, id)
	if err != nil || dir == "" {
		return env
	}
	st := g.LoaderStatus(dir, set.Loaders[id])
	env.GameVersion, env.APIVersion = st.GameVersion, st.Version
	return env
}

func fingerprint(env Environment, mods []Installed, runID string) string {
	var b strings.Builder
	b.WriteString(env.GameVersion + "|" + env.APIVersion + "|run:" + runID)
	for _, m := range mods {
		b.WriteString("\n" + m.Key + "|" + m.UniqueID + "|" + m.Version + "|" + m.Name)
		if m.Enabled {
			b.WriteString("|on")
		}
		if m.Folder != "" {
			if info, err := os.Stat(filepath.Join(m.Folder, "content.json")); err == nil {
				b.WriteString("|" + strconv.FormatInt(info.ModTime().UnixNano(), 10))
			}
			if info, err := os.Stat(filepath.Join(m.Folder, "config.json")); err == nil {
				b.WriteString("|config:" + strconv.FormatInt(info.ModTime().UnixNano(), 10))
			}
		}
		for _, d := range m.Dependencies {
			b.WriteString("|" + d.UniqueID + ">=" + d.MinimumVersion)
		}
	}
	return b.String()
}

func (s *Service) installed(gameID, id string) ([]Installed, error) {
	installed, err := s.profiles.Installed(gameID, id)
	if err != nil {
		return nil, err
	}
	mods := make([]Installed, len(installed))
	for i, m := range installed {
		mods[i] = Installed{
			Key: m.Key, SourceKind: m.Source.Kind, SourceVersion: m.Source.Version, Enabled: m.Enabled,
			Pinned: m.Pinned, SkipVersion: m.SkipVersion, IgnoreUpdates: m.IgnoreUpdates, Manifest: m.Manifest,
		}
		if m.Enabled {
			if folder, err := s.profiles.ModFolder(gameID, id, m.Key, m.UniqueID); err == nil {
				mods[i].Folder = folder
			}
		}
	}
	return mods, nil
}

// Problems checks the profile's mods. The answer is kept until the mods or versions change, unless a lookup
// failed, in which case the next call tries again.
func (s *Service) Problems(ctx context.Context, gameID, id string) (Result, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return Result{}, err
	}
	env := s.Environment(gameID)
	runID := ""
	if s.Runs != nil {
		lastID, _, err := s.Runs.LastRunSummary(gameID, id)
		if err == nil {
			runID = lastID
		}
	}
	fp := fingerprint(env, mods, runID)
	key := gameID + "/" + id
	s.mu.Lock()
	c, ok := s.cache[key]
	s.mu.Unlock()
	if ok && c.fingerprint == fp {
		return s.withDrift(gameID, id, s.withDismissed(gameID, id, c.result))
	}
	r := Check(ctx, s.meta, env, mods)
	if s.Runs != nil && runID != "" {
		_, summary, err := s.Runs.LastRunSummary(gameID, id)
		if err == nil {
			r.RunErrors = RunErrorsFromSummary(runID, summary, mods)
		}
	} else if r.RunErrors == nil {
		r.RunErrors = []RunError{}
	}
	if !r.Unknown {
		s.mu.Lock()
		s.cache[key] = cached{fp, r}
		s.mu.Unlock()
	}
	return s.withDrift(gameID, id, s.withDismissed(gameID, id, r))
}

func (s *Service) withDrift(gameID, id string, r Result) (Result, error) {
	drift, err := s.profiles.ScanModsDrift(gameID, id)
	if err != nil {
		return Result{}, err
	}
	if drift == nil {
		drift = []profile.Drift{}
	}
	r.Drift = drift
	return r, nil
}

func (s *Service) withDismissed(gameID, id string, r Result) Result {
	tokens := s.settings.Get().Dismissed[dismissBucket(gameID, id)]
	r.AssetConflicts = hideDismissed(r.AssetConflicts, tokens)
	r.Broken = hideDismissedBroken(r.Broken, tokens)
	r.Missing = hideDismissedListed(r.Missing, tokens)
	r.Settings = hideDismissedSettings(r.Settings, tokens)
	return r
}

// DismissAbandonedMod hides an abandoned-mod info row for this profile until the mod is gone.
func (s *Service) DismissAbandonedMod(_ context.Context, gameID, id, uniqueID string) error {
	uniqueID = strings.TrimSpace(uniqueID)
	if uniqueID == "" {
		return errors.New("missing mod id")
	}
	token := dismissToken("abandoned", strings.ToLower(uniqueID))
	bucket := dismissBucket(gameID, id)
	_, err := s.settings.Update(func(v *settings.Settings) {
		if slices.Contains(v.Dismissed[bucket], token) {
			return
		}
		next := maps.Clone(v.Dismissed)
		next[bucket] = append(slices.Clone(v.Dismissed[bucket]), token)
		v.Dismissed = next
	})
	return err
}

// DismissListedRequirement hides a Nexus-listed requirement for this profile until it is gone.
func (s *Service) DismissListedRequirement(_ context.Context, gameID, id, uniqueID string) error {
	uniqueID = strings.TrimSpace(uniqueID)
	if uniqueID == "" {
		return errors.New("missing requirement id")
	}
	token := dismissToken("listed", strings.ToLower(uniqueID))
	bucket := dismissBucket(gameID, id)
	_, err := s.settings.Update(func(v *settings.Settings) {
		if slices.Contains(v.Dismissed[bucket], token) {
			return
		}
		next := maps.Clone(v.Dismissed)
		next[bucket] = append(slices.Clone(v.Dismissed[bucket]), token)
		v.Dismissed = next
	})
	return err
}

// DismissSetting hides one compatibility setting for this profile until its patch group is gone.
func (s *Service) DismissSetting(_ context.Context, gameID, id, uniqueID, field string) error {
	uniqueID, field = strings.TrimSpace(uniqueID), strings.TrimSpace(field)
	if uniqueID == "" || field == "" {
		return errors.New("missing setting")
	}
	token := dismissToken("setting", strings.ToLower(uniqueID)+"\t"+strings.ToLower(field))
	bucket := dismissBucket(gameID, id)
	_, err := s.settings.Update(func(v *settings.Settings) {
		if slices.Contains(v.Dismissed[bucket], token) {
			return
		}
		next := maps.Clone(v.Dismissed)
		next[bucket] = append(slices.Clone(v.Dismissed[bucket]), token)
		v.Dismissed = next
	})
	return err
}

// RememberSettingChoice keeps a setting hint hidden while its chosen value remains current.
func (s *Service) RememberSettingChoice(_ context.Context, gameID, id, uniqueID, field, value string) error {
	uniqueID, field = strings.TrimSpace(uniqueID), strings.TrimSpace(field)
	if uniqueID == "" || field == "" {
		return errors.New("missing setting")
	}
	token := settingChoiceToken(uniqueID, field, value)
	bucket := dismissBucket(gameID, id)
	_, err := s.settings.Update(func(v *settings.Settings) {
		if slices.Contains(v.Dismissed[bucket], token) {
			return
		}
		next := maps.Clone(v.Dismissed)
		next[bucket] = append(slices.Clone(v.Dismissed[bucket]), token)
		v.Dismissed = next
	})
	return err
}

// DismissAssetConflict hides a soft (edit) Content Patcher overlap for this profile until it is gone.
func (s *Service) DismissAssetConflict(_ context.Context, gameID, id, kind, target string) error {
	if kind != "edit" || target == "" {
		return errors.New("only overlapping image or map edits can be dismissed")
	}
	token := dismissToken(kind, target)
	bucket := dismissBucket(gameID, id)
	_, err := s.settings.Update(func(v *settings.Settings) {
		if slices.Contains(v.Dismissed[bucket], token) {
			return
		}
		next := maps.Clone(v.Dismissed)
		next[bucket] = append(slices.Clone(v.Dismissed[bucket]), token)
		v.Dismissed = next
	})
	return err
}

// Updates lists the newer versions SMAPI's API suggests for the profile's mods. The answer is kept for an hour
// or until the mods or versions change, unless SMAPI's API could not be reached, in which case the next call
// tries again.
func (s *Service) Updates(ctx context.Context, gameID, id string) (UpdatesResult, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return UpdatesResult{}, err
	}
	env := s.Environment(gameID)
	fp := fingerprint(env, mods, "")
	key := gameID + "/" + id
	s.mu.Lock()
	c, ok := s.updates[key]
	s.mu.Unlock()
	if ok && c.fingerprint == fp && time.Since(c.at) < updatesTTL {
		return hideUpdates(c.result, mods, s.settings.Get()), nil
	}
	set := s.settings.Get()
	r := CheckUpdates(ctx, s.meta, env, mods, set.CheckOnlyEnabledMods)
	if !r.Unknown {
		s.mu.Lock()
		s.updates[key] = cachedUpdates{fp, time.Now(), r}
		s.mu.Unlock()
	}
	return hideUpdates(r, mods, set), nil
}

func hideUpdates(r UpdatesResult, mods []Installed, set settings.Settings) UpdatesResult {
	return HideHeld(r, mods, set.IncludePrereleaseModVersions)
}

// UpdateWarning is the Play dialog after a game update: the last launched Stardew version versus the installed one.
type UpdateWarning struct {
	Changed   bool     `json:"changed"`
	Recorded  string   `json:"recorded"`
	Installed string   `json:"installed"`
	Broken    []Broken `json:"broken"`
}

// UpdateWarning reports whether the installed game version differs from the last successful launch,
// and which of this profile's mods the compatibility data marks broken for the installed version.
func (s *Service) UpdateWarning(ctx context.Context, gameID, id string) (UpdateWarning, error) {
	env := s.Environment(gameID)
	recorded := s.settings.Get().LastPlayed[gameID].GameVersion
	if !stardew.GameVersionChanged(recorded, env.GameVersion) {
		return versionChangeWarning(recorded, env.GameVersion, nil), nil
	}
	mods, err := s.installed(gameID, id)
	if err != nil {
		return UpdateWarning{}, err
	}
	return versionChangeWarning(recorded, env.GameVersion, Check(ctx, s.meta, env, mods).Broken), nil
}

func versionChangeWarning(recorded, installed string, broken []Broken) UpdateWarning {
	out := UpdateWarning{Recorded: recorded, Installed: installed, Broken: []Broken{}}
	if !stardew.GameVersionChanged(recorded, installed) {
		return out
	}
	out.Changed = true
	if broken == nil {
		return out
	}
	out.Broken = broken
	return out
}

// Relations says what the mod key/uniqueID needs, which mods need it and where its page is.
func (s *Service) Relations(gameID, id, key, uniqueID string) (Relations, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return Relations{}, err
	}
	r, ok := Relate(mods, key, uniqueID)
	if !ok {
		return Relations{}, errors.New("no such mod in this profile")
	}
	return r, nil
}

// Pages says where each mod's page is, keyed "key/uniqueId"; mods without a known page are left out.
func (s *Service) Pages(gameID, id string) (map[string]string, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return nil, err
	}
	return Pages(mods), nil
}
