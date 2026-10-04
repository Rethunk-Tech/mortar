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

	"github.com/Rethunk-AI/mortar/internal/bridge"
	"github.com/Rethunk-AI/mortar/internal/datadir"
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
	ID      string `json:"id"`
	Name    string `json:"name"`
	AssetMs int64  `json:"assetMs"`
	LoadMs  int64  `json:"loadMs"`
	Ms      int64  `json:"ms"`
}

// StartupMod is one mod's exclusive startup time by where it was spent.
type StartupMod struct {
	SampleMs int64            `json:"sampleMs"`
	ID       string           `json:"id"`
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
	Smapi          string        `json:"smapi"`
	Game           string        `json:"game"`
	ProcessStart   string        `json:"processStart"`
	Phases         StartupPhases `json:"phases"`
	EntryTimed     bool          `json:"entryTimed"`
	EntryMissed    int           `json:"entryMissed"`
	Mods           []StartupMod  `json:"mods"`
	OtherMs        int64         `json:"otherMs"`
}

// prepareStartup makes SMAPI load the bridge before every other mod, so the bridge can time their Entry and
// handlers, and reports whether this launch was asked to be measured (consuming the request).
func prepareStartup(modsDir string) (bool, error) {
	if err := loadBridgeEarly(filepath.Join(modsDir, smapiGroupConfig)); err != nil {
		return false, err
	}
	marker := filepath.Join(filepath.Dir(modsDir), startupDir, measureMarker)
	err := os.Remove(marker)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
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
	if found && slices.ContainsFunc(early, func(v any) bool { s, _ := v.(string); return strings.EqualFold(s, bridge.UniqueID) }) {
		return nil
	}
	cfg[loadEarlyKey] = append([]any{bridge.UniqueID}, early...)
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
	return readStartupReports(filepath.Join(dir, startupDir))
}

func readStartupReports(dir string) ([]StartupReport, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	out := []StartupReport{}
	for _, name := range names {
		if strings.HasSuffix(name, ".samples.json") {
			continue
		}
		var r StartupReport
		if found, err := datadir.ReadJSON(name, &r); err != nil || !found {
			continue
		}
		r.ID = strings.TrimSuffix(filepath.Base(name), ".json")
		if r.Mods == nil {
			r.Mods = []StartupMod{}
		}
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
		report.Mods[index].SampleMs = samples.Mods[report.Mods[index].ID]
	}
}
