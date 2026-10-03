package launchsvc

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/game/stardew"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

const (
	// SweepEvent is emitted when a version change triggers a patch-day sweep.
	SweepEvent = "launch:sweep"

	fixUpdate = "update"
	fixOff    = "off"
	fixNone   = "none"
)

// SweepMod is one profile mod that needs attention after a game or SMAPI change.
type SweepMod struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	UniqueID string `json:"uniqueId"`
	Status   string `json:"status"`
	Fix      string `json:"fix"`
}

// ProfileSweep is one profile's patch-day findings.
type ProfileSweep struct {
	Profile     string     `json:"profile"`
	Name        string     `json:"name"`
	Broken      []SweepMod `json:"broken"`
	MissingDeps int        `json:"missingDeps"`
}

// SweepReport is the patch-day summary for one game.
type SweepReport struct {
	Game         string         `json:"game"`
	GameName     string         `json:"gameName"`
	GameVersion  string         `json:"gameVersion"`
	SMAPIVersion string         `json:"smapiVersion"`
	Triggered    bool           `json:"triggered"`
	Profiles     []ProfileSweep `json:"profiles"`
}

// Sweep compares the installed game and SMAPI versions to the last sweep. When they differ it
// records the new versions and lists broken or obsolete mods, update-fixable ones, and missing deps.
func (s *Service) Sweep(ctx context.Context, gameID string) (SweepReport, error) {
	g := game.Find(gameID)
	if g == nil {
		return SweepReport{}, fmt.Errorf("unknown game %q", gameID)
	}
	gameVer, smapiVer, err := s.installedSweepVersions(gameID, g)
	if err != nil {
		return SweepReport{}, err
	}
	rep := SweepReport{Game: gameID, GameName: g.Name(), GameVersion: gameVer, SMAPIVersion: smapiVer}
	if !s.sweepVersionsChanged(gameID, gameVer, smapiVer) {
		return rep, nil
	}
	rep.Triggered = true
	idx, err := s.sweepCompat(ctx)
	if err != nil {
		return SweepReport{}, err
	}
	profiles, err := s.profiles.List(gameID)
	if err != nil {
		return SweepReport{}, err
	}
	updates := s.sweepUpdateSet(ctx, gameID, gameVer, smapiVer, profiles)
	rep.Profiles = s.scanSweepProfiles(gameID, profiles, idx, updates)
	if err := s.settings.RecordLastSweep(gameID, gameVer, smapiVer); err != nil {
		return SweepReport{}, err
	}
	return rep, nil
}

// MaybeSweep runs Sweep and emits SweepEvent when versions changed and any profile needs attention.
func (s *Service) MaybeSweep(ctx context.Context, gameID string) {
	rep, err := s.Sweep(ctx, gameID)
	if err != nil || !rep.Triggered || !rep.NeedsAttention() {
		return
	}
	s.emit(SweepEvent, rep)
}

// SweepOnStart runs MaybeSweep in the background after the window can receive events.
func SweepOnStart(s *Service, gameID string) {
	go s.MaybeSweep(context.Background(), gameID)
}

func (r SweepReport) NeedsAttention() bool {
	for _, p := range r.Profiles {
		if len(p.Broken) > 0 || p.MissingDeps > 0 {
			return true
		}
	}
	return false
}

func (s *Service) installedSweepVersions(gameID string, g game.Game) (string, string, error) {
	if s.SweepVersions != nil {
		return s.SweepVersions(gameID)
	}
	set := s.settings.Get()
	dir, err := game.InstallDir(s.home, set, gameID)
	if err != nil {
		return "", "", err
	}
	st := g.LoaderStatus(dir, set.Loaders[gameID])
	return st.GameVersion, st.Version, nil
}

func (s *Service) sweepVersionsChanged(gameID, gameVer, smapiVer string) bool {
	last := s.settings.Get().GamePrefs(gameID)
	return stardew.GameVersionChanged(last.LastSweepGameVersion, gameVer) || last.LastSweepSMAPIVersion != smapiVer
}

