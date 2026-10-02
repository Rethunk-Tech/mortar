package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"

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
	StoreLutris: LauncherLutris, StoreGOG: LauncherGOG,
}

type launcherSpec struct {
	id, name string
	// looked lists the folders searched for this launcher, the user's own first.
	looked func(home string, custom []string) []string
	// usable reports whether a folder is this launcher's.
	usable func(dir string) bool
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func withCustom(custom []string, rest ...string) []string {
	return append(slices.Clone(custom), rest...)
}

func launcherSpecs() []launcherSpec {
	steamDir := func(dir string) bool { return isDir(filepath.Join(dir, "steamapps")) }
	specs := []launcherSpec{{
		id: LauncherSteam, name: "Steam", usable: steamDir,
		looked: func(home string, custom []string) []string { return steam.Roots(home, custom...) },
	}}
	if runtime.GOOS == "linux" {
		specs = append(specs, launcherSpec{
			id: LauncherFlatpakSteam, name: "Steam (Flatpak)", usable: steamDir,
			looked: func(home string, _ []string) []string { return []string{steam.FlatpakRoot(home)} },
		})
	}
	specs = append(specs, launcherSpec{
		id: LauncherHeroic, name: "Heroic", usable: func(dir string) bool { return isDir(filepath.Join(dir, "gog_store")) },
		looked: func(home string, custom []string) []string { return gog.HeroicDirs(home, custom...) },
	})
	if runtime.GOOS == "linux" {
		specs = append(specs, launcherSpec{
			id: LauncherLutris, name: "Lutris", usable: isDir,
			looked: func(home string, custom []string) []string { return lutris.ConfigDirs(home, custom...) },
		})
	}
	gogSpec := launcherSpec{
		id: LauncherGOG, name: "GOG", usable: isDir,
		looked: func(home string, custom []string) []string { return gog.GamesDirs(home, custom...) },
	}
	if runtime.GOOS == "windows" {
		gogSpec.name = "GOG Galaxy"
		gogSpec.looked = func(home string, custom []string) []string {
			return append(withCustom(custom, gog.GalaxyDir()), gog.GamesDirs(home)...)
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
	for _, spec := range launcherSpecs() {
		custom := s.LauncherRoots[spec.id]
		l := StoreApp{
			ID: spec.id, Name: spec.name, Looked: spec.looked(home, custom), Games: byLauncher[spec.id],
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
			real, err := filepath.EvalSymlinks(dir)
			if err != nil || dir == "" || seen[real] || !spec.usable(dir) {
				continue
			}
			seen[real] = true
			l.Roots = append(l.Roots, dir)
		}
		l.Found = len(l.Roots) > 0
		out = append(out, l)
	}
	return out, nil
}

// ValidateLauncherRoot checks that dir is a folder of the named launcher before it is saved.
func ValidateLauncherRoot(launcher, dir string) error {
	for _, spec := range launcherSpecs() {
		if spec.id != launcher {
			continue
		}
		if !isDir(dir) {
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
