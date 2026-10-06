package launchsvc

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// askPerfFor bounds one question to the companion; it answers from a summary it keeps current, so this is generous.
const askPerfFor = 3 * time.Second

// InGameResult is one answer from the running game's in-game measurement. Measured is false when the launch was not
// measured, so the companion measured nothing; Report is the saved report a summary produced.
type InGameResult struct {
	Measured bool         `json:"measured"`
	Report   *SavedReport `json:"report,omitempty"`
}

// bridgePerf is the companion's perf reply.
type bridgePerf struct {
	Measured bool    `json:"measured"`
	Seconds  float64 `json:"seconds"`
	Frames   int64   `json:"frames"`
	FPS      float64 `json:"fps"`
	FrameMs  struct {
		Avg float64 `json:"avg"`
		P50 float64 `json:"p50"`
		P95 float64 `json:"p95"`
		P99 float64 `json:"p99"`
		Max float64 `json:"max"`
	} `json:"frameMs"`
	MonoUsedBytes int64 `json:"monoUsedBytes"`
	MonoHeapBytes int64 `json:"monoHeapBytes"`
	GCCollections int   `json:"gcCollections"`
	Plugins       []struct {
		GUID          string  `json:"guid"`
		MsPerFrame    float64 `json:"msPerFrame"`
		PeakMs        float64 `json:"peakMs"`
		CallsPerFrame float64 `json:"callsPerFrame"`
	} `json:"plugins"`
}

// MeasureInGame starts (start) or reads the running game's in-game measurement. A summary is saved with the
// profile's performance reports, its rows named by the packages whose plugins they time.
func (s *Service) MeasureInGame(gameID, profileID string, start bool) (InGameResult, error) {
	l, ok := s.loaderOf(gameID, profileID)
	perf, can := l.(loader.InGamePerf)
	if !ok || !can {
		return InGameResult{}, usererr.New(usererr.Invalid, "this profile's loader does not measure in game")
	}
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return InGameResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), askPerfFor)
	defer cancel()
	raw, err := perf.Perf(ctx, loader.ProfileView{Game: gameID, Dir: dir}, start)
	if err != nil {
		return InGameResult{}, err
	}
	var got bridgePerf
	if err := json.Unmarshal(raw, &got); err != nil {
		return InGameResult{}, err
	}
	if !got.Measured || start {
		return InGameResult{Measured: got.Measured}, nil
	}
	rows := perfRows(got, s.pluginPackages(gameID, profileID))
	if len(rows) == 0 {
		return InGameResult{Measured: true}, errors.New("the game has not timed any plugin yet; start measuring first")
	}
	saved, err := s.savePerformanceReport(gameID, profileID, rows, &FrameSummary{
		Seconds: got.Seconds, Frames: got.Frames, FPS: got.FPS,
		AvgMs: got.FrameMs.Avg, P50Ms: got.FrameMs.P50, P95Ms: got.FrameMs.P95, P99Ms: got.FrameMs.P99, MaxMs: got.FrameMs.Max,
		MonoUsed: got.MonoUsedBytes, MonoHeap: got.MonoHeapBytes, GCCollections: got.GCCollections,
	})
	if err != nil {
		return InGameResult{}, err
	}
	return InGameResult{Measured: true, Report: &saved}, nil
}

// perfRows turns the companion's plugin rows into one row per package, its plugins' times and calls added (a peak
// is the sum of the plugins' peaks, an upper bound since they need not fall in one frame). A plugin no package
// declares keeps its GUID; a plugin that cost nothing is left out.
func perfRows(got bridgePerf, owners func() map[string]startupOwner) []PerformanceRow {
	rows := []PerformanceRow{}
	at := map[string]int{}
	for _, p := range got.Plugins {
		if p.MsPerFrame == 0 && p.CallsPerFrame == 0 {
			continue
		}
		name := p.GUID
		if o, ok := owners()[strings.ToLower(p.GUID)]; ok {
			name = o.Name
		}
		if i, seen := at[name]; seen {
			rows[i].AverageMs += p.MsPerFrame
			rows[i].PeakMs += p.PeakMs
			rows[i].Calls += p.CallsPerFrame
			continue
		}
		at[name] = len(rows)
		rows = append(rows, PerformanceRow{Name: name, AverageMs: p.MsPerFrame, PeakMs: p.PeakMs, Calls: p.CallsPerFrame})
	}
	slices.SortStableFunc(rows, func(a, b PerformanceRow) int { return cmp.Compare(b.AverageMs, a.AverageMs) })
	return rows
}
