// Package launchplan is what a launch is assembled from before anything starts: loaders contribute arguments,
// environment, runtime requirements and install-side files to a Plan, and the pipeline then runs it.
package launchplan

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Mode is how the game is started.
type Mode string

const (
	// ModeProfile starts the game with the profile's mods.
	ModeProfile Mode = "profile"
	// ModeVanilla starts the game without mods.
	ModeVanilla Mode = "vanilla"
	// ModeTool starts a modding tool (xEdit, BodySlide) with the profile's files deployed.
	ModeTool Mode = "tool"
)

// RuntimeReq is something the runtime must provide before the game starts.
type RuntimeReq struct {
	// Kind is dll-override, java or env.
	Kind, Key, Value string
}

// PlanFile is a file the deployer places for a launch: Src is an absolute path in the profile, Dst a path relative to
// the game folder, or, when Root names a catalog path role (a game's mods folder outside the install), to that folder.
type PlanFile struct {
	Src, Dst string
	Root     string
}

// VersionManifest describes a launcher profile a loader injects (Minecraft's versions/<id>/ file).
type VersionManifest struct {
	ID, InheritsFrom, MainClass string
	Libraries, Arguments        []string
}

// ErrSecondOverride is a plan in which two loaders both want to replace the executable or the version manifest.
var ErrSecondOverride = errors.New("another loader already replaces the game's launcher")

// Plan is one launch. Loaders only add to it, through the methods below.
type Plan struct {
	Mode Mode
	Exe  string
	// Entry is the executable a direct start runs when no loader replaces Exe. A store that relays the start lets the
	// game's own launcher run instead.
	Entry        string
	Args         []string
	Env          map[string]string
	Prefix       []string
	RuntimeReqs  []RuntimeReq
	Files        []PlanFile
	ProcessNames []string
	Version      *VersionManifest

	owner string
}

// New is an empty plan for mode.
func New(mode Mode) *Plan { return &Plan{Mode: mode, Env: map[string]string{}} }

// SetEntry names the executable a direct start of the game runs.
func (p *Plan) SetEntry(exe string) { p.Entry = exe }

// AddArgs appends game arguments.
func (p *Plan) AddArgs(args ...string) { p.Args = append(p.Args, args...) }

// SetEnv sets an environment variable of the game process.
func (p *Plan) SetEnv(key, value string) { p.Env[key] = value }

// AddFile declares an install-side file for the deployer.
func (p *Plan) AddFile(f PlanFile) { p.Files = append(p.Files, f) }

// AddProcessName names an executable the running game can be found by.
func (p *Plan) AddProcessName(name string) {
	if !slices.Contains(p.ProcessNames, name) {
		p.ProcessNames = append(p.ProcessNames, name)
	}
}

// RequireRuntime asks the runtime for req; asking twice for the same thing is one request.
func (p *Plan) RequireRuntime(req RuntimeReq) {
	if !slices.Contains(p.RuntimeReqs, req) {
		p.RuntimeReqs = append(p.RuntimeReqs, req)
	}
}

// ApplyDLLOverrides sets WINEDLLOVERRIDES for a Wine-based runtime from the plan's dll-override requests, ahead of
// any value the plan already has, so a game whose prefix Wine has not made yet loads the loader's proxy DLL on its
// first start. A library the existing value already names keeps the player's setting.
func (p *Plan) ApplyDLLOverrides() {
	have := p.Env["WINEDLLOVERRIDES"]
	var add []string
	for _, r := range p.RuntimeReqs {
		if r.Kind != "dll-override" || strings.Contains(strings.ToLower(have), strings.ToLower(r.Key)+"=") {
			continue
		}
		add = append(add, r.Key+"="+wineLoadOrder(r.Value))
	}
	if len(add) > 0 {
		p.Env["WINEDLLOVERRIDES"] = strings.Join(append(add, have), ";")
		p.Env["WINEDLLOVERRIDES"] = strings.TrimSuffix(p.Env["WINEDLLOVERRIDES"], ";")
	}
}

// wineLoadOrder is a registry load order ("native,builtin") as WINEDLLOVERRIDES spells it ("n,b").
func wineLoadOrder(order string) string {
	parts := strings.Split(order, ",")
	for i, o := range parts {
		if o != "" {
			parts[i] = o[:1]
		}
	}
	return strings.Join(parts, ",")
}

func (p *Plan) claim(owner string) error {
	if p.owner != "" {
		return fmt.Errorf("%w: %s, then %s", ErrSecondOverride, p.owner, owner)
	}
	p.owner = owner
	return nil
}

// OverrideExe makes owner's executable the one that starts, refusing when another loader already overrides the
// executable or sets a version manifest.
func (p *Plan) OverrideExe(owner, exe string) error {
	if err := p.claim(owner); err != nil {
		return err
	}
	p.Exe = exe
	return nil
}

// SetVersion makes owner's version manifest the one that starts, under the same one-owner rule as OverrideExe.
func (p *Plan) SetVersion(owner string, v VersionManifest) error {
	if err := p.claim(owner); err != nil {
		return err
	}
	p.Version = &v
	return nil
}
