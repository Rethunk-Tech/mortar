package nativehost

import (
	"cmp"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// updatesTimeout bounds the app's update check, which may ask each of the game's sources.
const updatesTimeout = 30 * time.Second

// appUpdate is the part of the app's updates reply the extension's rows come from (problems.Update).
type appUpdate struct {
	Name             string `json:"name"`
	Installed        string `json:"installed"`
	Version          string `json:"version"`
	NexusID          int    `json:"nexusId"`
	GitHubRepo       string `json:"githubRepo"`
	Package          string `json:"package"`
	PackageSource    string `json:"packageSource"`
	Unofficial       bool   `json:"unofficial"`
	Switch           bool   `json:"switch"`
	NotDistributable bool   `json:"notDistributable"`
}

// activeProfileUpdates is the app's own update list for the open profile of the source's game, every source included.
func activeProfileUpdates(src, key string) (state, profile string, rows []modUpdate) {
	info, ok := gameBySourceKey(src, key)
	if !ok {
		info, ok = components.GameByNexusDomain(key)
	}
	if !ok {
		return stateNoProfile, "", nil
	}
	state, profile = stateOf(info)
	if state != stateReady {
		return state, profile, nil
	}
	dataDir, err := datadir.Dir()
	if err != nil {
		return state, profile, nil
	}
	var res struct {
		Updates []appUpdate `json:"updates"`
	}
	if err := controlwire.CallDir(dataDir, "updates", map[string]any{"game": info.ID, "profile": openProfile(info)}, &res, updatesTimeout); err != nil {
		return state, profile, nil
	}
	for _, u := range res.Updates {
		if row, ok := updateRow(info, u); ok {
			rows = append(rows, row)
		}
	}
	return state, profile, rows
}

// updateRow names an update at its source, with the mod's page built by the same source drivers the app's links use.
// An unofficial build, a source switch and a mod that forbids downloads elsewhere are not offered.
func updateRow(info components.GameInfo, u appUpdate) (modUpdate, bool) {
	if u.Unofficial || u.Switch || u.NotDistributable {
		return modUpdate{}, false
	}
	row := modUpdate{Name: u.Name, Installed: u.Installed, Latest: u.Version}
	switch {
	case u.NexusID > 0:
		row.Source, row.ID, row.ModID = "nexus", strconv.Itoa(u.NexusID), u.NexusID
		row.URL = nexus.ModURL(info.NexusDomain(), u.NexusID)
	case u.Package != "":
		row.Source = cmp.Or(u.PackageSource, "thunderstore")
		row.ID = u.Package
		if sk, ok := info.Source(row.Source); ok {
			if entry, ok := source.Get(row.Source); ok {
				if linker, ok := entry.Source.(source.PageLinker); ok {
					row.URL = linker.ModPageURL(sk.Key, u.Package)
				}
			}
		}
		if _, asset, found := strings.Cut(u.Package, "-"); row.Source == "vsmoddb" && found {
			row.ID = asset
		}
	case u.GitHubRepo != "":
		row.Source, row.ID, row.URL = "github", u.GitHubRepo, "https://github.com/"+u.GitHubRepo
	default:
		return modUpdate{}, false
	}
	return row, true
}
