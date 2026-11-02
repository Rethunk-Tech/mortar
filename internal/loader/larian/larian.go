// Package larian is the folder loader's extension for Larian Studios games: Baldur's Gate 3 reads its mods as .pak
// files from a Mods folder and takes their load order from a modsettings.lsx, which this writes from each pak's meta.lsx
// for the length of a launch.
package larian

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	larianpak "github.com/Rethunk-Tech/go-larian"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/folder"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	modpkg "github.com/Rethunk-Tech/mortar/internal/mod"
)

// Role is the catalog path role of the folder holding modsettings.lsx.
const Role = "modSettings"

const (
	settingsFile = "modsettings.lsx"
	// generatedDir is where the profile keeps the file it places: the deployer gives the player's own back afterwards.
	generatedDir = ".mortar-larian"
	unranked     = 1 << 20
)

var _ = folder.Extend(extension{})

type extension struct{}

func (extension) Role() string { return Role }

func (extension) Contribute(plan *launchplan.Plan, p loader.ProfileView) {
	plan.AddFile(launchplan.PlanFile{Src: generatedPath(p.Dir), Dst: settingsFile, Root: Role})
}

func generatedPath(profileDir string) string {
	return folder.GeneratedPath(profileDir, generatedDir, settingsFile)
}

// gustavX is the main campaign's entry, listed first as BG3 Mod Manager writes it (MAIN_CAMPAIGN_UUID in DivinityApp.cs).
var gustavX = larianpak.ModuleInfo{
	UUID: larianpak.GustavXUUID, Name: "GustavX", Folder: "GustavX", Version64: larianpak.NewVersion64(1, 0, 0, 0),
}

// listed is a module modsettings.lsx lists, with the pak it came from.
type listed struct {
	info larianpak.ModuleInfo
	pak  string
	// folder is the enabled mod folder that holds the pak; empty for a pak the profile does not hold.
	folder string
}

// modules are GustavX and then the module of each pak in the profile's mods folder, in the profile's mod order.
// A pak with no meta.lsx is an override the game reads without a list entry.
func modules(p loader.ProfileView) ([]listed, error) {
	root, err := folder.ProfileModsDir(p.Game, p.Dir)
	if err != nil {
		return nil, err
	}
	paks, err := pakPaths(root, false)
	if err != nil {
		return nil, err
	}
	rank := orderRanks(p.Enabled)
	slices.SortStableFunc(paks, func(a, b string) int {
		ra, rb := rankOf(rank, a), rankOf(rank, b)
		if ra != rb {
			return ra - rb
		}
		return strings.Compare(a, b)
	})
	mods := []listed{{info: gustavX}}
	seen := map[string]bool{strings.ToLower(gustavX.UUID): true}
	for _, path := range paks {
		mod, err := moduleOf(path)
		if err != nil {
			log.Printf("larian: %s could not read module info: %v", filepath.Base(path), err)
			continue
		}
		if seen[strings.ToLower(mod.UUID)] {
			continue
		}
		seen[strings.ToLower(mod.UUID)] = true
		l := listed{info: mod, pak: filepath.Base(path)}
		if r := rankOf(rank, path); r != unranked {
			l.folder = p.Enabled[r]
		}
		mods = append(mods, l)
	}
	return mods, nil
}

// Prelaunch writes the modules the load order lists.
func (extension) Prelaunch(p loader.ProfileView) error {
	mods, err := modules(p)
	if err != nil {
		return err
	}
	infos := make([]larianpak.ModuleInfo, len(mods))
	for i, m := range mods {
		infos[i] = m.info
	}
	return folder.WriteGenerated(p.Dir, generatedDir, settingsFile, larianpak.FormatModSettings(infos))
}

// Order is the list Prelaunch would write. Row.Folder names the profile's mod folder to move to change a row's place.
func (extension) Order(p loader.ProfileView) ([]loadorder.Row, error) {
	mods, err := modules(p)
	if err != nil {
		return nil, err
	}
	rows := make([]loadorder.Row, len(mods))
	for i, m := range mods {
		rows[i] = loadorder.Row{
			Position: i + 1, ID: modpkg.NewID(modpkg.FormatFolder, m.info.UUID), Name: m.info.Name,
			Version: m.info.Version64.String(), Detail: m.pak, Official: i == 0, Folder: m.folder,
		}
	}
	return rows, nil
}

func moduleOf(path string) (info larianpak.ModuleInfo, err error) {
	pak, err := larianpak.OpenFile(path)
	if err != nil {
		return larianpak.ModuleInfo{}, err
	}
	defer func() { err = errors.Join(err, pak.Close()) }()
	return pak.ReadModuleInfo()
}

// pakPaths lists the paks in root. BG3 loads only the top level of its Mods folder; the mod folders orderRanks reads
// may nest theirs.
func pakPaths(root string, deep bool) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && !deep && path != root {
			return fs.SkipDir
		}
		if d.Type().IsRegular() && strings.EqualFold(filepath.Ext(path), ".pak") {
			out = append(out, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return out, err
}

// orderRanks maps a pak's file name to the place of the first enabled mod folder that holds it.
func orderRanks(enabled []string) map[string]int {
	rank := map[string]int{}
	for i, dir := range enabled {
		paks, err := pakPaths(dir, true)
		if err != nil {
			continue
		}
		for _, path := range paks {
			name := strings.ToLower(filepath.Base(path))
			if _, ok := rank[name]; !ok {
				rank[name] = i
			}
		}
	}
	return rank
}

// rankOf puts a pak no enabled folder holds (one added by hand) after every ranked one.
func rankOf(rank map[string]int, path string) int {
	if r, ok := rank[strings.ToLower(filepath.Base(path))]; ok {
		return r
	}
	return unranked
}
