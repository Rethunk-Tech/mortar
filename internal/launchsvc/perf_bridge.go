package launchsvc

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// askPerfFor bounds one question to the companion; it answers from a summary it keeps current, so this is generous.
const askPerfFor = 3 * time.Second

// InGameResult is one answer from the running game's in-game measurement. Measured is false when the launch was not
// measured, so the companion measured nothing; Report is the saved report a summary produced.
type InGameResult struct {
	Measured bool `json:"measured"`
	// Reason is why the companion stopped measuring, when it says.
	Reason string       `json:"reason,omitempty"`
	Report *SavedReport `json:"report,omitempty"`
}

// InGameLive is the measurement so far, read without saving a report: the rows and frame summary a running
// measurement shows as it goes.
type InGameLive struct {
	Measured bool             `json:"measured"`
	Reason   string           `json:"reason,omitempty"`
	Rows     []PerformanceRow `json:"rows"`
	Frame    *FrameSummary    `json:"frame,omitempty"`
}

// bridgePerf is the companion's perf reply.
type bridgePerf struct {
	Measured bool    `json:"measured"`
	Reason   string  `json:"reason"`
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
	Baseline      struct {
		ModsMs        float64 `json:"modsMs"`
		WithoutModsMs float64 `json:"withoutModsMs"`
	} `json:"baseline"`
	Plugins []struct {
		GUID          string  `json:"guid"`
		MsPerFrame    float64 `json:"msPerFrame"`
		P95Ms         float64 `json:"p95Ms"`
		PeakMs        float64 `json:"peakMs"`
		Share         float64 `json:"share"`
		CallsPerFrame float64 `json:"callsPerFrame"`
	} `json:"plugins"`
}

// MeasureInGame starts (start) or reads the running game's in-game measurement. A summary is saved with the
// profile's performance reports, its rows named by the packages whose plugins they time.
func (s *Service) MeasureInGame(gameID, profileID string, start bool) (InGameResult, error) {
	got, l, err := s.askPerf(gameID, profileID, start)
	if err != nil {
		return InGameResult{}, err
	}
	if !got.Measured || start {
		return InGameResult{Measured: got.Measured, Reason: got.Reason}, nil
	}
	rows := perfRows(got, s.perfOwners(l, gameID, profileID))
	if len(rows) == 0 {
		return InGameResult{Measured: true}, errors.New("the game has not timed any plugin yet; start measuring first")
	}
	saved, err := s.savePerformanceReport(gameID, profileID, rows, frameOf(got))
	if err != nil {
		return InGameResult{}, err
	}
	return InGameResult{Measured: true, Report: &saved}, nil
}

// PeekInGame reads the running measurement so far without saving a report, for a view that updates as it goes.
func (s *Service) PeekInGame(gameID, profileID string) (InGameLive, error) {
	got, l, err := s.askPerf(gameID, profileID, false)
	if err != nil {
		return InGameLive{}, err
	}
	if !got.Measured {
		return InGameLive{Reason: got.Reason, Rows: []PerformanceRow{}}, nil
	}
	return InGameLive{Measured: true, Rows: perfRows(got, s.perfOwners(l, gameID, profileID)), Frame: frameOf(got)}, nil
}

// askPerf asks the running game's companion for its measurement; start begins one instead.
func (s *Service) askPerf(gameID, profileID string, start bool) (bridgePerf, loader.Loader, error) {
	l, ok := s.loaderOf(gameID, profileID)
	perf, can := l.(loader.InGamePerf)
	if !ok || !can {
		return bridgePerf{}, nil, usererr.New(usererr.Invalid, "this profile's loader does not measure in game")
	}
	dir, err := s.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return bridgePerf{}, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), askPerfFor)
	defer cancel()
	raw, err := perf.Perf(ctx, loader.ProfileView{Game: gameID, Dir: dir}, start)
	if err != nil {
		return bridgePerf{}, nil, err
	}
	var got bridgePerf
	if err := json.Unmarshal(raw, &got); err != nil {
		return bridgePerf{}, nil, err
	}
	return got, l, nil
}

func frameOf(got bridgePerf) *FrameSummary {
	return &FrameSummary{
		Seconds: got.Seconds, Frames: got.Frames, FPS: got.FPS,
		AvgMs: got.FrameMs.Avg, P50Ms: got.FrameMs.P50, P95Ms: got.FrameMs.P95, P99Ms: got.FrameMs.P99, MaxMs: got.FrameMs.Max,
		MonoUsed: got.MonoUsedBytes, MonoHeap: got.MonoHeapBytes, GCCollections: got.GCCollections,
		ModsMs: got.Baseline.ModsMs, WithoutModsMs: got.Baseline.WithoutModsMs,
	}
}

// perfOwners maps what the companion names a timed unit to the package that holds it: a BepInEx plugin GUID by the
// DLLs the packages declare, a SMAPI mod's UniqueID by the installed mod.
func (s *Service) perfOwners(l loader.Loader, gameID, profileID string) func() map[string]startupOwner {
	if c, ok := l.(loader.WithCompanion); ok && c.Companion().ID == bridge.SMAPI.ID {
		return func() map[string]startupOwner {
			owners := map[string]startupOwner{}
			installed, err := s.profiles.Installed(gameID, profileID)
			if err != nil {
				return owners
			}
			for _, im := range installed {
				if im.Enabled {
					owners[strings.ToLower(im.ModID().Local())] = startupOwner{ID: im.ModID(), Name: im.Name, Version: im.Version}
				}
			}
			return owners
		}
	}
	return s.pluginPackages(gameID, profileID)
}

// perfRows turns the companion's plugin rows into one row per package, its plugins' times and calls added (a peak
// or 95th percentile is the sum of the plugins', an upper bound since they need not fall in one frame). A plugin no package
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
			rows[i].P95Ms += p.P95Ms
			rows[i].Share += p.Share
			rows[i].Calls += p.CallsPerFrame
			continue
		}
		at[name] = len(rows)
		rows = append(rows, PerformanceRow{Name: name, AverageMs: p.MsPerFrame, PeakMs: p.PeakMs, P95Ms: p.P95Ms, Share: p.Share, Calls: p.CallsPerFrame})
	}
	slices.SortStableFunc(rows, func(a, b PerformanceRow) int { return cmp.Compare(b.AverageMs, a.AverageMs) })
	return rows
}
