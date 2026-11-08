package launchsvc

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-Tech/go-smapi/logline"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
)

// PerformanceRow is one parsed row from SMAPI's performance report.
type PerformanceRow struct {
	Name      string  `json:"name"`
	AverageMs float64 `json:"averageMs"`
	PeakMs    float64 `json:"peakMs"`
	Calls     float64 `json:"calls"`
	// P95Ms is the 95th-percentile cost per frame and Share the average's fraction of the average frame, both from a
	// companion's measurement; zero in SMAPI's own console report, which has neither.
	P95Ms float64 `json:"p95Ms,omitempty"`
	Share float64 `json:"share,omitempty"`
}

type SavedReport struct {
	ID    string           `json:"id"`
	RunID string           `json:"runId,omitempty"`
	At    string           `json:"at"`
	Rows  []PerformanceRow `json:"rows"`
	// Frame is the frame and memory summary of a report the companion measured; nil for SMAPI's.
	Frame *FrameSummary `json:"frame,omitempty"`
}

// FrameSummary is the frame times and memory over a measured window: frame times in ms, memory in bytes.
type FrameSummary struct {
	Seconds       float64 `json:"seconds"`
	Frames        int64   `json:"frames"`
	FPS           float64 `json:"fps"`
	AvgMs         float64 `json:"avgMs"`
	P50Ms         float64 `json:"p50Ms"`
	P95Ms         float64 `json:"p95Ms"`
	P99Ms         float64 `json:"p99Ms"`
	MaxMs         float64 `json:"maxMs"`
	MonoUsed      int64   `json:"monoUsed"`
	MonoHeap      int64   `json:"monoHeap"`
	GCCollections int     `json:"gcCollections"`
	// ModsMs is the average frame's time in the mods' timed code and WithoutModsMs the rest of it (the game, the loader
	// and what a mod does outside its handlers): the baseline the mods are measured against.
	ModsMs        float64 `json:"modsMs"`
	WithoutModsMs float64 `json:"withoutModsMs"`
}

type performanceReportsIndex struct {
	Reports []SavedReport `json:"reports"`
}

const maxPerformanceReports = 20

// ParsePerformanceReport parses the pipe-delimited tables emitted by SMAPI's performance commands.
func ParsePerformanceReport(lines []string) []PerformanceRow {
	parsed := logline.ParsePerformanceReport(lines)
	rows := make([]PerformanceRow, len(parsed))
	for i, r := range parsed {
		rows[i] = PerformanceRow{Name: r.Name, AverageMs: r.AverageMs, PeakMs: r.PeakMs, Calls: r.Calls}
	}
	return rows
}

// PerformanceReport exposes the pure parser to the console panel.
func (s *Service) PerformanceReport(lines []string) []PerformanceRow {
	return ParsePerformanceReport(lines)
}

func (s *Service) SavePerformanceReport(gameID, profileID string, rows []PerformanceRow) (SavedReport, error) {
	return s.savePerformanceReport(gameID, profileID, rows, nil)
}

func (s *Service) savePerformanceReport(gameID, profileID string, rows []PerformanceRow, frame *FrameSummary) (SavedReport, error) {
	if len(rows) == 0 {
		return SavedReport{}, errors.New("cannot save an empty performance report")
	}
	dir, err := s.runsDirFor(gameID, profileID)
	if err != nil {
		return SavedReport{}, err
	}
	now := time.Now().UTC()
	report := SavedReport{
		ID:    fmt.Sprintf("%s-%d", now.Format("20060102T150405"), now.UnixNano()),
		RunID: s.activeRunID(gameID, profileID),
		At:    now.Format(time.RFC3339Nano),
		Rows:  append([]PerformanceRow(nil), rows...),
		Frame: frame,
	}
	index, err := readPerformanceReports(dir)
	if err != nil {
		return SavedReport{}, err
	}
	// The console panel saves whatever the log currently parses to; the same rows again would only push history out.
	if latest := index.Reports; frame == nil && len(latest) > 0 && latest[0].Frame == nil &&
		latest[0].RunID == report.RunID && slices.Equal(latest[0].Rows, rows) {
		return latest[0], nil
	}
	index.Reports = append([]SavedReport{report}, index.Reports...)
	if len(index.Reports) > maxPerformanceReports {
		index.Reports = index.Reports[:maxPerformanceReports]
	}
	if err := datadir.WriteJSON(filepath.Join(dir, "performance.json"), index); err != nil {
		return SavedReport{}, err
	}
	return report, nil
}

func (s *Service) PerformanceReports(gameID, profileID string) ([]SavedReport, error) {
	dir, err := s.runsDirFor(gameID, profileID)
	if err != nil {
		return nil, err
	}
	index, err := readPerformanceReports(dir)
	if err != nil {
		return nil, err
	}
	return index.Reports, nil
}

func readPerformanceReports(dir string) (performanceReportsIndex, error) {
	var index performanceReportsIndex
	found, err := datadir.ReadJSON(filepath.Join(dir, "performance.json"), &index)
	if err != nil {
		return performanceReportsIndex{}, err
	}
	if !found {
		return performanceReportsIndex{Reports: []SavedReport{}}, nil
	}
	if index.Reports == nil {
		index.Reports = []SavedReport{}
	}
	return index, nil
}

func (s *Service) activeRunID(gameID, profileID string) string {
	g := game.Find(gameID)
	if g == nil {
		return ""
	}
	key := keyOf(s.profileSlot(g, profileID))
	s.mu.Lock()
	defer s.mu.Unlock()
	if status, ok := s.status[key]; !ok || !status.State.Active() || status.Profile != profileID {
		return ""
	}
	session, ok := s.logs[key]
	if !ok || session.profile != profileID || session.started.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s-%d", session.started.UTC().Format("20060102T150405"), session.started.UnixNano())
}
