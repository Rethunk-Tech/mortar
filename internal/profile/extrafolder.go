package profile

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
)

func (s *Service) extraModsFolders(game string) []string {
	return s.settings.Get().GamePrefs(game).ExtraModsFolderList()
}

// ExtraFolderMods lists the mods in the game's extra mods folders (setting extraModsFolder) as Import from the game's
// Mods folder previews them, each mod's Folder inside the folder it came from; none set lists none.
func (s *Service) ExtraFolderMods(game string) (GameModsPreview, error) {
	out := GameModsPreview{Mods: []GameModPreview{}}
	for _, dir := range s.extraModsFolders(game) {
		got, err := s.store.previewExtraFolder(game, dir)
		if err != nil {
			return GameModsPreview{}, err
		}
		out.Mods = append(out.Mods, got.Mods...)
		out.CanMove = out.CanMove || got.CanMove
	}
	return out, nil
}

// InstallExtraFolderMod installs folder, a top-level folder of one extra mods folder as ExtraFolderMods returned it,
// into the profile the way a dropped folder is installed.
func (s *Service) InstallExtraFolderMod(ctx context.Context, game, id, folder string) (InstallResult, error) {
	roots := s.extraModsFolders(game)
	if len(roots) == 0 {
		return InstallResult{}, errors.New("no extra mods folder is set")
	}
	for _, root := range roots {
		// A mod is a folder in the extra folder, or the extra folder itself when it is one mod's build output.
		whole := filepath.Clean(folder) == filepath.Clean(root) && buildOutput(game, root)
		if !whole && filepath.Dir(filepath.Clean(folder)) != filepath.Clean(root) {
			continue
		}
		resolved, err := fsx.EvalSymlinks(root)
		if err != nil {
			return InstallResult{}, err
		}
		if datadir.RealDirUnder(resolved, folder) {
			return s.store.InstallFolder(ctx, game, id, folder)
		}
	}
	return InstallResult{}, errors.New("that folder is not in an extra mods folder")
}

// InstallBuildFolder installs folder, one of the folders set to auto-install, into the profile as one local mod; a
// rebuild replaces the entry the previous install added, since both hold the same mods.
func (s *Service) InstallBuildFolder(ctx context.Context, game, id, folder string) (InstallResult, error) {
	if !slices.Contains(s.settings.Get().AutoInstalledFolders(), folder) {
		return InstallResult{}, errors.New("that folder is not set to auto-install")
	}
	return s.store.InstallFolder(ctx, game, id, folder)
}

// previewExtraFolder lists the mods in dir the way the game's loader keeps them: a game modded through Thunderstore
// packages and plugin DLLs takes dir itself when its top holds a manifest or a DLL (a C# project's build output), else
// each folder in it that holds a mod; any other game lists dir as Import from the game's Mods folder would.
func (s *Store) previewExtraFolder(game, dir string) (GameModsPreview, error) {
	g := installerGame(game)
	if !installer.TakesPackages(g) {
		return s.PreviewGameMods(game, dir)
	}
	out := GameModsPreview{Mods: []GameModPreview{}}
	if p, ok := packagePreview(dir, g, true); ok {
		out.Mods = append(out.Mods, p)
		return out, nil
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return GameModsPreview{}, err
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if p, ok := packagePreview(filepath.Join(dir, e.Name()), g, false); ok {
			out.Mods = append(out.Mods, p)
		}
	}
	return out, nil
}

// packagePreview is folder as one mod when the game's installer takes it as a package. top asks for the manifest or a
// DLL directly in folder, which tells a build output folder from a folder holding several mods.
func packagePreview(folder string, g installer.Game, top bool) (GameModPreview, bool) {
	arch := installer.Open(folder, "").PackageRoot()
	if inst, ok := installer.Pick(arch, g); !ok || inst.ID() != driverThunderstore {
		return GameModPreview{}, false
	}
	if b, err := fsx.ReadFile(filepath.Join(arch.Dir, "manifest.json")); err == nil {
		if m, err := bepinex5.ParseManifest(b); err == nil && m.Name != "" {
			return GameModPreview{Name: m.Name, Version: m.Version, Status: outcomeImported, Folder: folder}, true
		}
	}
	if top && !topDLL(folder) {
		return GameModPreview{}, false
	}
	c := pluginComponent(filepath.Base(folder), folder)
	return GameModPreview{ID: c.ID, Name: c.Name, Status: outcomeImported, Folder: folder}, true
}

// buildOutput reports dir as one mod's files, as previewExtraFolder lists it.
func buildOutput(game, dir string) bool {
	g := installerGame(game)
	if !installer.TakesPackages(g) {
		return false
	}
	_, ok := packagePreview(dir, g, true)
	return ok
}

func topDLL(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && slices.ContainsFunc(entries, func(e fs.DirEntry) bool {
		return !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".dll")
	})
}
