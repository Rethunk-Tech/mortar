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
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
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
	// Stoppable is whether the companion understands perf stop; a release before it was added does not say so.
	Stoppable bool `json:"stoppable"`
}

// InGameLive is the measurement so far, read without saving a report: the rows and frame summary a running
// measurement shows as it goes.
type InGameLive struct {
	Measured  bool             `json:"measured"`
	Stoppable bool             `json:"stoppable"`
	Reason    string           `json:"reason,omitempty"`
	Rows      []PerformanceRow `json:"rows"`
	Frame     *FrameSummary    `json:"frame,omitempty"`
}

// bridgePerf is the companion's perf reply.
type bridgePerf struct {
	Measured  bool    `json:"measured"`
	Stoppable bool    `json:"stoppable"`
	Reason    string  `json:"reason"`
	Seconds   float64 `json:"seconds"`
	Frames    int64   `json:"frames"`
	FPS       float64 `json:"fps"`
	FrameMs   struct {
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
		return InGameResult{Measured: got.Measured, Reason: got.Reason, Stoppable: got.Stoppable}, nil
	}
	rows := perfRows(got, s.perfOwners(l, gameID, profileID))
	if len(rows) == 0 {
		return InGameResult{Measured: true, Stoppable: got.Stoppable}, errors.New("the game has not timed any plugin yet; start measuring first")
	}
	saved, err := s.savePerformanceReport(gameID, profileID, rows, frameOf(got))
	if err != nil {
		return InGameResult{}, err
	}
	return InGameResult{Measured: true, Stoppable: got.Stoppable, Report: &saved}, nil
}

// PeekInGame reads the running measurement so far without saving a report, for a view that updates as it goes.
func (s *Service) PeekInGame(gameID, profileID string) (InGameLive, error) {
	got, l, err := s.askPerf(gameID, profileID, false)
	if err != nil {
		return InGameLive{}, err
	}
	if !got.Measured {
		return InGameLive{Stoppable: got.Stoppable, Reason: got.Reason, Rows: []PerformanceRow{}}, nil
	}
	return InGameLive{Measured: true, Stoppable: got.Stoppable, Rows: perfRows(got, s.perfOwners(l, gameID, profileID)), Frame: frameOf(got)}, nil
}

// StopInGame ends the running game's in-game measurement and reports whether one had started; the last summary stays
// readable. Only a loader whose companion can stop measuring offers it.
func (s *Service) StopInGame(gameID, profileID string) (bool, error) {
	l, ok := s.loaderOf(gameID, profileID)
	stopper, can := l.(loader.InGamePerfStop)
	if !ok || !can {
		return false, usererr.New(usererr.Invalid, "this profile's loader cannot stop measuring")
	}
	view, err := s.bridgeView(gameID, profileID)
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), askPerfFor)
	defer cancel()
	raw, err := stopper.PerfStop(ctx, view)
	if err != nil {
		return false, err
	}
	var got struct {
		Measured bool `json:"measured"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		return false, err
	}
	return got.Measured, nil
}

// askPerf asks the running game's companion for its measurement; start begins one instead.
func (s *Service) askPerf(gameID, profileID string, start bool) (bridgePerf, loader.Loader, error) {
	l, ok := s.loaderOf(gameID, profileID)
	perf, can := l.(loader.InGamePerf)
	if !ok || !can {
		return bridgePerf{}, nil, usererr.New(usererr.Invalid, "this profile's loader does not measure in game")
	}
	if g := game.Find(gameID); g.ID() != "" && !s.companionServes(g, profileID, bridge.FeaturePerf) {
		return bridgePerf{}, nil, usererr.New(usererr.Invalid, "this profile's companion cannot measure in game")
	}
	view, err := s.bridgeView(gameID, profileID)
	if err != nil {
		return bridgePerf{}, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), askPerfFor)
	defer cancel()
	raw, err := perf.Perf(ctx, view, start)
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
		return smapiOwners(func() ([]profile.Installed, error) { return s.profiles.Installed(gameID, profileID) })
	}
	return s.pluginPackages(gameID, profileID)
}

func smapiOwners(installed func() ([]profile.Installed, error)) func() map[string]startupOwner {
	return ownersBy(installed, func(im profile.Installed, owners map[string]startupOwner) {
		owners[strings.ToLower(im.ModID().Local())] = startupOwner{ID: im.ModID(), Name: im.Name, Version: im.Version}
	})
}

// perfRows turns the companion's plugin rows into one row per package, its plugins' times and calls added (a peak
// or 95th percentile is the sum of the plugins', an upper bound since they need not fall in one frame). A plugin no package
// declares keeps its GUID; a plugin that cost nothing is left out. Rows are keyed by package id, since two packages can
// share a display name.
func perfRows(got bridgePerf, owners func() map[string]startupOwner) []PerformanceRow {
	rows := []foldRow[string, PerformanceRow]{}
	for _, p := range got.Plugins {
		if p.MsPerFrame == 0 && p.CallsPerFrame == 0 {
			continue
		}
		key, name := p.GUID, p.GUID
		if o, ok := owners()[strings.ToLower(p.GUID)]; ok {
			key, name = string(o.ID), o.Name
		}
		rows = append(rows, foldRow[string, PerformanceRow]{key, PerformanceRow{Name: name, AverageMs: p.MsPerFrame, PeakMs: p.PeakMs, P95Ms: p.P95Ms, Share: p.Share, Calls: p.CallsPerFrame}})
	}
	out := foldRows(rows, func(dst, src *PerformanceRow) {
		dst.AverageMs += src.AverageMs
		dst.PeakMs += src.PeakMs
		dst.P95Ms += src.P95Ms
		dst.Share += src.Share
		dst.Calls += src.Calls
	})
	slices.SortStableFunc(out, func(a, b PerformanceRow) int { return cmp.Compare(b.AverageMs, a.AverageMs) })
	return out
}
