package nativehost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// installTimeout bounds the app's answer to a queued install; the queue only takes the request, the downloads run on.
const installTimeout = 2 * time.Minute

// answerPackages replies to installedPackages and installPackage for a Thunderstore community.
func (h handlers) answerPackages(req request) reply {
	if req.Source != "thunderstore" || h.packages == nil {
		return reply{Error: "Mortar does not read " + req.Source + " pages"}
	}
	st, name, pkgs := h.packages(req.SourceGameKey)
	rep := reply{State: st, Connected: st == stateReady, Profile: name, Games: hostGames()}
	if st == stateOff {
		return rep
	}
	if req.Type == "installedPackages" {
		if pkgs == nil {
			pkgs = []string{}
		}
		rep.Packages = &pkgs
		return rep
	}
	switch {
	case st != stateReady:
		rep.Error = "open a profile for this game in Mortar first"
	case h.installPackage == nil:
		rep.Error = "Mortar cannot install packages here"
	default:
		if err := h.installPackage(req.SourceGameKey, req.Package); err != nil {
			rep.Error = err.Error()
		} else {
			rep.OK = true
		}
	}
	return rep
}

func gameBySourceKey(source, key string) (components.GameInfo, bool) {
	m, err := components.BundledManifest()
	if err != nil {
		return components.GameInfo{}, false
	}
	for _, g := range m.Games {
		for _, src := range g.Sources {
			if src.ID == source && src.Key == key && key != "" {
				return g, true
			}
		}
	}
	return components.GameInfo{}, false
}

// openProfile is the game's last open profile id, or "" when there is none.
func openProfile(info components.GameInfo) string {
	store, err := settings.Open()
	if err != nil {
		return ""
	}
	id := store.Get().LastProfile[info.ID]
	if filepath.Base(id) != id {
		return ""
	}
	return id
}

// activePackages lists the Thunderstore packages the community's game holds in its open profile.
func activePackages(key string) (state, profile string, packages []string) {
	info, ok := gameBySourceKey("thunderstore", key)
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
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return state, profile, nil
	}
	defer func() { _ = root.Close() }()
	data, err := root.ReadFile(filepath.Join("profiles", info.ID, openProfile(info), "profile.json"))
	if err != nil {
		return state, profile, nil
	}
	var p struct {
		Entries []diskEntry `json:"entries"`
	}
	if json.Unmarshal(data, &p) != nil {
		return state, profile, nil
	}
	for _, e := range p.Entries {
		if e.Source.Kind == "thunderstore" && e.Source.Name != "" {
			packages = append(packages, e.Source.Name)
		}
		for _, m := range e.Mods {
			if m.ID.Format() == mod.FormatThunderstore {
				packages = append(packages, m.ID.Local())
			}
		}
	}
	slices.SortFunc(packages, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	return state, profile, slices.CompactFunc(packages, strings.EqualFold)
}

// queuePackage asks the running app to queue the newest version of the package, with its dependencies, into the
// community's open profile.
func queuePackage(key, pkg string) error {
	info, ok := gameBySourceKey("thunderstore", key)
	if !ok {
		return os.ErrNotExist
	}
	dataDir, err := datadir.Dir()
	if err != nil {
		return err
	}
	return controlwire.CallDir(dataDir, "installPackage", map[string]any{
		"game": info.ID, "profile": openProfile(info), "name": pkg,
	}, nil, installTimeout)
}
