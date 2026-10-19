package launchsvc

import (
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/dotnetstack"
	"github.com/Rethunk-Tech/mortar/internal/loader/gamelog"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// CrashHint is what Mortar makes of a profile's latest run when that run crashed.
type CrashHint struct {
	RunID string `json:"runId"`
	// ModKey and ModName name the mod the log blames; both are empty when it blames none.
	ModKey  string `json:"modKey"`
	ModName string `json:"modName"`
	// Reason is one line from the log, empty when it showed nothing usable.
	Reason string `json:"reason"`
	// Evidence is the log line behind the blame, such as the stack frame that runs through the mod's code.
	Evidence string `json:"evidence"`
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
		return hint, nil
	}
	found := s.loaderFindings(gameID, profileID)
	if len(found) > 0 {
		mods, err := s.profiles.UserMods(gameID, profileID)
		if err != nil {
			return nil, err
		}
		if m, f, ok := pickCulprit(found, mods); ok {
			hint.ModKey, hint.ModName, hint.Reason = m.Key, m.Name, f.Message
			return hint, nil
		}
	}
	if installed, err := s.profiles.Installed(gameID, profileID); err == nil {
		if m, b, ok := blamedMod(s.stackBlames(gameID, profileID), installed); ok {
			hint.ModKey, hint.ModName, hint.Reason, hint.Evidence = m.Key, m.Name, b.Exception, b.Frame
			return hint, nil
		}
		if m, reason, evidence, ok := s.dumpBlame(gameID, run, installed); ok {
			hint.Evidence = evidence
			if m.Key != "" {
				hint.ModKey, hint.ModName, hint.Reason = m.Key, m.Name, "The game crashed inside this mod's code."
				return hint, nil
			}
			if len(found) == 0 {
				hint.Reason = reason
			}
		}
	}
	if len(found) > 0 {
		hint.Reason = found[0].Message
	}
	return hint, nil
}

// loaderFindings runs the profile loader's log analyzers over its logs; nil when it has none.
func (s *Service) loaderFindings(gameID, profileID string) []loader.Finding {
	l, ok := s.loaderOf(gameID, profileID)
	if !ok {
		return nil
	}
	logs, ok := l.(loader.WithLogs)
	if !ok && game.HasPath(gameID, game.PathLog) {
		return gamelog.Analyze(s.gamePathText(gameID, game.PathLog) + "\n" + s.gamePathText(gameID, game.PathCrashLog))
	}
	if !ok || len(logs.Analyzers()) == 0 {
		return nil
	}
	raw, ok := s.loaderLog(logs, gameID, profileID)
	if !ok {
		return nil
	}
	in := loader.Logs{Loader: raw}
	if w, ok := l.(loader.WithPlayerLog); ok && s.settings != nil {
		if p, err := game.PathFor(s.home, s.settings.Get(), gameID, "", w.PlayerLogRole()); err == nil {
			b, _ := fsx.ReadFile(p)
			in.Player = string(b)
		}
	}
	var out []loader.Finding
	for _, a := range logs.Analyzers() {
		out = append(out, a.Analyze(in)...)
	}
	return out
}

// loaderLog reads the profile loader's log.
func (s *Service) loaderLog(logs loader.WithLogs, gameID, profileID string) (string, bool) {
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return "", false
	}
	path, err := logs.Path(loader.ProfileView{Game: gameID, Dir: dir})
	if err != nil {
		return "", false
	}
	raw, err := fsx.ReadFile(path)
	return string(raw), err == nil
}

// stackBlames reads the exception stacks of the run's logs that run through a mod: the loader's log and Unity's
// player log for a loader that parses stacks, else the game's own crash report and log. Nil when the game keeps none.
func (s *Service) stackBlames(gameID, profileID string) []loader.Blame {
	l, ok := s.loaderOf(gameID, profileID)
	if !ok {
		return nil
	}
	if stacks, has := l.(loader.WithStackBlame); has {
		var text string
		if logs, isLogs := l.(loader.WithLogs); isLogs {
			text, _ = s.loaderLog(logs, gameID, profileID)
		}
		if w, has := l.(loader.WithPlayerLog); has {
			text += "\n" + s.gamePathText(gameID, w.PlayerLogRole())
		}
		return stacks.StackBlame(text)
	}
	// Vintage Story's mods are .NET code in the game's own process, and its logs are the crash report and main log.
	if game.HasPath(gameID, game.PathCrashLog) {
		return dotnetstack.Blames(s.gamePathText(gameID, game.PathLog)+"\n"+s.gamePathText(gameID, game.PathCrashLog), dotnetstack.VintageStory)
	}
	return nil
}

// gamePathText is the text of the file the catalog places under role, empty when the game keeps none.
func (s *Service) gamePathText(gameID, role string) string {
	if s.settings == nil || !game.HasPath(gameID, role) {
		return ""
	}
	path, err := game.PathFor(s.home, s.settings.Get(), gameID, "", role)
	if err != nil {
		return ""
	}
	b, _ := fsx.ReadFile(path)
	return strings.ToValidUTF8(string(b), "")
}

// pickCulprit is the first finding, in log order, whose plugin is an enabled installed mod. A log names a plugin by
// its display name, GUID or root namespace: names match when equal once case and punctuation are dropped, or when
// one is a whole dot or space separated run of the other's segments (a GUID's namespace prefix is noise). A shorter
// name inside a longer one matches only when it covers at least half of it, so plugin "Lib" never blames "LethalLib".
func pickCulprit(found []loader.Finding, mods []profile.Mod) (profile.Mod, loader.Finding, bool) {
	for _, f := range found {
		plugin := mod.LooseName(f.Plugin)
		if len(plugin) < 3 {
			continue
		}
		for _, m := range mods {
			if !m.Enabled {
				continue
			}
			for _, name := range []string{m.Name, string(m.ID), m.Key} {
				if namesMatch(f.Plugin, name) {
					return m, f, true
				}
			}
		}
	}
	return profile.Mod{}, loader.Finding{}, false
}

func namesMatch(plugin, name string) bool {
	p, n := mod.LooseName(plugin), mod.LooseName(name)
	if len(n) < 3 || len(p) < 3 {
		return false
	}
	if n == p {
		return true
	}
	// The mod's name inside the plugin's GUID: any whole segment run counts.
	if hasSegmentRun(plugin, n, false) {
		return true
	}
	return hasSegmentRun(name, p, true)
}

// hasSegmentRun reports whether some run of whole segments of s, split at non-alphanumerics, loosely equals want.
// With half set, want must also cover at least half of s.
func hasSegmentRun(s, want string, half bool) bool {
	segs := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	if half && len(want)*2 < len(mod.LooseName(s)) {
		return false
	}
	for i := range segs {
		joined := ""
		for j := i; j < len(segs) && len(joined) < len(want); j++ {
			joined += mod.LooseName(segs[j])
		}
		if joined == want {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