func (s *Service) sweepCompat(ctx context.Context) (meta.CompatIndex, error) {
	if s.SweepCompat != nil {
		return s.SweepCompat(ctx)
	}
	c := meta.Client{}
	return c.CompatList(ctx)
}

func (s *Service) scanSweepProfiles(gameID string, profiles []profile.Profile, idx meta.CompatIndex, updates map[string]bool) []ProfileSweep {
	out := make([]ProfileSweep, 0, len(profiles))
	for _, p := range profiles {
		if p.Error != "" {
			continue
		}
		mods, err := s.profiles.Mods(gameID, p.ID)
		if err != nil {
			continue
		}
		has := func(uniqueID string, _ int) bool {
			if s.SweepHasUpdate != nil {
				return s.SweepHasUpdate(uniqueID, 0)
			}
			return updates[strings.ToLower(uniqueID)]
		}
		row := ProfileSweep{Profile: p.ID, Name: p.Name, Broken: sweepBroken(mods, idx, has)}
		row.MissingDeps = s.sweepMissingDeps(gameID, p.ID, mods)
		out = append(out, row)
	}
	return out
}

func (s *Service) sweepMissingDeps(gameID, profileID string, mods []profile.Mod) int {
	if s.SweepMissingDeps != nil {
		return s.SweepMissingDeps(gameID, profileID)
	}
	have := map[string]bool{}
	for _, m := range mods {
		if m.Enabled && m.UniqueID != "" {
			have[strings.ToLower(m.UniqueID)] = true
		}
	}
	n := 0
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		for _, need := range m.Needs {
			if !have[strings.ToLower(need)] {
				n++
			}
		}
	}
	return n
}

func (s *Service) sweepUpdateSet(ctx context.Context, gameID, gameVer, smapiVer string, profiles []profile.Profile) map[string]bool {
	if s.SweepHasUpdate != nil {
		return nil
	}
	var installed []meta.InstalledMod
	for _, p := range profiles {
		if p.Error != "" {
			continue
		}
		list, err := s.profiles.Mods(gameID, p.ID)
		if err != nil {
			continue
		}
		for _, m := range list {
			if m.UniqueID != "" {
				installed = append(installed, meta.InstalledMod{ID: m.UniqueID, Version: m.Version})
			}
		}
	}
	out := map[string]bool{}
	c := meta.Client{}
	for _, r := range c.CheckUpdates(ctx, meta.UpdateRequest{
		APIVersion: smapiVer, GameVersion: gameVer, Platform: sweepPlatform(), Mods: installed,
	}) {
		if r.Suggested != nil {
			out[strings.ToLower(r.ID)] = true
		}
	}
	return out
}

func sweepBroken(mods []profile.Mod, idx meta.CompatIndex, hasUpdate func(string, int) bool) []SweepMod {
	var out []SweepMod
	for _, m := range mods {
		e, ok := idx.Lookup(m.UniqueID, 0)
		if !ok || (e.Status != meta.StatusBroken && e.Status != meta.StatusObsolete) {
			continue
		}
		out = append(out, SweepMod{
			Key: m.Key, Name: sweepModName(m), UniqueID: m.UniqueID, Status: e.Status, Fix: sweepFix(e, m, hasUpdate),
		})
	}
	return out
}

func sweepFix(e meta.CompatEntry, m profile.Mod, hasUpdate func(string, int) bool) string {
	if e.Status == meta.StatusObsolete {
		return fixOff
	}
	if useLatest(e) && hasUpdate != nil && hasUpdate(m.UniqueID, 0) {
		return fixUpdate
	}
	return fixNone
}

func useLatest(e meta.CompatEntry) bool {
	s := strings.ToLower(e.Summary)
	return strings.Contains(s, "use latest") || strings.Contains(s, "use the latest")
}

func sweepModName(m profile.Mod) string {
	if m.Name != "" {
		return m.Name
	}
	if m.UniqueID != "" {
		return m.UniqueID
	}
	return m.Key
}

func sweepPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "Mac"
	default:
		return "Linux"
	}
}
