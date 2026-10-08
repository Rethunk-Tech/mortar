package launchsvc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// RunProblems returns recognised load errors in a stored run (or the live/latest log when runID is empty): SMAPI's, and
// what the profile loader's own analyzers find in its log and the same run's Unity player log.
func (s *Service) RunProblems(gameID, profileID, runID string) ([]launch.SMAPIProblem, error) {
	g := game.Find(gameID)
	if g == nil {
		return nil, fmt.Errorf("unknown game %q", gameID)
	}
	text, player, err := s.problemLogs(g, profileID, runID)
	if err != nil {
		return nil, err
	}
	found := launch.ParseSMAPIProblems(text)
	found = append(found, s.loaderProblems(gameID, profileID, text, player)...)
	refs := s.installedModRefs(gameID, profileID)
	return launch.ResolveSMAPIProblemMods(found, refs), nil
}

// problemLogs is the run's loader log and Unity player log. A live run reads the game's player log, which is its own
// while the game runs; a stored run reads the copy kept with it, never the game's, which a later launch rewrote.
func (s *Service) problemLogs(g game.Game, profileID, runID string) (string, string, error) {
	if runID != "" {
		return s.storedRunLogs(g.ID(), profileID, runID)
	}
	modsDir, err := s.profiles.ModsDir(g.ID(), profileID)
	if err != nil {
		return "", "", err
	}
	text := s.runText(g, profileID, modsDir, time.Time{})
	if st := s.statusOf(s.profileSlot(g, profileID)); text != "" && (st.State == Running || st.State == Launching) {
		player, _ := s.playerLogSince(g.ID(), profileID, time.Time{})
		return text, player, nil
	}
	runs, err := s.Runs(g.ID(), profileID)
	if err != nil {
		return "", "", err
	}
	if len(runs) == 0 {
		return text, "", nil
	}
	return s.storedRunLogs(g.ID(), profileID, runs[0].ID)
}

func (s *Service) storedRunLogs(gameID, profileID, runID string) (string, string, error) {
	text, err := s.RunLog(gameID, profileID, runID)
	if err != nil {
		return "", "", err
	}
	path, err := s.runFile(gameID, profileID, runID)
	if err != nil {
		return "", "", err
	}
	b, err := fsx.ReadFile(runPlayerLogPath(filepath.Dir(path), runID))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	return text, strings.ToValidUTF8(string(b), ""), nil
}

// loaderFixes is the one-click fix offered per loader finding kind; a kind left out offers none.
var loaderFixes = map[string]launch.SMAPIFixKind{
	bepinex5.KindMissingDependency:  launch.SMAPIFixInstallDependency,
	bepinex5.KindIncompatible:       launch.SMAPIFixUpdate,
	bepinex5.KindLoaderVersion:      launch.SMAPIFixUpdateLoader,
	bepinex5.KindLoadException:      launch.SMAPIFixDisable,
	bepinex5.KindPatchException:     launch.SMAPIFixDisable,
	bepinex5.KindPreloader:          launch.SMAPIFixDisable,
	bepinex5.KindUnityException:     launch.SMAPIFixDisable,
	bepinex5.KindIncompatiblePlugin: launch.SMAPIFixDisable,
}

// loaderProblems runs the profile loader's log analyzers over a run's logs and names the package behind each plugin.
// A plugin's own logged errors are left to the mod list's last-run column: a plugin that logs an error still loaded.
func (s *Service) loaderProblems(gameID, profileID, text, player string) []launch.SMAPIProblem {
	l, ok := s.loaderOf(gameID, profileID)
	logs, isLogs := l.(loader.WithLogs)
	if !ok || !isLogs || text == "" {
		return nil
	}
	var found []loader.Finding
	for _, a := range logs.Analyzers() {
		found = append(found, a.Analyze(loader.Logs{Loader: text, Player: player})...)
	}
	if len(found) == 0 {
		return nil
	}
	owners := s.pluginOwners(gameID, profileID)
	var out []launch.SMAPIProblem
	for _, f := range found {
		if f.Kind == bepinex5.KindPluginError {
			continue
		}
		p := launch.SMAPIProblem{Kind: launch.SMAPIProblemKind(f.Kind), ModName: f.Plugin, Detail: f.Message, Fix: loaderFixes[f.Kind], Dependency: f.Dependency}
		if owner, ok := owners[strings.ToLower(f.Plugin)]; ok {
			p.ModID, p.ModName = owner.ModID().Local(), owner.Name
		}
		out = append(out, p)
	}
	return out
}

// pluginOwners maps every name a loader log gives a plugin (see dotnet.Owners) to the profile mod whose DLLs declare it.
func (s *Service) pluginOwners(gameID, profileID string) map[string]profile.Installed {
	if s.profiles == nil || profileID == "" {
		return nil
	}
	// A loader with no analyzers (SMAPI) logs mods by manifest name, never by plugin, so there is nothing to map and the
	// DLL walk is skipped.
	l, ok := s.loaderOf(gameID, profileID)
	if logs, isLogs := l.(loader.WithLogs); !ok || !isLogs || len(logs.Analyzers()) == 0 {
		return nil
	}
	installed, err := s.profiles.Installed(gameID, profileID)
	if err != nil {
		return nil
	}
	var with []profile.Installed
	var dirs []string
	for _, im := range installed {
		if im.Folder != "" {
			with, dirs = append(with, im), append(dirs, im.Folder)
		}
	}
	owners := map[string]profile.Installed{}
	for name, i := range dotnet.Owners(dirs) {
		owners[name] = with[i]
	}
	return owners
}

func (s *Service) installedModRefs(gameID, profileID string) []launch.ModRef {
	if s.profiles == nil || profileID == "" {
		return nil
	}
	installed, err := s.profiles.Installed(gameID, profileID)
	if err != nil {
		return nil
	}
	refs := make([]launch.ModRef, 0, len(installed))
	for _, im := range installed {
		refs = append(refs, launch.ModRef{
			Name: im.Name, ID: im.ModID(), Key: im.Key, Version: im.Version, SourceVersion: im.Source.Version,
		})
	}
	return refs
}
