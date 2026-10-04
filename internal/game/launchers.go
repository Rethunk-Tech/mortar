package game

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/gog"
	"github.com/Rethunk-AI/mortar/internal/lutris"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/steam"
)

// StoreApp is a launcher (a store app) that tells Mortar which games are installed.
type StoreApp struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Found bool   `json:"found"`
	// Roots are every searched folder that is this launcher's, in search order.
	Roots []string `json:"roots"`
	// Custom are the folders the user added, usable or not.
	Custom []string `json:"custom"`
	// Looked lists every folder searched, in order.
	Looked []string       `json:"looked"`
	Games  []StoreAppGame `json:"games"`
}

// StoreAppGame is a supported game a launcher holds.
type StoreAppGame struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Dir  string `json:"dir"`
}

// storeLauncher maps each install's store to the launcher that reported it.
var storeLauncher = map[string]string{
	StoreSteam: LauncherSteam, StoreFlatpakSteam: LauncherFlatpakSteam, StoreGOGHeroic: LauncherHeroic,
	StoreLutris: LauncherLutris, StoreGOG: LauncherGOG, StoreMinigalaxy: LauncherMinigalaxy,
}

type launcherSpec struct {
	id, name string
	// looked lists the folders searched for this launcher, the user's own first; added holds every launcher's added
	// folders, since one launcher's folder can rule out another's.
	looked func(home string, added map[string][]string) []string
	// usable reports whether a folder is this launcher's.
	usable func(dir string) bool
}

func withCustom(custom []string, rest ...string) []string {
	return append(slices.Clone(custom), rest...)
}

// launcherSpecs are the launchers Mortar reads on goos.
func launcherSpecs(goos string) []launcherSpec {
	steamDir := func(dir string) bool { return fsx.IsDir(filepath.Join(dir, "steamapps")) }
	specs := []launcherSpec{{
		id: LauncherSteam, name: "Steam", usable: steamDir,
		looked: func(home string, added map[string][]string) []string {
			return steam.Roots(home, added[LauncherSteam]...)
		},
	}}
	if goos == "linux" {
		specs = append(specs, launcherSpec{
			id: LauncherFlatpakSteam, name: "Steam (Flatpak)", usable: steamDir,
			looked: func(home string, _ map[string][]string) []string { return []string{steam.FlatpakRoot(home)} },
		})
	}
	specs = append(specs, launcherSpec{
		id: LauncherHeroic, name: "Heroic", usable: func(dir string) bool { return fsx.IsDir(filepath.Join(dir, "gog_store")) },
		looked: func(home string, added map[string][]string) []string {
			return gog.HeroicDirs(home, added[LauncherHeroic]...)
		},
	})
	if goos == "linux" {
		specs = append(specs, launcherSpec{
			id: LauncherLutris, name: "Lutris", usable: fsx.IsDir,
			looked: func(home string, added map[string][]string) []string {
				return lutris.ConfigDirs(home, added[LauncherLutris]...)
			},
		}, launcherSpec{
			id: LauncherMinigalaxy, name: "Minigalaxy", usable: fsx.IsDir,
			looked: func(home string, added map[string][]string) []string {
				return withCustom(added[LauncherMinigalaxy], gog.MinigalaxyConfigDirs(home)...)
			},
		})
	}
	gogSpec := launcherSpec{
		id: LauncherGOG, name: "GOG", usable: fsx.IsDir,
		looked: func(home string, added map[string][]string) []string {
			return gog.OfflineDirs(home, gog.Roots{Games: added[LauncherGOG], Minigalaxy: added[LauncherMinigalaxy]})
		},
	}
	if goos == "windows" {
		gogSpec.name = "GOG Galaxy"
		gogSpec.looked = func(home string, added map[string][]string) []string {
			return append(withCustom(added[LauncherGOG], gog.GalaxyDir()), gog.GamesDirs(home)...)
		}
	}
	return append(specs, gogSpec)
}

// Launchers lists the launchers Mortar reads on this system, whether each was found, and the supported games in it.
func Launchers(home string, s settings.Settings) ([]StoreApp, error) {
	byLauncher := map[string][]StoreAppGame{}
	for _, g := range games {
		for _, in := range collect(g, home, s) {
			id := storeLauncher[in.Store]
			byLauncher[id] = append(byLauncher[id], StoreAppGame{ID: g.ID(), Name: g.Name(), Dir: in.Dir})
		}
	}
	var out []StoreApp
	for _, spec := range launcherSpecs(runtime.GOOS) {
		custom := s.LauncherRoots[spec.id]
		l := StoreApp{
			ID: spec.id, Name: spec.name, Looked: spec.looked(home, s.LauncherRoots), Games: byLauncher[spec.id],
			Roots: []string{}, Custom: slices.Clone(custom),
		}
		if l.Games == nil {
			l.Games = []StoreAppGame{}
		}
		if l.Custom == nil {
			l.Custom = []string{}
		}
		seen := map[string]bool{}
		for _, dir := range l.Looked {
			resolved, err := filepath.EvalSymlinks(dir)
			if err != nil || dir == "" || seen[resolved] || !spec.usable(dir) {
				continue
			}
			seen[resolved] = true
			l.Roots = append(l.Roots, dir)
		}
		l.Found = len(l.Roots) > 0
		out = append(out, l)
	}
	return out, nil
}

// ValidateLauncherRoot checks that dir is a folder of the named launcher before it is saved.
func ValidateLauncherRoot(launcher, dir string) error {
	for _, spec := range launcherSpecs(runtime.GOOS) {
		if spec.id != launcher {
			continue
		}
		if !fsx.IsDir(dir) {
			return fmt.Errorf("%s is not a folder", dir)
		}
		if !spec.usable(dir) {
			switch launcher {
			case LauncherSteam, LauncherFlatpakSteam:
				return errors.New("this folder has no steamapps folder; choose the folder Steam is installed in")
			case LauncherHeroic:
				return errors.New("this folder has no gog_store folder; choose Heroic's config folder")
			}
			return fmt.Errorf("%s is not a %s folder", dir, spec.name)
		}
		return nil
	}
	return fmt.Errorf("unknown launcher %q", launcher)
}

// Launchers lists the launchers on this system for the setup screen.
func (s *Service) Launchers() ([]StoreApp, error) { return Launchers(s.home, s.store.Get()) }
