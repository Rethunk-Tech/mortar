package game

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gamestore"
	"github.com/Rethunk-Tech/mortar/internal/settings"
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

// Launchers lists the launchers Mortar reads on this system, whether each was found, and the supported games in it.
func Launchers(home string, s settings.Settings) ([]StoreApp, error) {
	byLauncher := map[string][]StoreAppGame{}
	for _, g := range games {
		for _, in := range collect(g, home, s) {
			id := gamestore.LauncherOf(in.Store)
			byLauncher[id] = append(byLauncher[id], StoreAppGame{ID: g.ID(), Name: g.Name(), Dir: in.Dir})
		}
	}
	var out []StoreApp
	for _, spec := range gamestore.Launchers(runtime.GOOS) {
		custom := s.LauncherRoots[spec.ID]
		l := StoreApp{
			ID: spec.ID, Name: spec.Name, Looked: spec.Looked(home, s.LauncherRoots), Games: byLauncher[spec.ID],
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
			if err != nil || dir == "" || seen[resolved] || !spec.Usable(dir) {
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
	for _, spec := range gamestore.Launchers(runtime.GOOS) {
		if spec.ID != launcher {
			continue
		}
		if !fsx.IsDir(dir) {
			return fmt.Errorf("%s is not a folder", dir)
		}
		if !spec.Usable(dir) {
			switch launcher {
			case LauncherSteam, LauncherFlatpakSteam:
				return errors.New("this folder has no steamapps folder; choose the folder Steam is installed in")
			case LauncherHeroic:
				return errors.New("this folder has no gog_store folder; choose Heroic's config folder")
			}
			return fmt.Errorf("%s is not a %s folder", dir, spec.Name)
		}
		return nil
	}
	return fmt.Errorf("unknown launcher %q", launcher)
}

// Launchers lists the launchers on this system for the setup screen.
func (s *Service) Launchers() ([]StoreApp, error) { return Launchers(s.home, s.store.Get()) }
