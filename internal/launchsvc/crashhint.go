package launchsvc

import (
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/dotnetstack"
	"github.com/Rethunk-Tech/mortar/internal/loader/gamelog"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// CrashHint is what Mortar makes of a profile's latest run when that run crashed.
type CrashHint struct {
	RunID string `json:"runId"`
	// ModKey and ModName name the mod the log blames; both are empty when it blames none.
	ModKey  string `json:"modKey"`
	ModName string `json:"modName"`
	// Reason is one line from the log, empty when it showed nothing usable. Code and Params let the window word it in
	// the user's language when Mortar wrote it; a line quoted from a log has no Code.
	Reason string            `json:"reason"`
	Code   string            `json:"code,omitempty"`
	Params map[string]string `json:"params,omitempty"`
	// Evidence is the log line behind the blame, such as the stack frame that runs through the mod's code.
	Evidence string `json:"evidence"`
	// NativeFrame is the native function the game died in, from its crash dump or the event log:
	// "UnityPlayer!GfxDeviceD3D11::DrawBuffers (+0x1c)", or "UnityPlayer.dll+0x1a2b3c" without symbols.
	NativeFrame string `json:"nativeFrame,omitempty"`
	// ReportDir is the folder of a crash report the game's mod library kept, empty when it kept none; the window opens
	// it through OpenCrashReport.
	ReportDir string `json:"reportDir"`
}

// CrashHint explains the profile's latest recorded run: the mod its log blames, or what the log showed when none.
// It is nil when the latest run did not crash.
func (s *Service) CrashHint(gameID, profileID string) (*CrashHint, error) {
	runs, err := s.Runs(gameID, profileID)
	if err != nil || len(runs) == 0 || runs[0].Outcome != launch.OutcomeCrashed {
		return nil, err
	}
	run := runs[0]
	hint := &CrashHint{RunID: run.ID}
	if run.Cause != nil {
		hint.ModKey, hint.ModName, hint.Reason = run.Cause.ModKey, run.Cause.ModName, firstLine(run.Cause.Detail)
		hint.Code, hint.Params = run.Cause.Reason, run.Cause.Params
		return hint, nil
	}
	// The run's own logs, never the live ones: a later launch rewrites those, and the hint names this run.
	text, player, err := s.storedRunLogs(gameID, profileID, run.ID)
	if err != nil {
		return nil, err
	}
	found, owners := s.runFindings(gameID, profileID, text, player)
	if len(found) > 0 {
		if m, f, ok := pickCulprit(found, owners); ok {
			hint.ModKey, hint.ModName, hint.Reason = m.Key, m.Name, f.Message
			hint.Code, hint.Params = f.Code, f.Params
			return hint, nil
		}
	}
	if installed, err := s.profiles.Installed(gameID, profileID); err == nil {
		if m, b, ok := blamedMod(s.stackBlames(gameID, profileID, text, player), installed); ok {
			hint.ModKey, hint.ModName, hint.Reason, hint.Evidence = m.Key, m.Name, b.Exception, b.Frame
			return hint, nil
		}
		if m, why, evidence, native, reportDir, ok := s.dumpBlame(gameID, profileID, run, installed); ok {
			hint.Evidence, hint.NativeFrame, hint.ReportDir = evidence, native, reportDir
			if m.Key != "" {
				hint.ModKey, hint.ModName, hint.Reason = m.Key, m.Name, "The game crashed inside this mod's code."
				hint.Code = "modCode"
				return hint, nil
			}
			if len(found) == 0 {
				hint.Reason, hint.Code, hint.Params = why.text, why.code, why.params
			}
		}
	}
	if len(found) > 0 {
		hint.Reason, hint.Code, hint.Params = found[0].Message, found[0].Code, found[0].Params
	}
	return hint, nil
}

// OpenCrashReport shows the profile's latest crash report folder in the system file manager. The folder is the one
// CrashHint finds, never a path the caller names.
func (s *Service) OpenCrashReport(gameID, profileID string) error {
	hint, err := s.CrashHint(gameID, profileID)
	if err != nil {
		return err
	}
	if hint == nil || hint.ReportDir == "" {
		return errors.New("the last run left no crash report")
	}
	return datadir.Open(hint.ReportDir)
}

// runFindings runs the profile loader's analyzers over a run's logs, or the game's own log analyzers for a game whose
// mods have no loader log, and maps every way such a log names a mod to the enabled installed mod that owns it: the
// same attribution the Problems tab makes. Nil when the game keeps no log of its own.
func (s *Service) runFindings(gameID, profileID, text, player string) ([]loader.Finding, map[string]profile.Installed) {
	if logs, ok := s.loaderOf(gameID, profileID); ok {
		if w, isLogs := logs.(loader.WithLogs); isLogs {
			if len(w.Analyzers()) == 0 {
				return nil, nil
			}
			var found []loader.Finding
			for _, a := range w.Analyzers() {
				found = append(found, a.Analyze(loader.Logs{Loader: text, Player: player})...)
			}
			return found, s.pluginOwners(gameID, profileID)
		}
	}
	if !game.HasPath(gameID, game.PathLog) {
		return nil, nil
	}
	return gamelog.Analyze(text), s.modIDOwners(gameID, profileID)
}

// stackBlames reads the exception stacks of the run's logs that run through a mod: the loader's log and Unity's
// player log for a loader that parses stacks, else the game's own crash report and log. Nil when the game keeps none.
func (s *Service) stackBlames(gameID, profileID, text, player string) []loader.Blame {
	l, ok := s.loaderOf(gameID, profileID)
	if !ok {
		return nil
	}
	if stacks, has := l.(loader.WithStackBlame); has {
		return stacks.StackBlame(text + "\n" + player)
	}
	// Vintage Story's mods are .NET code in the game's own process, and its logs are the crash report and main log:
	// the run kept the main log, and the crash report is the live file the game rewrites each launch.
	if game.HasPath(gameID, game.PathCrashLog) {
		return dotnetstack.Blames(text+"\n"+s.gameLogText(gameID, profileID, game.PathCrashLog), dotnetstack.VintageStory)
	}
	return nil
}

// gameLogText is the text of the game's own log roles, empty when it keeps none of them or another profile ran after
// this one: the game writes those files for whichever profile ran last.
func (s *Service) gameLogText(gameID, profileID string, roles ...string) string {
	if s.settings == nil || !s.ranLastHere(gameID, profileID) {
		return ""
	}
	return game.GameLogText(s.home, s.settings.Get(), gameID, roles...)
}

// ranLastHere is true when no other profile of the game has a run newer than this one's.
func (s *Service) ranLastHere(gameID, profileID string) bool {
	mine, err := s.LastRunID(gameID, profileID)
	if err != nil || mine == "" || s.profiles == nil {
		return false
	}
	all, err := s.profiles.List(gameID)
	if err != nil {
		return false
	}
	for _, p := range all {
		if p.ID == profileID {
			continue
		}
		if theirs, err := s.LastRunID(gameID, p.ID); err == nil && theirs > mine {
			return false
		}
	}
	return true
}

// pickCulprit is the first finding, in log order, whose plugin an installed enabled mod's own DLLs declare: the exact,
// lower-cased ownership lookup Problems attributes a loader log by, so both name the same mod for one log line.
func pickCulprit(found []loader.Finding, owners map[string]profile.Installed) (profile.Installed, loader.Finding, bool) {
	for _, f := range found {
		if m, ok := owners[strings.ToLower(f.Plugin)]; ok {
			return m, f, true
		}
	}
	return profile.Installed{}, loader.Finding{}, false
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
