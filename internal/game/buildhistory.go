package game

import (
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// maxBuildsPerDepot caps how many manifests of one depot are kept.
const maxBuildsPerDepot = 50

// PastBuild is one manifest of a Steam depot Mortar has seen installed.
type PastBuild struct {
	DepotID  string `json:"depotId"`
	Manifest string `json:"manifest"`
	BuildID  string `json:"buildId,omitempty"`
	Seen     string `json:"seen"` // YYYY-MM-DD the manifest was first seen
	Current  bool   `json:"current"`
}

// buildHistory is one game's file: install folder -> depot id -> manifests, oldest first.
type buildHistory struct {
	Installs map[string]map[string][]PastBuild `json:"installs"`
}

var buildHistoryMu sync.Mutex

// LibraryOf is the Steam library folder that holds the install in dir (the parent of its steamapps folder), "" when dir
// is not in one.
func LibraryOf(dir string) string {
	for d := filepath.Clean(dir); d != filepath.Dir(d); d = filepath.Dir(d) {
		if filepath.Base(d) == "steamapps" {
			return filepath.Dir(d)
		}
	}
	return ""
}

func buildHistoryPath(dataDir, gameID string) string {
	return filepath.Join(dataDir, "build-history", gameID+".json")
}

func isSteamStore(store string) bool {
	return store == LauncherSteam || store == LauncherFlatpakSteam
}

// RecordBuildsIn notes the depot manifests the Steam install in dir has now, appending one that differs from the last
// seen for its depot. It runs on every install scan, so it writes only on a change.
func RecordBuildsIn(dataDir, gameID, appID, dir string, now time.Time) {
	library := LibraryOf(dir)
	if library == "" || appID == "" {
		return
	}
	depots, err := steam.InstalledDepots(library, appID)
	if err != nil || len(depots) == 0 {
		return
	}
	buildID := steam.BuildID(dir, appID)
	buildHistoryMu.Lock()
	defer buildHistoryMu.Unlock()
	path := buildHistoryPath(dataDir, gameID)
	var h buildHistory
	if _, err := datadir.ReadJSONOrQuarantine(path, &h); err != nil {
		slog.Warn("read build history", "game", gameID, "err", err)
		return
	}
	if h.Installs == nil {
		h.Installs = map[string]map[string][]PastBuild{}
	}
	byDepot := h.Installs[dir]
	if byDepot == nil {
		byDepot = map[string][]PastBuild{}
		h.Installs[dir] = byDepot
	}
	changed := false
	for _, d := range depots {
		if d.Manifest == "" {
			continue
		}
		seen := byDepot[d.ID]
		if n := len(seen); n > 0 && seen[n-1].Manifest == d.Manifest {
			continue
		}
		seen = append(seen, PastBuild{DepotID: d.ID, Manifest: d.Manifest, BuildID: buildID, Seen: now.Format("2006-01-02")})
		byDepot[d.ID] = seen[max(0, len(seen)-maxBuildsPerDepot):]
		changed = true
	}
	if !changed {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		slog.Warn("write build history", "game", gameID, "err", err)
		return
	}
	if err := datadir.WriteJSON(path, h); err != nil {
		slog.Warn("write build history", "game", gameID, "err", err)
	}
}

// RecordBuilds is recordBuilds in Mortar's data folder.
func RecordBuilds(gameID, appID, dir string) {
	data, err := datadir.Dir()
	if err != nil {
		return
	}
	RecordBuildsIn(data, gameID, appID, dir, time.Now())
}

// BuildHistory lists the manifests seen for the install in dir, newest first; the one each depot has now is marked
// current. An install never scanned has none.
func BuildHistory(dataDir, gameID, dir string) []PastBuild {
	buildHistoryMu.Lock()
	defer buildHistoryMu.Unlock()
	var h buildHistory
	if _, err := datadir.ReadJSON(buildHistoryPath(dataDir, gameID), &h); err != nil {
		return nil
	}
	var out []PastBuild
	for _, id := range slices.Sorted(maps.Keys(h.Installs[dir])) {
		builds := h.Installs[dir][id]
		for i, b := range slices.Backward(builds) {
			b.Current = i == len(builds)-1
			out = append(out, b)
		}
	}
	// Stable, so builds seen the same day keep their depot's newest-first order.
	slices.SortStableFunc(out, func(a, b PastBuild) int { return strings.Compare(b.Seen, a.Seen) })
	return out
}
