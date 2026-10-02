package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
	// Root is the folder it was found in, or the user's chosen folder when that one is not usable.
	Root string `json:"root"`
	// Custom is whether the user chose Root.
	Custom bool `json:"custom"`
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
	// looked lists the folders searched for this launcher, custom folder first.
	looked func(home, custom string) []string
	// usable reports whether a folder is this launcher's.
	usable func(dir string) bool
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func withCustom(custom string, rest ...string) []string {
	if custom == "" {
		return rest
	}
	return append([]string{custom}, rest...)
}

func launcherSpecs() []launcherSpec {
	steamDir := func(dir string) bool { return isDir(filepath.Join(dir, "steamapps")) }
	specs := []launcherSpec{{
		id: LauncherSteam, name: "Steam", usable: steamDir,
		looked: func(home, custom string) []string { return steam.Roots(home, withCustom(custom)...) },
	}}
	if runtime.GOOS == "linux" {
		specs = append(specs,
			launcherSpec{
				id: LauncherFlatpakSteam, name: "Steam (Flatpak)", usable: steamDir,
				looked: func(home, _ string) []string { return []string{steam.FlatpakRoot(home)} },
			},
			launcherSpec{
				id: LauncherHeroic, name: "Heroic", usable: func(dir string) bool { return isDir(filepath.Join(dir, "gog_store")) },
				looked: gog.HeroicDirs,
			},
			launcherSpec{
				id: LauncherLutris, name: "Lutris", usable: isDir,
				looked: func(home, custom string) []string { return lutris.ConfigDirs(home, withCustom(custom)...) },
			},
		)
	}
	gogSpec := launcherSpec{id: LauncherGOG, name: "GOG", usable: isDir, looked: gog.GamesDirs}
	if runtime.GOOS == "windows" {
		gogSpec.name = "GOG Galaxy"
		gogSpec.looked = func(home, custom string) []string {
			return append(withCustom(custom, gog.GalaxyDir()), gog.GamesDirs(home, "")...)
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
		l := StoreApp{ID: spec.id, Name: spec.name, Looked: spec.looked(home, custom), Games: byLauncher[spec.id], Custom: custom != ""}
		if l.Games == nil {
			l.Games = []StoreAppGame{}
		}
		for _, dir := range l.Looked {
			if dir != "" && spec.usable(dir) {
				l.Found, l.Root = true, dir
				break
			}
		}
		if !l.Found && custom != "" {
			l.Root = custom
		}
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
