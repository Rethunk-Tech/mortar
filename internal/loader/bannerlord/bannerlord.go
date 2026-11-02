// Package bannerlord is the folder loader's extension for Mount & Blade II: Bannerlord. The game reads each module
// folder in its Modules folder and takes the modules to load, in order, from the launcher's LauncherData.xml; this
// writes that list for the length of a launch and reports what its dependency sort finds wrong.
package bannerlord

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	bl "github.com/Rethunk-Tech/go-bannerlord"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/folder"
)

// Role is the catalog path role of the folder holding LauncherData.xml.
const Role = "launcherData"

const (
	launcherFile = "LauncherData.xml"
	// generatedDir is where the profile keeps the file it places: the deployer gives the player's own back afterwards.
	generatedDir = ".mortar-bannerlord"
	submodule    = "SubModule.xml"
	modulesDir   = "Modules"
)

// Finding codes, worded in the frontend's reasonText.
const (
	CodeMissing      = "moduleMissing"
	CodeOutdated     = "moduleOutdated"
	CodeIncompatible = "moduleIncompatible"
	CodeDuplicate    = "moduleDuplicate"
	CodeCycle        = "moduleCycle"
)

var _ = folder.Extend(extension{})

type extension struct{}

func (extension) Role() string { return Role }

func (extension) Contribute(plan *launchplan.Plan, p loader.ProfileView) {
	plan.AddFile(launchplan.PlanFile{Src: generatedPath(p.Dir), Dst: launcherFile, Root: Role})
}

func generatedPath(profileDir string) string {
	return folder.GeneratedPath(profileDir, generatedDir, launcherFile)
}

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
	entries, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
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
			log.Printf("bannerlord: %s is not a module: %v", e.Name(), err)
			continue
		}
		out = append(out, mod)
	}
	return out, nil
}

// emptyLauncher is the list the launcher keeps before the player has changed anything.
const emptyLauncher = `<UserData><SingleplayerData><ModDatas/></SingleplayerData></UserData>`

// Prelaunch writes a LauncherData.xml whose singleplayer list selects the profile's modules and the official modules
// the game starts with (and those a profile module needs), in the order the dependency sort gives.
func (extension) Prelaunch(p loader.ProfileView) error {
	m, err := load(p)
	if err != nil {
		return err
	}
	order, _ := bl.Sort(m.all)
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
	selected := order[:0:0]
	for _, id := range order {
		mod := byID[id]
		if m.profile[id] || mod.Official && (mod.Default || needed[id]) {
			selected = append(selected, id)
		}
	}
	l, err := bl.ParseLauncher(strings.NewReader(emptyLauncher))
	if err != nil {
		return err
	}
	if err := l.SetMods(bl.Singleplayer, selected); err != nil {
		return err
	}
	b, err := l.Marshal()
	if err != nil {
		return err
	}
	return folder.WriteGenerated(p.Dir, generatedDir, launcherFile, b)
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
