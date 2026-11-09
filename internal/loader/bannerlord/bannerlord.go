// Package bannerlord is the folder loader's extension for Mount & Blade II: Bannerlord. The game reads each module
// folder in its Modules folder and takes the modules to load, in order, from the launcher's LauncherData.xml; this
// writes that list for the length of a launch and reports what its dependency sort finds wrong.
package bannerlord

import (
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	bl "github.com/Rethunk-Tech/go-bannerlord"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/folder"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	modpkg "github.com/Rethunk-Tech/mortar/internal/mod"
)

// Role is the catalog path role of the folder holding LauncherData.xml.
const Role = "launcherData"

const (
	submodule  = "SubModule.xml"
	modulesDir = "Modules"
)

// Finding codes, worded in the frontend's reasonText.
const (
	CodeMissing      = "moduleMissing"
	CodeOutdated     = "moduleOutdated"
	CodeIncompatible = "moduleIncompatible"
	CodeDuplicate    = "moduleDuplicate"
	CodeCycle        = "moduleCycle"
)

var ext = extension{folder.Generated{RoleName: Role, Dir: ".mortar-bannerlord", File: "LauncherData.xml"}}

var _ = folder.Extend(ext)

type extension struct{ folder.Generated }

// modules are the game's official modules and the profile's, in that order.
type modules struct {
	all []bl.Module
	// profile holds the ids of the profile's own modules.
	profile map[string]bool
}

func load(p loader.ProfileView) (modules, error) {
	root, err := folder.ProfileModsDir(p.Game, p.Dir)
	if err != nil {
		return modules{}, err
	}
	var m modules
	official, err := readModules(filepath.Join(p.InstallDir, modulesDir))
	if err != nil {
		return modules{}, err
	}
	for _, mod := range official {
		if mod.Official {
			m.all = append(m.all, mod)
		}
	}
	own, err := readModules(root)
	if err != nil {
		return modules{}, err
	}
	m.profile = map[string]bool{}
	for _, mod := range own {
		m.profile[mod.ID] = true
	}
	m.all = append(m.all, own...)
	return m, nil
}

// readModules parses the SubModule.xml of each folder directly below root, in name order. A folder whose file does not
// parse is not a module the game could load.
func readModules(root string) ([]bl.Module, error) {
	entries, found, err := safefs.ReadDirIfExists(root)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	var out []bl.Module
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		f, err := safefs.Open(filepath.Join(root, e.Name(), submodule))
		if err != nil {
			continue
		}
		mod, err := bl.ParseModule(f)
		_ = f.Close()
		if err != nil {
			slog.Warn("bannerlord: not a module", "name", e.Name(), "err", err)
			continue
		}
		out = append(out, mod)
	}
	return out, nil
}

// emptyLauncher is the list the launcher keeps before the player has changed anything.
const emptyLauncher = `<UserData><SingleplayerData><ModDatas/></SingleplayerData></UserData>`

// selection is the modules the launcher selects, in the dependency sort's order, with the sort's issues.
func selection(m modules) (selected []bl.Module, issues []bl.Issue) {
	order, issues := bl.Sort(m.all)
	byID := map[string]bl.Module{}
	for _, mod := range m.all {
		if _, dup := byID[mod.ID]; !dup {
			byID[mod.ID] = mod
		}
	}
	needed := map[string]bool{}
	queue := slices.Collect(maps.Keys(m.profile))
	for len(queue) > 0 {
		id := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, d := range byID[id].Depends {
			if !needed[d.ID] {
				needed[d.ID] = true
				queue = append(queue, d.ID)
			}
		}
	}
	for _, id := range order {
		mod := byID[id]
		if m.profile[id] || mod.Official && (mod.Default || needed[id]) {
			selected = append(selected, mod)
		}
	}
	return selected, issues
}

