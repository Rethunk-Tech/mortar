package gamestore

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gog"
)

const bottlesKey = "bottles"

// bottlesDirs are the folders Bottles keeps its bottles in, the user's own first: native, then Flatpak.
func bottlesDirs(home string, extra ...string) []string {
	return append(append([]string(nil), extra...),
		filepath.Join(home, ".local", "share", "bottles", "bottles"),
		filepath.Join(home, ".var", "app", "com.usebottles.bottles", "data", "bottles", "bottles"),
	)
}

// bottleGameDirs are where a Steam or GOG install of folder sits inside a bottle's C: drive.
func bottleGameDirs(bottle, folder string) []string {
	c := filepath.Join(bottle, "drive_c")
	return []string{
		filepath.Join(c, "Program Files (x86)", "Steam", "steamapps", "common", folder),
		filepath.Join(c, "Program Files", "Steam", "steamapps", "common", folder),
		filepath.Join(c, "GOG Games", folder),
		filepath.Join(c, "Program Files (x86)", "GOG Galaxy", "Games", folder),
	}
}

type bottlesStore struct{}

func (bottlesStore) Key() string { return bottlesKey }

func (bottlesStore) Launchers(goos string) []LauncherSpec {
	if goos != "linux" {
		return nil
	}
	return []LauncherSpec{{
		ID: LauncherBottles, Name: "Bottles", Usable: fsx.IsDir,
		Looked: func(home string, added map[string][]string) []string {
			return bottlesDirs(home, added[LauncherBottles]...)
		},
	}}
}

// Discover looks through every bottle (a folder holding bottle.yml) for g's marker file.
func (bottlesStore) Discover(home string, roots map[string][]string, g components.GameInfo) []Install {
	if runtime.GOOS != "linux" || g.Stores.Bottles == nil {
		return nil
	}
	var out []Install
	for _, dir := range bottlesDirs(home, roots[LauncherBottles]...) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			bottle := filepath.Join(dir, ent.Name())
			if !fsx.IsDir(bottle) || !fsx.IsFile(filepath.Join(bottle, "bottle.yml")) {
				continue
			}
			for _, cand := range bottleGameDirs(bottle, g.Stores.Bottles.Folder) {
				if found := gog.GameDir(cand, g.Marker); found != "" {
					out = append(out, Install{Store: StoreBottles, Dir: found, Prefix: bottle})
					break
				}
			}
		}
	}
	return out
}

// BottleOf is the bottle that holds dir, found by walking up to a folder with bottle.yml; "" when dir is in none.
func BottleOf(dir string) string {
	for d := filepath.Clean(dir); ; d = filepath.Dir(d) {
		if fsx.IsFile(filepath.Join(d, "bottle.yml")) {
			return d
		}
		if filepath.Dir(d) == d {
			return ""
		}
	}
}
