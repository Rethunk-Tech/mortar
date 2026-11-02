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
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/folder"
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
	return filepath.Join(profileDir, generatedDir, settingsFile)
}

// gustavX is the main campaign's entry, listed first as BG3 Mod Manager writes it (MAIN_CAMPAIGN_UUID in DivinityApp.cs).
var gustavX = larianpak.ModuleInfo{
	UUID: larianpak.GustavXUUID, Name: "GustavX", Folder: "GustavX", Version64: larianpak.NewVersion64(1, 0, 0, 0),
}

// Prelaunch lists, after GustavX, the module of each pak in the profile's mods folder in the profile's mod order.
// A pak with no meta.lsx is an override the game reads without a list entry.
func (extension) Prelaunch(p loader.ProfileView) error {
	info, ok := components.Game(p.Game)
	if !ok {
		return errors.New("unknown game " + p.Game)
	}
	target, ok := info.Target(folder.TargetID)
	if !ok {
		return errors.New(info.Name + " has no " + folder.TargetID + " target")
	}
	paks, err := pakPaths(filepath.Join(p.Dir, filepath.FromSlash(target.ProfileFolder())))
	if err != nil {
		return err
	}
	rank := orderRanks(p.Enabled)
	slices.SortStableFunc(paks, func(a, b string) int {
		ra, rb := rankOf(rank, a), rankOf(rank, b)
		if ra != rb {
			return ra - rb
		}
		return strings.Compare(a, b)
	})
	mods := []larianpak.ModuleInfo{gustavX}
	seen := map[string]bool{strings.ToLower(gustavX.UUID): true}
	for _, path := range paks {
		mod, err := moduleOf(path)
		if err != nil {
			log.Printf("larian: %s is not listed in modsettings.lsx: %v", filepath.Base(path), err)
			continue
		}
		if seen[strings.ToLower(mod.UUID)] {
			continue
		}
		seen[strings.ToLower(mod.UUID)] = true
		mods = append(mods, mod)
	}
	out := generatedPath(p.Dir)
	if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
		return err
	}
	return safefs.WriteFileAtomic(out, larianpak.FormatModSettings(mods), 0o600)
}

func moduleOf(path string) (info larianpak.ModuleInfo, err error) {
	pak, err := larianpak.OpenFile(path)
	if err != nil {
		return larianpak.ModuleInfo{}, err
	}
	defer func() { err = errors.Join(err, pak.Close()) }()
	return pak.ReadModuleInfo()
}

func pakPaths(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
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
		paks, err := pakPaths(dir)
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