// Prelaunch writes a LauncherData.xml whose singleplayer list selects the profile's modules and the official modules
// the game starts with (and those a profile module needs), in the order the dependency sort gives.
func (extension) Prelaunch(p loader.ProfileView) error {
	m, err := load(p)
	if err != nil {
		return err
	}
	selected, _ := selection(m)
	ids := make([]string, len(selected))
	for i, mod := range selected {
		ids[i] = mod.ID
	}
	l, err := bl.ParseLauncher(strings.NewReader(emptyLauncher))
	if err != nil {
		return err
	}
	if err := l.SetMods(bl.Singleplayer, ids); err != nil {
		return err
	}
	b, err := l.Marshal()
	if err != nil {
		return err
	}
	return ext.Write(p.Dir, b)
}

// Order is the list Prelaunch would write, each module with the sort issues that name it. The sort alone orders it, so
// no row names a profile folder to move.
func (extension) Order(p loader.ProfileView) ([]loadorder.Row, error) {
	m, err := load(p)
	if err != nil {
		return nil, err
	}
	selected, issues := selection(m)
	rows := make([]loadorder.Row, len(selected))
	for i, mod := range selected {
		row := loadorder.Row{
			Position: i + 1, ID: modID(mod.ID), Name: mod.Name, Version: mod.Version.String(), Official: mod.Official, Default: mod.Default,
		}
		for _, d := range mod.Depends {
			if !d.Optional {
				row.Required = append(row.Required, modID(d.ID))
			}
		}
		for _, is := range issues {
			if is.Module == mod.ID || is.Kind == bl.CycleDetected && slices.Contains(is.Modules, mod.ID) {
				f := finding(is)
				row.Issues = append(row.Issues, loadorder.Issue{Code: f.Code, Message: f.Message, Params: f.Params})
				if is.Kind == bl.MissingDependency {
					row.MissingRequired = append(row.MissingRequired, modID(is.Other))
				}
				row.Cycle = row.Cycle || is.Kind == bl.CycleDetected
			}
		}
		rows[i] = row
	}
	return rows, nil
}

// Findings are the problems the dependency sort finds among the profile's modules and the official ones. A problem
// among official modules alone is the game's, not the profile's.
func (extension) Findings(p loader.ProfileView) []loader.Finding {
	m, err := load(p)
	if err != nil {
		return nil
	}
	_, issues := bl.Sort(m.all)
	var out []loader.Finding
	for _, is := range issues {
		if !m.profile[is.Module] && !slices.ContainsFunc(is.Modules, func(id string) bool { return m.profile[id] }) {
			continue
		}
		out = append(out, finding(is))
	}
	return out
}

func modID(id string) modpkg.ID { return modpkg.NewID(modpkg.FormatModInfo, id) }

func finding(is bl.Issue) loader.Finding {
	f := loader.Finding{Plugin: is.Module, Message: is.String(), Dependency: is.Other, Params: map[string]string{"other": is.Other, "detail": is.Detail}}
	switch is.Kind {
	case bl.MissingDependency:
		f.Kind, f.Code, f.Message = "missing-dependency", CodeMissing, "requires "+is.Other+", which is not installed"
	case bl.OutdatedDependency:
		f.Kind, f.Code, f.Message = "incompatible-version", CodeOutdated, fmt.Sprintf("requires %s %s", is.Other, is.Detail)
	case bl.IncompatibleModule:
		f.Kind, f.Code, f.Message = "incompatible-plugin", CodeIncompatible, "is incompatible with "+is.Other
	case bl.DuplicateModule:
		f.Kind, f.Code, f.Message = "load-exception", CodeDuplicate, "is installed more than once"
	case bl.CycleDetected:
		f.Plugin = is.Modules[0]
		f.Kind, f.Code, f.Message = "load-exception", CodeCycle, "dependency cycle among "+strings.Join(is.Modules, ", ")
		f.Params["list"] = strings.Join(is.Modules, ", ")
	}
	return f
}
