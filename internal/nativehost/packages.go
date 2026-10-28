package nativehost

import (
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
	"github.com/Rethunk-Tech/mortar/internal/source/modrinth"
	"github.com/Rethunk-Tech/mortar/internal/source/paradox"
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

// answerProjects replies to installedProjects for a CurseForge, Mod DB, Paradox or Modrinth game key. A key Mortar does not manage (the
// extension's probe sends none) still gets the games list.
func (h handlers) answerProjects(req request) reply {
	if req.Source == "nexus" || req.Source == "thunderstore" || !req.reads(req.Source) || h.projects == nil {
		return reply{Error: "Mortar does not read " + req.Source + " pages"}
	}
	st, name, found := h.projects(req.Source, req.SourceGameKey)
	rep := reply{State: st, Connected: st == stateReady, Profile: name, Games: hostGames()}
	if st != stateOff {
		if found == nil {
			found = []hostProject{}
		}
		rep.Projects = &found
	}
	return rep
}

// answerInstallProject queues a Paradox or Modrinth project into the open profile, refusing an id the source's own
// pattern rejects before anything reaches the app.
func (h handlers) answerInstallProject(req request) reply {
	var valid bool
	switch req.Source {
	case "paradox":
		valid = paradox.ValidID(req.Project)
	case "modrinth":
		valid = modrinth.ValidID(req.Project)
	default:
		return reply{Error: "Mortar does not install from " + req.Source + " pages"}
	}
	if !valid || h.projects == nil {
		return reply{Error: "not a " + req.Source + " project id"}
	}
	st, name, _ := h.projects(req.Source, req.SourceGameKey)
	rep := reply{State: st, Connected: st == stateReady, Profile: name, Games: hostGames()}
	switch {
	case st == stateOff:
	case st != stateReady:
		rep.Error = "open a profile for this game in Mortar first"
	case h.installProject == nil:
		rep.Error = "Mortar cannot install projects here"
	default:
		if err := h.installProject(req.Source, req.SourceGameKey, req.Project); err != nil {
			rep.Error = err.Error()
		} else {
			rep.OK = true
		}
	}
	return rep
}

func gameBySourceKey(source, key string) (components.GameInfo, bool) {
	for _, g := range components.Games() {
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
	info, found := gameBySourceKey("thunderstore", key)
	state, profile, dataDir, ok := readyProfile(info, found)
	if !ok {
		return state, profile, nil
	}
	p, ok := readOpenProfile(dataDir, info)
	if !ok {
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
	return installVia("thunderstore", key, map[string]any{"name": pkg})
}

// queueProject asks the running app to queue the project's newest version, with its dependencies, into the source's
// game's open profile.
func queueProject(src, key, project string) error {
	return installVia(src, key, map[string]any{"name": project, "source": src})
}

// installVia asks the running app to install args' package into the open profile of the source's game.
func installVia(src, key string, args map[string]any) error {
	info, ok := gameBySourceKey(src, key)
	if !ok {
		return os.ErrNotExist
	}
	dataDir, err := datadir.Dir()
	if err != nil {
		return err
	}
	args["game"], args["profile"] = info.ID, openProfile(info)
	return controlwire.CallDir(dataDir, "installPackage", args, nil, installTimeout)
}

// readyProfile says whether the game has an open profile ready to read, with the data folder when it has; state and
// profile are what the extension shows either way.
func readyProfile(info components.GameInfo, found bool) (state, profile, dataDir string, ok bool) {
	if !found {
		return stateNoProfile, "", "", false
	}
	state, profile = stateOf(info)
	if state != stateReady {
		return state, profile, "", false
	}
	dataDir, err := datadir.Dir()
	return state, profile, dataDir, err == nil
}

// activeProjects lists the CurseForge, Mod DB, Paradox or Modrinth projects the source's game holds in its open
// profile. A CurseForge entry's Name is the project id; a Mod DB entry's is "<modid>-<assetid>", which pages are
// addressed by; a Paradox entry's is the mod id and a Modrinth entry's the project id.
func activeProjects(src, key string) (state, profile string, projects []hostProject) {
	info, found := gameBySourceKey(src, key)
	state, profile, dataDir, ok := readyProfile(info, found)
	if !ok {
		return state, profile, nil
	}
	p, ok := readOpenProfile(dataDir, info)
	if !ok {
		return state, profile, nil
	}
	var driverIDs []string
	for _, e := range p.Entries {
		if e.Source.Kind != src || e.Source.Name == "" {
			continue
		}
		id := e.Source.Name
		if src == "vsmoddb" {
			var found bool
			if _, id, found = strings.Cut(id, "-"); !found || id == "" {
				continue
			}
		}
		driverIDs = append(driverIDs, e.Source.Name)
		name := e.Source.ModName
		if name == "" && len(e.Mods) > 0 {
			name = e.Mods[0].Name
		}
		projects = append(projects, hostProject{ID: id, Name: name})
	}
	if src != "paradox" {
		withSlugs(src, projects, driverIDs)
	}
	return state, profile, projects
}
