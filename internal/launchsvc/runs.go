package launchsvc

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

const (
	// CrashEvent is emitted with a Crash when a Mortar-started run exits with errors or a SMAPI crash/fatal.
	CrashEvent = "launch:crash"
	maxRuns    = 20
)

var runIDPattern = regexp.MustCompile(`^[0-9A-Za-z.-]+$`)

// Run is one recorded launch of a profile, without the log body.
type Run struct {
	ID         string `json:"id"`
	Started    string `json:"started"`
	Ended      string `json:"ended"`
	DurationMs int64  `json:"durationMs"`
	// Loader is the id of the loader the run used, LoaderVersion the version its log reported.
	Loader        string `json:"loader"`
	LoaderVersion string `json:"loaderVersion"`
	GameVersion   string `json:"gameVersion"`
	// Preset is the name of the launch preset the run used.
	Preset  string         `json:"preset,omitempty"`
	Outcome launch.Outcome `json:"outcome"`
	// Error is why a launch that failed to start did not start, in the words the launch reported.
	Error    string `json:"error,omitempty"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
	// Unclassified counts the log's warnings and errors no Mortar rule recognises.
	Unclassified int             `json:"unclassified,omitempty"`
	Mods         []launch.ModRef `json:"mods,omitempty"`
	Cause        *Cause          `json:"cause,omitempty"`
	Exit         *launch.Exit    `json:"exit,omitempty"`
}

type Cause struct {
	ModKey  string `json:"modKey"`
	ModName string `json:"modName"`
	ID      mod.ID `json:"id"`
	Reason  string `json:"reason"`
	Detail  string `json:"detail"`
	Path    string `json:"path"`
}

type RunHit struct {
	RunID      string         `json:"runId"`
	Started    string         `json:"started"`
	Outcome    launch.Outcome `json:"outcome"`
	LineNumber int            `json:"lineNumber"`
	Line       string         `json:"line"`
}

type RunSearch struct {
	Hits      []RunHit `json:"hits"`
	Truncated bool     `json:"truncated"`
}

// Crash names the mods that logged errors in a run that just ended.
type Crash struct {
	Game    string            `json:"game"`
	Profile string            `json:"profile"`
	RunID   string            `json:"runId"`
	Mods    []launch.ModError `json:"mods"`
	Cause   *Cause            `json:"cause,omitempty"`
	// Crashed separates a crash from a run that only logged errors.
	Crashed bool `json:"crashed"`
}

type runIndex struct {
	Runs []Run `json:"runs"`
}

func runsDir(modsDir string) string {
	return filepath.Join(filepath.Dir(modsDir), "runs")
}

func runLogPath(dir, id string) string {
	return filepath.Join(dir, id+".txt")
}

// Runs lists the newest recorded launches of the profile, at most the last 20.
func (s *Service) Runs(gameID, profileID string) ([]Run, error) {
	if _, err := game.Require(gameID); err != nil {
		return nil, err
	}
	modsDir, err := s.profiles.ModsDir(gameID, profileID)
	if err != nil {
		return nil, err
	}
	idx, err := readIndex(runsDir(modsDir))
	if err != nil {
		return nil, err
	}
	return idx.Runs, nil
}

// RunLog returns the stored SMAPI log of a recorded run, or "" when it is gone.
func (s *Service) RunLog(gameID, profileID, runID string) (string, error) {
	path, err := s.runFile(gameID, profileID, runID)
	if err != nil {
		return "", err
	}
	data, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.ToValidUTF8(string(data), ""), nil
}

func (s *Service) RunCause(gameID, profileID, runID string) (Cause, error) {
	text, err := s.RunLog(gameID, profileID, runID)
	if err != nil {
		return Cause{}, err
	}
	return s.cause(gameID, profileID, text), nil
}

func (s *Service) SearchRuns(gameID, profileID, query string) (RunSearch, error) {
	if strings.TrimSpace(query) == "" {
		return RunSearch{Hits: []RunHit{}}, nil
	}
	runs, err := s.Runs(gameID, profileID)
	if err != nil {
		return RunSearch{}, err
	}
	needle := strings.ToLower(query)
	result := RunSearch{Hits: []RunHit{}}
	for _, run := range runs {
		text, err := s.RunLog(gameID, profileID, run.ID)
		if err != nil {
			return RunSearch{}, err
		}
		for number, line := range strings.Split(text, "\n") {
			if !strings.Contains(strings.ToLower(line), needle) {
				continue
			}
			if len(result.Hits) == 500 {
				result.Truncated = true
				return result, nil
			}
			result.Hits = append(result.Hits, RunHit{
				RunID: run.ID, Started: run.Started, Outcome: run.Outcome,
				LineNumber: number + 1, Line: trimRunSearchLine(line),
			})
		}
	}
	return result, nil
}

func trimRunSearchLine(line string) string {
	line = strings.TrimSpace(line)
	runes := []rune(line)
	if len(runes) > 300 {
		return string(runes[:300])
	}
	return line
}

// RunLines returns the stored run's log as console entries.
func (s *Service) RunLines(gameID, profileID, runID string) ([]launch.Entry, error) {
	text, err := s.RunLog(gameID, profileID, runID)
	if err != nil {
		return nil, err
	}
	entries := launch.ParseLog(text)
	for i := range entries {
		entries[i].Seq = int64(i + 1)
	}
	return entries, nil
}

// RunIssues is ERROR/ALERT and WARN counts per installed mod from one recorded run.
type RunIssues struct {
	RunID string                `json:"runId"`
	Mods  []launch.ModRunIssues `json:"mods"`
}

// LastRunID returns the newest recorded run's id from the run index, without reading its log.
//
//wails:ignore
func (s *Service) LastRunID(gameID, profileID string) (string, error) {
	runs, err := s.Runs(gameID, profileID)
	if err != nil || len(runs) == 0 {
		return "", err
	}
	return runs[0].ID, nil
}

// LastRunSummary returns the newest recorded run's id and what its SMAPI log reports.
//
//wails:ignore
func (s *Service) LastRunSummary(gameID, profileID string) (string, launch.Summary, error) {
	runs, err := s.Runs(gameID, profileID)
	if err != nil {
		return "", launch.Summary{}, err
	}
	if len(runs) == 0 {
		return "", launch.Summary{}, nil
	}
	run := runs[0]
	text, err := s.RunLog(gameID, profileID, run.ID)
	if err != nil {
		return "", launch.Summary{}, err
	}
	summary := launch.Summarize(text)
	summary.ModRefs = append([]launch.ModRef{}, run.Mods...)
	if run.Exit != nil {
		launch.ApplyExit(&summary, *run.Exit)
	}
	return run.ID, summary, nil
}

// LastRunIssues attributes the newest completed run's SMAPI log to user mods.
func (s *Service) LastRunIssues(gameID, profileID string) (RunIssues, error) {
	runs, err := s.Runs(gameID, profileID)
	if err != nil {
		return RunIssues{}, err
	}
	if len(runs) == 0 {
		return RunIssues{Mods: []launch.ModRunIssues{}}, nil
	}
	run := runs[0]
	text, err := s.RunLog(gameID, profileID, run.ID)
	if err != nil {
		return RunIssues{}, err
	}
	installed, err := s.profiles.UserMods(gameID, profileID)
	if err != nil {
		return RunIssues{}, err
	}
	refs := make([]launch.ModRef, len(installed))
	for i, m := range installed {
		refs[i] = launch.ModRef{Name: m.Name, ID: m.ID}
	}
	return RunIssues{RunID: run.ID, Mods: launch.AttributeLog(text, refs)}, nil
}

func (s *Service) runFile(gameID, profileID, runID string) (string, error) {
	if _, err := game.Require(gameID); err != nil {
		return "", err
	}
	if !runIDPattern.MatchString(runID) {
		return "", fmt.Errorf("unknown run %q", runID)
	}
	modsDir, err := s.profiles.ModsDir(gameID, profileID)
	if err != nil {
		return "", err
	}
	return runLogPath(runsDir(modsDir), runID), nil
}

func readIndex(dir string) (runIndex, error) {
	var idx runIndex
	found, err := datadir.ReadJSON(filepath.Join(dir, "index.json"), &idx)
	if err != nil {
		return runIndex{}, err
	}
	if !found {
		return runIndex{}, nil
	}
	if idx.Runs == nil {
		idx.Runs = []Run{}
	}
	return idx, nil
}

func (s *Service) profileModRefs(gameID, profileID string) []launch.ModRef {
	if profileID == "" {
		return nil
	}
	installed, err := s.profiles.Installed(gameID, profileID)
	if err != nil {
		return nil
	}
	refs := make([]launch.ModRef, 0, len(installed))
	for _, im := range installed {
		if !im.Enabled {
			continue
		}
		refs = append(refs, launch.ModRef{
			Name: im.Name, ID: im.ModID(), Key: im.Key, Version: im.Version, SourceVersion: im.Source.Version,
		})
	}
	return refs
}

func (s *Service) record(g game.Game, profileID string, started time.Time, failed bool, refs ...[]launch.ModRef) {
	if s.profiles == nil || profileID == "" {
		return
	}
	modsDir, err := s.profiles.ModsDir(g.ID(), profileID)
	if err != nil {
		return
	}
	var since time.Time
	if failed {
		since = started
	}
	text := s.runText(g, profileID, modsDir, since)
	if folder, ok := launch.LoadedSave(text); ok && s.OnSavePlayed != nil {
		s.OnSavePlayed(g.ID(), profileID, folder)
	}
	stats := launch.Summarize(text)
	ldr := cmp.Or(s.profileLoader(g.ID(), profileID), loaderID(g.ID()))
	unclassified := launch.Unclassified(text)
	if ldr == bepinex5.ID {
		unclassified = bepinex5.Unclassified(text)
	}
	s.mu.Lock()
	sess := s.logs[keyOf(g)]
	s.mu.Unlock()
	if sess.haveExit {
		launch.ApplyExit(&stats, sess.exit)
	}
	// A run the player stopped from Mortar is a crash only when the game logged one before the stop: stopping a
	// frozen game is how a player gets out of a crash, but the stop itself is never one.
	stopped := sess.haveExit && sess.exit.Stopped
	if stopped {
		stats.Crashed = crashedBefore(text, s.logEnd(g, profileID), sess.stoppedAt)
	} else if !stats.Crashed && s.playerCrashed(g.ID(), profileID, started) {
		stats.Crashed = true
	}
	text = launch.CapLog(text, launch.MaxLogBytes)
	ended := time.Now()
	if started.IsZero() {
		started = ended
	}
	outcome := launch.OutcomeOf(failed, stats.Crashed)
	id := fmt.Sprintf("%s-%d", started.UTC().Format("20060102T150405"), started.UnixNano())
	dir := runsDir(modsDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	if err := datadir.WriteFile(runLogPath(dir, id), []byte(text), 0o600); err != nil {
		return
	}
	idx, err := readIndex(dir)
	if err != nil {
		idx = runIndex{}
	}
	var cause Cause
	if !stopped || stats.Crashed {
		cause = s.cause(g.ID(), profileID, text)
	}
	run := Run{
		ID: id, Started: started.UTC().Format(time.RFC3339Nano), Ended: ended.UTC().Format(time.RFC3339Nano),
		DurationMs: ended.Sub(started).Milliseconds(), Loader: ldr, LoaderVersion: stats.SMAPI, GameVersion: stats.Game,
		Preset: sess.preset, Outcome: outcome, Errors: stats.Errors, Warnings: stats.Warnings,
		Unclassified: unclassified,
	}
	if failed {
		run.Error = s.LaunchFailure(g.ID(), installOf(g))
	}
	if len(refs) > 0 {
		run.Mods = append([]launch.ModRef{}, refs[0]...)
	}
	if sess.haveExit {
		ex := stats.Exit
		run.Exit = &ex
	}
	if cause.ModName != "" {
		run.Cause = &cause
	}
	idx.Runs = append([]Run{run}, idx.Runs...)
	keep := maxRuns
	if s.settings != nil {
		if n := s.settings.Get().GamePrefs(g.ID()).RunsKept; n > 0 {
			keep = n
		}
	}
	var drop []Run
	if len(idx.Runs) > keep {
		drop = idx.Runs[keep:]
		idx.Runs = idx.Runs[:keep]
	}
	if err := datadir.WriteJSON(filepath.Join(dir, "index.json"), idx); err != nil {
		return
	}
	for _, old := range drop {
		_ = os.Remove(runLogPath(dir, old.ID))
	}
	if !failed && s.settings != nil {
		_, _ = s.settings.AddPlaytime(g.ID(), ended.Sub(started))
	}
	if title, body, crashed := RunEndNotificationText(g.Name(), stats); crashed && !failed && s.NotifyRunEnd != nil {
		s.NotifyRunEnd(RunEndNotice{Game: g.ID(), Profile: profileID, Title: title, Body: body})
	}
	if !failed && (stats.Crashed || (!stopped && stats.Errors > 0)) {
		var crashCause *Cause
		if run.Cause != nil {
			crashCause = run.Cause
		}
		s.emit(CrashEvent, Crash{Game: g.ID(), Profile: profileID, RunID: id, Mods: stats.Mods, Cause: crashCause, Crashed: stats.Crashed})
	}
}

func (s *Service) cause(gameID, profileID, text string) Cause {
	mods, err := s.profiles.UserMods(gameID, profileID)
	if err != nil {
		return Cause{}
	}
	modsDir, err := s.profiles.ModsDir(gameID, profileID)
	if err != nil {
		return Cause{}
	}
	if cause, ok := missingFileCause(mods, modsDir, strings.Split(text, "\n")); ok {
		return cause
	}
	// SMAPI prefixes each line with the name of the mod that logged it, which is the mod's manifest name.
	byName := map[string]profile.Mod{}
	for _, im := range mods {
		if _, seen := byName[im.Name]; !seen {
			byName[im.Name] = im
		}
	}
	entries := launch.ParseLog(text)
	asset := ""
	for _, e := range entries {
		if _, rest, ok := strings.Cut(e.Message, "Failed loading asset '"); ok {
			asset, _, _ = strings.Cut(rest, "'")
			break
		}
	}
	if asset != "" {
		for _, e := range entries {
			if im, ok := byName[e.Mod]; ok && strings.Contains(e.Message, asset) {
				return Cause{
					ModKey: im.Key, ModName: im.Name, ID: im.ID, Reason: "asset-load",
					Detail: fmt.Sprintf("%s: it could not load an asset. Reinstall it.", im.Name),
				}
			}
		}
	}
	for _, e := range entries {
		if e.Level != launch.Error && e.Level != launch.Alert {
			continue
		}
		msg := strings.ToLower(e.Message)
		if im, ok := byName[e.Mod]; ok && (strings.Contains(msg, "exception") || strings.Contains(msg, " failed ")) {
			return Cause{
				ModKey: im.Key, ModName: im.Name, ID: im.ID, Reason: "mod-exception",
				Detail: fmt.Sprintf("%s: it encountered an error. Reinstall it.", im.Name),
			}
		}
	}
	return Cause{}
}

func missingFileCause(mods []profile.Mod, modsDir string, lines []string) (Cause, bool) {
	for _, line := range lines {
		if !strings.Contains(line, "DirectoryNotFoundException") &&
			!strings.Contains(line, "FileNotFoundException") &&
			!strings.Contains(line, "Could not find a part of the path") {
			continue
		}
		path := extractQuotedPath(line)
		if path == "" || !pathWithin(path, modsDir) {
			continue
		}
		rel, _ := filepath.Rel(modsDir, path)
		key, _, _ := strings.Cut(rel, string(filepath.Separator))
		for _, im := range mods {
			if im.Key == key {
				return Cause{
					ModKey: im.Key, ModName: im.Name, ID: im.ID, Reason: "missing-file",
					Detail: fmt.Sprintf("%s: a file it needs could not be opened. Reinstall it.", im.Name), Path: path,
				}, true
			}
		}
	}
	return Cause{}, false
}

func extractQuotedPath(line string) string {
	start := strings.IndexAny(line, "\"'")
	if start < 0 {
		return ""
	}
	end := strings.LastIndexAny(line, "\"'")
	if end <= start {
		return ""
	}
	return strings.TrimSpace(line[start+1 : end])
}

func pathWithin(path, dir string) bool {
	absPath, err1 := filepath.Abs(path)
	absDir, err2 := filepath.Abs(dir)
	if err1 != nil || err2 != nil {
		return false
	}
	return datadir.UnderRoot(absDir, absPath)
}

// runText is the run's loader log, else the session's console. A non-zero since skips a loader log last written
// before it: a launch that failed before the game started would otherwise inherit the previous run's log.
func (s *Service) runText(g game.Game, profileID, modsDir string, since time.Time) string {
	if path, own, err := s.logPath(g.ID(), profileID); err == nil {
		if text, ok := readOwnedLog(path, own, s.home, modsDir, since); ok {
			return text
		}
	}
	s.mu.Lock()
	sess, ok := s.logs[keyOf(g)]
	s.mu.Unlock()
	if ok && sess.profile == profileID && sess.buf != nil {
		return launch.FormatLog(sess.buf.Lines())
	}
	return ""
}

// logPath is the loader log of the profile's loader. own is true when the loader writes it inside the profile's folder,
// where it can only be this profile's; a log in a shared place must say which mods folder it loaded.
func (s *Service) logPath(gameID, profileID string) (path string, own bool, err error) {
	l, ok := s.loaderOf(gameID, profileID)
	logs, isLogs := l.(loader.WithLogs)
	if !ok || !isLogs {
		return "", false, fmt.Errorf("game %q has no loader log", gameID)
	}
	if s.profiles == nil || profileID == "" {
		path, err = logs.Path(loader.ProfileView{Game: gameID})
		return path, false, err
	}
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return "", false, err
	}
	path, err = logs.Path(loader.ProfileView{Game: gameID, Dir: dir})
	if err != nil {
		return "", false, err
	}
	rel, relErr := filepath.Rel(dir, path)
	return path, relErr == nil && !strings.HasPrefix(rel, ".."), nil
}

func readOwnedLog(path string, own bool, home, modsDir string, since time.Time) (string, bool) {
	f, err := fsx.Open(path)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil || st.ModTime().Before(since) {
		return "", false
	}
	var head []byte
	if st.Size() > launch.MaxLogBytes {
		head = make([]byte, 2048)
		n, _ := io.ReadFull(f, head)
		head = head[:n]
		if _, err := f.Seek(-int64(launch.MaxLogBytes), io.SeekEnd); err != nil {
			return "", false
		}
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return "", false
	}
	text := strings.ToValidUTF8(string(data), "")
	if len(head) > 0 {
		text = strings.ToValidUTF8(string(head), "") + "\n" + launch.CapLog(text, launch.MaxLogBytes)
	}
	if !own && !launch.LogOwnedBy(text, home, modsDir) && !launch.LogOwnedBy(strings.ToValidUTF8(string(head), ""), home, modsDir) {
		return "", false
	}
	return text, true
}

// crashedBefore reports a crash line in text logged no later than at, or any crash line when at is unknown. Log lines
// carry only a time of day; logEnd, when the log was last written, dates them when the log has no start header.
func crashedBefore(text string, logEnd, at time.Time) bool {
	entries := launch.ParseLog(text)
	when := entryTimes(entries, text, logEnd)
	for i, e := range entries {
		if !e.Cont && launch.IsCrash(e) && (at.IsZero() || !when[i].After(at)) {
			return true
		}
	}
	return false
}

var logStartedRe = regexp.MustCompile(`Log started at (\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d) UTC`)

// entryTimes dates each timed entry: SMAPI's "Log started at" header dates the first line, else logEnd dates the
// last, and a clock that goes back between two lines is a midnight passed.
func entryTimes(entries []launch.Entry, text string, logEnd time.Time) []time.Time {
	out := make([]time.Time, len(entries))
	at := func(day time.Time, clock string) time.Time {
		c, err := time.Parse(time.TimeOnly, clock)
		if err != nil {
			return time.Time{}
		}
		y, m, d := day.Date()
		return time.Date(y, m, d, c.Hour(), c.Minute(), c.Second(), 0, time.Local)
	}
	if m := logStartedRe.FindStringSubmatch(text); m != nil {
		if start, err := time.Parse("2006-01-02T15:04:05", m[1]); err == nil {
			day, prev := start.Local(), ""
			for i, e := range entries {
				if e.Time == "" {
					continue
				}
				if e.Time < prev {
					day = day.AddDate(0, 0, 1)
				}
				out[i], prev = at(day, e.Time), e.Time
			}
			return out
		}
	}
	day, next := logEnd.Local(), ""
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Time == "" {
			continue
		}
		if next != "" && e.Time > next {
			day = day.AddDate(0, 0, -1)
		}
		out[i], next = at(day, e.Time), e.Time
	}
	return out
}

// logEnd is when the profile's loader log was last written, or now when it cannot be read.
func (s *Service) logEnd(g game.Game, profileID string) time.Time {
	if path, _, err := s.logPath(g.ID(), profileID); err == nil {
		if st, err := os.Stat(path); err == nil {
			return st.ModTime()
		}
	}
	return time.Now()
}
