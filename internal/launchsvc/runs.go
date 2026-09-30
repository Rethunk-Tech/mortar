package launchsvc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launch"
)

const (
	// CrashEvent is emitted with a Crash when a Mortar-started run exits with errors or a SMAPI crash/fatal.
	CrashEvent = "launch:crash"
	maxRuns    = 20
)

var runIDPattern = regexp.MustCompile(`^[0-9A-Za-z.-]+$`)

// Run is one recorded launch of a profile, without the log body.
type Run struct {
	ID           string         `json:"id"`
	Started      string         `json:"started"`
	Ended        string         `json:"ended"`
	DurationMs   int64          `json:"durationMs"`
	SMAPIVersion string         `json:"smapiVersion"`
	GameVersion  string         `json:"gameVersion"`
	Outcome      launch.Outcome `json:"outcome"`
	Errors       int            `json:"errors"`
	Warnings     int            `json:"warnings"`
}

// Crash names the mods that logged errors in a run that just ended.
type Crash struct {
	Game    string            `json:"game"`
	Profile string            `json:"profile"`
	RunID   string            `json:"runId"`
	Mods    []launch.ModError `json:"mods"`
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
	if game.Find(gameID) == nil {
		return nil, fmt.Errorf("unknown game %q", gameID)
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

func (s *Service) runFile(gameID, profileID, runID string) (string, error) {
	if game.Find(gameID) == nil {
		return "", fmt.Errorf("unknown game %q", gameID)
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
	data, err := fsx.ReadFile(filepath.Join(dir, "index.json"))
	if errors.Is(err, os.ErrNotExist) {
		return runIndex{}, nil
	}
	if err != nil {
		return runIndex{}, err
	}
	var idx runIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return runIndex{}, err
	}
	if idx.Runs == nil {
		idx.Runs = []Run{}
	}
	return idx, nil
}

func (s *Service) record(g game.Game, profileID string, started time.Time, failed bool) {
	if s.profiles == nil || profileID == "" {
		return
	}
	modsDir, err := s.profiles.ModsDir(g.ID(), profileID)
	if err != nil {
		return
	}
	text := s.runText(g, profileID, modsDir)
	stats := launch.Summarize(text)
	text = launch.CapLog(text, launch.MaxLogBytes)
	ended := time.Now()
	if started.IsZero() {
		started = ended
	}
	outcome := launch.OutcomeRan
	if failed {
		outcome = launch.OutcomeFailed
	} else if stats.Crashed {
		outcome = launch.OutcomeCrashed
	}
	id := fmt.Sprintf("%s-%d", ended.UTC().Format("20060102T150405"), ended.UnixNano())
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
	run := Run{
		ID: id, Started: started.UTC().Format(time.RFC3339Nano), Ended: ended.UTC().Format(time.RFC3339Nano),
		DurationMs: ended.Sub(started).Milliseconds(), SMAPIVersion: stats.SMAPI, GameVersion: stats.Game,
		Outcome: outcome, Errors: stats.Errors, Warnings: stats.Warnings,
	}
	idx.Runs = append([]Run{run}, idx.Runs...)
	var drop []Run
	if len(idx.Runs) > maxRuns {
		drop = idx.Runs[maxRuns:]
		idx.Runs = idx.Runs[:maxRuns]
	}
	if err := datadir.WriteJSON(filepath.Join(dir, "index.json"), idx); err != nil {
		return
	}
	for _, old := range drop {
		_ = os.Remove(runLogPath(dir, old.ID))
	}
	if !failed && (stats.Errors > 0 || stats.Crashed) {
		s.emit(CrashEvent, Crash{Game: g.ID(), Profile: profileID, RunID: id, Mods: stats.Mods})
	}
}

func (s *Service) runText(g game.Game, profileID, modsDir string) string {
	if path, err := g.LogFile(); err == nil {
		if text, ok := readOwnedLog(path, s.home, modsDir); ok {
			return text
		}
	}
	s.mu.Lock()
	sess, ok := s.logs[g.ID()]
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
