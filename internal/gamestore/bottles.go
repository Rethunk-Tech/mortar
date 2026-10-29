package gamestore

import (
	"runtime"

	gamelocator "github.com/Rethunk-Tech/go-gamelocator"
	"github.com/Rethunk-Tech/go-gamelocator/bottles"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
)

const bottlesKey = "bottles"

type bottlesStore struct{}

func (bottlesStore) Key() string { return bottlesKey }

func (bottlesStore) Launchers(goos string) []LauncherSpec {
	if goos != "linux" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherBottles, Name: "Bottles", Usable: safefs.IsDir,
		Looked: func(home string, added map[string][]string) []string {
			return bottles.Dirs(home, added[LauncherBottles]...)
		},
	}}
}

// Discover looks through every bottle (a folder holding bottle.yml) for g's marker file.
func (bottlesStore) Discover(home string, roots map[string][]string, g components.GameInfo) []Install {
	if runtime.GOOS != "linux" || g.Stores.Bottles == nil {
		return nil
	}
	var out []Install
	for _, dir := range bottles.Dirs(home, roots[LauncherBottles]...) {
		for _, bottle := range bottles.List(dir) {
			for _, cand := range bottles.GameDirs(bottle, g.Stores.Bottles.Folder) {
				if found := gamelocator.GameDir(cand, g.MarkerPath()); found != "" {
					out = append(out, Install{Store: StoreBottles, Dir: found, Prefix: bottle})
					break
				}
			}
		}
	}
	return out
}
