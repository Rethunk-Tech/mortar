package launchsvc

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

const (
	// startupDir holds the bridge's startup reports inside the profile folder.
	startupDir = "startup"
	// measureMarker in startupDir asks for a measured next launch; the launch consumes it.
	measureMarker = ".measure-next-launch"
	// smapiGroupConfig is SMAPI's per-mods-folder settings override (Constants.ApiModGroupConfigPath).
	smapiGroupConfig = "SMAPI-config.json"
	loadEarlyKey     = "ModsToLoadEarly"
)

// StartupPhases are milliseconds from the game process start.
type StartupPhases struct {
	BridgeEntry  int64 `json:"bridgeEntry"`
	EntryDone    int64 `json:"entryDone"`
	GameLaunched int64 `json:"gameLaunched"`
	TitleMenu    int64 `json:"titleMenu"`
	TitleScreen  int64 `json:"titleScreen"`
}

// StartupPack is one content pack's share of its framework's startup time.
type StartupPack struct {
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	AssetMs int64  `json:"assetMs"`
	LoadMs  int64  `json:"loadMs"`
	Ms      int64  `json:"ms"`
}

// StartupMod is one mod's exclusive startup time by where it was spent.
type StartupMod struct {
	SampleMs int64            `json:"sampleMs"`
	ID       mod.ID           `json:"id"`
	Name     string           `json:"name"`
	Version  string           `json:"version"`
	EntryMs  int64            `json:"entryMs"`
	EventMs  map[string]int64 `json:"eventMs"`
	AssetMs  int64            `json:"assetMs"`
	LoadMs   int64            `json:"loadMs"`
	Packs    []StartupPack    `json:"packs"`
}

// StartupReport is one launch's startup timings as the bridge writes them.
type StartupReport struct {
	SampledOtherMs int64         `json:"sampledOtherMs"`
	ID             string        `json:"id"`
	Loader         string        `json:"loader"`
	Game           string        `json:"game"`
	ProcessStart   string        `json:"processStart"`
	Phases         StartupPhases `json:"phases"`
	EntryTimed     bool          `json:"entryTimed"`
	EntryMissed    int           `json:"entryMissed"`
	Mods           []StartupMod  `json:"mods"`
	OtherMs        int64         `json:"otherMs"`
}

// prepareStartup reports whether this launch was asked to be measured (consuming the request) when l times
// startup; under SMAPI it also loads the bridge before every other mod, so the bridge can time their Entry and
// handlers, and under BepInEx it arms or disarms the bridge's patcher for the launch.
func prepareStartup(l loader.Loader, gameID, modsDir string) (bool, error) {
	if _, ok := l.(loader.StartupTimings); !ok {
		return false, nil
	}
	if l.ID() == smapi.ID {
		if err := loadBridgeEarly(filepath.Join(modsDir, smapiGroupConfig)); err != nil {
			return false, err
		}
	}
	dir := filepath.Join(filepath.Dir(modsDir), startupDir)
	err := os.Remove(filepath.Join(dir, measureMarker))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	measure := err == nil
	if l.ID() == bepinex5.ID {
		return measure, bepinex5.RequestStartup(dir, gameID, measure)
	}
	return measure, nil
}

// loadBridgeEarly adds the bridge to ModsToLoadEarly, keeping anything else a user put in the file. A file that is
// not valid JSON is left alone: SMAPI reads it leniently and it is the user's.
func loadBridgeEarly(path string) error {
	cfg := map[string]any{}
	found, err := datadir.ReadJSON(path, &cfg)
	if err != nil {
		if _, ok := errors.AsType[*json.SyntaxError](err); ok {
			return nil
		}
		return err
	}
	var early []any
	if list, ok := cfg[loadEarlyKey].([]any); ok {
		early = list
	}
	if found && slices.ContainsFunc(early, func(v any) bool { s, _ := v.(string); return mod.Equal(mod.SMAPI(s), mod.SMAPI(bridge.SMAPI.ID)) }) {
		return nil
	}
	cfg[loadEarlyKey] = append([]any{bridge.SMAPI.ID}, early...)
	return datadir.WriteJSON(path, cfg)
}

// MeasureNextLaunch asks the bridge to time every mod's Entry on the profile's next launch.
func (s *Service) MeasureNextLaunch(gameID, profileID string) error {
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, startupDir), 0o700); err != nil {
		return err
	}
	return datadir.WriteFile(filepath.Join(dir, startupDir, measureMarker), nil, 0o600)
}

