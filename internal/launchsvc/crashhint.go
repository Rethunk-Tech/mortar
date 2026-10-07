package launchsvc

import (
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
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

// stackBlames reads the exception stacks of the loader's log that run through a mod; nil for a loader without them.
func (s *Service) stackBlames(gameID, profileID string) []loader.Blame {
	l, ok := s.loaderOf(gameID, profileID)
	if !ok {
		return nil
	}
	logs, isLogs := l.(loader.WithLogs)
	stacks, hasStacks := l.(loader.WithStackBlame)
	if !isLogs || !hasStacks {
		return nil
	}
	raw, ok := s.loaderLog(logs, gameID, profileID)
	if !ok {
		return nil
	}
	return stacks.StackBlame(raw)
}

// pickCulprit is the first finding, in log order, whose plugin is an enabled installed mod. A log names a plugin by
// its display name, GUID or root namespace, so names match when one holds the other once case and punctuation are
// dropped.
func pickCulprit(found []loader.Finding, mods []profile.Mod) (profile.Mod, loader.Finding, bool) {
	for _, f := range found {
		plugin := fold(f.Plugin)
		if len(plugin) < 3 {
			continue
		}
		for _, m := range mods {
			if !m.Enabled {
				continue
			}
			for _, name := range []string{m.Name, string(m.ID), m.Key} {
				n := fold(name)
				if len(n) >= 3 && (n == plugin || strings.Contains(n, plugin) || strings.Contains(plugin, n)) {
					return m, f, true
				}
			}
		}
	}
	return profile.Mod{}, loader.Finding{}, false
}

func fold(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
