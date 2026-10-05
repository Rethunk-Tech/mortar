package launchsvc

import (
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
	ID           string `json:"id"`
	Started      string `json:"started"`
	Ended        string `json:"ended"`
	DurationMs   int64  `json:"durationMs"`
	SMAPIVersion string `json:"smapiVersion"`
	GameVersion  string `json:"gameVersion"`
	// Preset is the name of the launch preset the run used.
	Preset   string          `json:"preset,omitempty"`
	Outcome  launch.Outcome  `json:"outcome"`
	Errors   int             `json:"errors"`
	Warnings int             `json:"warnings"`
	Mods     []launch.ModRef `json:"mods,omitempty"`
	Cause    *Cause          `json:"cause,omitempty"`
	Exit     *launch.Exit    `json:"exit,omitempty"`
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
	text := s.runText(g, profileID, modsDir)
	if folder, ok := launch.LoadedSave(text); ok && s.OnSavePlayed != nil {
		s.OnSavePlayed(g.ID(), profileID, folder)
	}
	stats := launch.Summarize(text)
	s.mu.Lock()
	sess := s.logs[keyOf(g)]
	s.mu.Unlock()
	if sess.haveExit {
		launch.ApplyExit(&stats, sess.exit)
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
	cause := s.cause(g.ID(), profileID, text)
	run := Run{
		ID: id, Started: started.UTC().Format(time.RFC3339Nano), Ended: ended.UTC().Format(time.RFC3339Nano),
		DurationMs: ended.Sub(started).Milliseconds(), SMAPIVersion: stats.SMAPI, GameVersion: stats.Game,
		Preset: sess.preset, Outcome: outcome, Errors: stats.Errors, Warnings: stats.Warnings,
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
	if !failed && (stats.Errors > 0 || stats.Crashed) {
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
	lines := strings.Split(text, "\n")
	if cause, ok := missingFileCause(mods, modsDir, lines); ok {
		return cause
	}
	asset := ""
	for _, line := range lines {
		if rest, _, ok := strings.Cut(line, "Failed loading asset '"); ok {
			asset = strings.TrimSuffix(rest, "'")
			break
		}
	}
	if asset != "" {
		for _, line := range lines {
			for _, im := range mods {
				if strings.Contains(line, "["+im.Name+"]") && strings.Contains(line, asset) {
					return Cause{
						ModKey: im.Key, ModName: im.Name, ID: im.ID, Reason: "asset-load",
						Detail: fmt.Sprintf("%s: it could not load an asset. Reinstall it.", im.Name),
					}
				}
			}
		}
	}
	for _, line := range lines {
		for _, im := range mods {
			if strings.Contains(line, "["+im.Name+"]") && (strings.Contains(strings.ToLower(line), "exception") ||
				strings.Contains(strings.ToLower(line), " failed ")) {
				return Cause{
					ModKey: im.Key, ModName: im.Name, ID: im.ID, Reason: "mod-exception",
					Detail: fmt.Sprintf("%s: it encountered an error. Reinstall it.", im.Name),
				}
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

func (s *Service) runText(g game.Game, profileID, modsDir string) string {
	if path, err := game.LogFile(g.ID()); err == nil {
		if text, ok := readOwnedLog(path, s.home, modsDir); ok {
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

func readOwnedLog(path, home, modsDir string) (string, bool) {
	f, err := fsx.Open(path)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
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
	if !launch.LogOwnedBy(text, home, modsDir) && !launch.LogOwnedBy(strings.ToValidUTF8(string(head), ""), home, modsDir) {
		return "", false
	}
	return text, true
}