// CancelMeasureNextLaunch takes back a pending measured launch; the next launch is timed as usual.
func (s *Service) CancelMeasureNextLaunch(gameID, profileID string) error {
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(dir, startupDir, measureMarker)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// MeasureNextLaunchPending reports whether the profile's next launch will be measured.
func (s *Service) MeasureNextLaunchPending(gameID, profileID string) (bool, error) {
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(filepath.Join(dir, startupDir, measureMarker))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// StartupReports lists the profile's startup reports, newest first. A report the bridge is still writing or that
// cannot be read is skipped.
func (s *Service) StartupReports(gameID, profileID string) ([]StartupReport, error) {
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return nil, err
	}
	scope := scopeStartupIDs
	if l, ok := s.loaderOf(gameID, profileID); ok && l.ID() == bepinex5.ID {
		scope = func(r *StartupReport) { pluginsToPackages(r, s.pluginPackages(gameID, profileID)) }
	}
	return readStartupReports(filepath.Join(dir, startupDir), scope)
}

// startupOwner is the package a BepInEx plugin's startup time is shown under.
type startupOwner struct {
	ID            mod.ID
	Name, Version string
}

// pluginPackages maps each plugin GUID (lower-cased) the profile's enabled packages declare in their DLLs to its
// package. It reads the DLLs once per call and only when a report needs it.
func (s *Service) pluginPackages(gameID, profileID string) func() map[string]startupOwner {
	return sync.OnceValue(func() map[string]startupOwner {
		owners := map[string]startupOwner{}
		installed, err := s.profiles.Installed(gameID, profileID)
		if err != nil {
			return owners
		}
		for _, im := range installed {
			if !im.Enabled || im.Folder == "" {
				continue
			}
			for _, pl := range dotnet.PluginsIn(im.Folder) {
				owners[strings.ToLower(pl.GUID)] = startupOwner{ID: im.ModID(), Name: im.Name, Version: im.Version}
			}
		}
		return owners
	})
}

// pluginsToPackages turns a BepInEx report's plugin rows, which the bridge names by GUID, into rows for the
// packages that hold them, the way the rest of Mortar names mods: a package's plugins add up to one row, and a
// plugin no package declares keeps its own row under a BepInEx id.
func pluginsToPackages(r *StartupReport, owners func() map[string]startupOwner) {
	out := make([]StartupMod, 0, len(r.Mods))
	at := map[mod.ID]int{}
	for _, m := range r.Mods {
		row := StartupMod{ID: mod.NewID(mod.FormatBepInEx, string(m.ID)), Name: m.Name, Version: m.Version, EntryMs: m.EntryMs}
		if o, ok := owners()[strings.ToLower(string(m.ID))]; ok {
			row.ID, row.Name, row.Version = o.ID, o.Name, o.Version
		}
		if i, seen := at[row.ID]; seen {
			out[i].EntryMs += row.EntryMs
			continue
		}
		at[row.ID] = len(out)
		out = append(out, row)
	}
	r.Mods = out
}

// startupReportPaths lists the bridge's reports in dir, newest first.
func startupReportPaths(dir string) ([]string, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	names = slices.DeleteFunc(names, func(name string) bool { return strings.HasSuffix(name, ".samples.json") })
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	return names, nil
}

// LatestReplaces reads, from the newest readable startup report in the profile folder, the game methods each
// Harmony owner can replace (keyed by Harmony ID, values "Type.FullName::Method"). Nil when there is no report or
// the bridge that wrote it predates the field.
func LatestReplaces(profileDir string) map[string][]string {
	names, err := startupReportPaths(filepath.Join(profileDir, startupDir))
	if err != nil {
		return nil
	}
	for _, name := range names {
		var r struct {
			Replaces map[string][]string `json:"replaces"`
		}
		if found, err := datadir.ReadJSON(name, &r); err == nil && found {
			return r.Replaces
		}
	}
	return nil
}

// readStartupReports reads the reports in dir, newest first, with scope naming their mods as Mortar does.
func readStartupReports(dir string, scope func(*StartupReport)) ([]StartupReport, error) {
	names, err := startupReportPaths(dir)
	if err != nil {
		return nil, err
	}
	out := []StartupReport{}
	for _, name := range names {
		var r StartupReport
		if found, err := datadir.ReadJSON(name, &r); err != nil || !found {
			continue
		}
		r.ID = strings.TrimSuffix(filepath.Base(name), ".json")
		if r.Mods == nil {
			r.Mods = []StartupMod{}
		}
		scope(&r)
		mergeStartupSamples(name, &r)
		out = append(out, r)
	}
	return out, nil
}

func mergeStartupSamples(reportPath string, report *StartupReport) {
	var samples startupSamples
	path := strings.TrimSuffix(reportPath, ".json") + ".samples.json"
	found, err := datadir.ReadJSON(path, &samples)
	if err != nil || !found || samples.Schema != 1 {
		return
	}
	report.SampledOtherMs = samples.OtherMs
	for index := range report.Mods {
		report.Mods[index].SampleMs = samples.Mods[report.Mods[index].ID.Local()]
	}
}

// scopeStartupIDs gives the bridge's bare SMAPI unique ids the SMAPI format, the way the rest of Mortar names mods.
func scopeStartupIDs(r *StartupReport) {
	for i := range r.Mods {
		m := &r.Mods[i]
		m.ID = mod.Parse(string(m.ID), mod.FormatSMAPI)
		for j := range m.Packs {
			m.Packs[j].ID = mod.Parse(string(m.Packs[j].ID), mod.FormatSMAPI)
		}
	}
}
