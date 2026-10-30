package profile

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tstore "github.com/Rethunk-Tech/go-thunderstore"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func (s *Service) extraModsFolders(game string) []string {
	return s.settings.Get().GamePrefs(game).ExtraModsFolderList()
}

// ExtraFolderMods lists the mods in the game's extra mods folders (setting extraModsFolder) as Import from the game's
// Mods folder previews them, each mod's Folder inside the folder it came from; none set lists none.
func (s *Service) ExtraFolderMods(ctx context.Context, game string) (GameModsPreview, error) {
	return withKnownBroken(ctx, game)(s.extraFolderMods(game))
}

func (s *Service) extraFolderMods(game string) (GameModsPreview, error) {
	out := GameModsPreview{Mods: []GameModPreview{}}
	for _, dir := range s.extraModsFolders(game) {
		got, err := s.store.previewExtraFolder(game, dir)
		if err != nil {
			return GameModsPreview{}, err
		}
		log.Printf("extra folder %s: found %d mods in %s", game, len(got.Mods), dir)
		for _, m := range got.Mods {
			if m.Status != outcomeImported {
				log.Printf("extra folder %s: %s not offered (%s): %s", game, m.Folder, m.Status, m.Reason)
			}
		}
		out.Mods = append(out.Mods, got.Mods...)
		out.CanMove = out.CanMove || got.CanMove
	}
	return out, nil
}

// InstallExtraFolderMod installs folder, a top-level folder of one extra mods folder as ExtraFolderMods returned it,
// into the profile the way a dropped folder is installed. A folder the profile already holds unchanged adds nothing.
func (s *Service) InstallExtraFolderMod(ctx context.Context, game, id, folder string) (InstallResult, error) {
	roots := s.extraModsFolders(game)
	if len(roots) == 0 {
		log.Printf("extra folder %s: install %s: no extra mods folder is set", game, folder)
		return InstallResult{}, errors.New("no extra mods folder is set")
	}
	for _, root := range roots {
		// A mod is a folder in the extra folder, or the extra folder itself when it is one mod's build output.
		self, child := placeIn(folder, root)
		if (!self || !buildOutput(game, root)) && !child {
			continue
		}
		resolved, err := safefs.EvalSymlinks(root)
		if err != nil {
			return InstallResult{}, err
		}
		if !safefs.RealDirUnder(resolved, folder) {
			continue
		}
		res, err := s.store.InstallExtraFolder(ctx, game, id, folder)
		if _, same := errors.AsType[*DuplicateError](err); same {
			log.Printf("extra folder %s: %s is already in profile %s unchanged", game, folder, id)
			p, err := s.store.Get(game, id)
			return InstallResult{Profile: p}, err
		}
		log.Printf("extra folder %s: installed %s into profile %s: added %d, error %v", game, folder, id, len(res.Added), err)
		return res, err
	}
	log.Printf("extra folder %s: %s is not in any extra mods folder %v", game, folder, roots)
	return InstallResult{}, errors.New("that folder is not in an extra mods folder")
}

// placeIn reports whether folder is root itself or a folder directly in it, ignoring case where the file system does.
func placeIn(folder, root string) (self, child bool) {
	return safefs.SamePath(folder, root), safefs.SamePath(filepath.Dir(filepath.Clean(folder)), root)
}

// previewExtraFolder lists the mods in dir the way the game's loader keeps them: a game modded through Thunderstore
// packages and plugin DLLs takes dir itself when its top holds a manifest or a DLL (a C# project's build output), else
// each folder in it that holds a mod; any other game takes dir itself when its top holds a mod manifest, else lists dir
// as Import from the game's Mods folder would.
func (s *Store) previewExtraFolder(game, dir string) (GameModsPreview, error) {
	g := installerGame(game)
	if !installer.TakesPackages(g) {
		if buildOutput(game, dir) {
			if slot, keep := classifyFolder(dir, filepath.Base(dir)); keep {
				return previewFrom([]gameModSlot{slot}), nil
			}
		}
		return s.PreviewGameMods(game, dir)
	}
	out := GameModsPreview{Mods: []GameModPreview{}}
	if p, ok := packagePreview(dir, g, true); ok {
		out.Mods = append(out.Mods, p)
		return out, nil
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		log.Printf("extra folder %s: %s does not exist", game, dir)
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
		} else {
			log.Printf("extra folder %s: skipped %s: it holds no files besides junk, no plugin DLL and no BepInEx folder", game, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

// packagePreview is folder as one mod when the game's installer takes it as a package. top asks for the manifest or a
// DLL directly in folder, which tells a build output folder from a folder holding several mods. A folder inside the
// extra folder with files but no DLL (asset bundles, config or content files) is a package too.
func packagePreview(folder string, g installer.Game, top bool) (GameModPreview, bool) {
	arch := installer.Open(folder, "").PackageRoot()
	inst, ok := installer.Pick(arch, g)
	if !ok {
		return GameModPreview{}, false
	}
	// A folder Mortar adopted as a project of files is a package at the top too: its record says it is one mod.
	_, adopted := os.Stat(filepath.Join(folder, ".mortar", "project.json"))
	content := (!top || adopted == nil) && inst.ID() == driverPlain && installer.HasFiles(arch)
	if inst.ID() != driverThunderstore && !content {
		return GameModPreview{}, false
	}
	if b, err := safefs.ReadFile(filepath.Join(arch.Dir, "manifest.json")); err == nil {
		if m, err := tstore.ParseManifest(b); err == nil && m.Name != "" {
			log.Printf("extra folder: %s is a package by its manifest.json (%s)", folder, m.Name)
			id := mod.NewID(mod.FormatThunderstore, cmp.Or(m.Namespace, m.Author, unknownPublisher)+"-"+m.Name)
			return GameModPreview{ID: id, Name: m.Name, Version: cmp.Or(m.Version, fallbackVersion), Status: outcomeImported, Folder: folder}, true
		}
	}
	if content {
		name := filepath.Base(folder)
		log.Printf("extra folder: %s is a package of content files, named for the folder", folder)
		id := mod.NewID(mod.FormatThunderstore, unknownPublisher+"-"+name)
		return GameModPreview{ID: id, Name: name, Version: fallbackVersion, Status: outcomeImported, Folder: folder}, true
	}
	if top && !topDLL(folder) {
		return GameModPreview{}, false
	}
	log.Printf("extra folder: %s is a plugin by its DLL or BepInEx folder", folder)
	c := pluginComponent(filepath.Base(folder), folder)
	return GameModPreview{ID: c.ID, Name: c.Name, Status: outcomeImported, Folder: folder}, true
}

// fallbackVersion is the version of a local package whose folder names none.
const fallbackVersion = "0.0.0"

const driverPlain = "plain"

// manifestFor is the manifest.json a package-game folder in an extra folder needs written into its staged copy to
// install as a package: its own with a name and version_number filled in, or one named for the folder. It reports
// false for a folder the installer takes as it is.
func manifestFor(game, folder string) ([]byte, bool) {
	g := installerGame(game)
	if !installer.TakesPackages(g) {
		return nil, false
	}
	arch := installer.Open(folder, "").PackageRoot()
	inst, ok := installer.Pick(arch, g)
	if !ok || (inst.ID() != driverThunderstore && inst.ID() != driverPlain) {
		return nil, false
	}
	fields := map[string]any{}
	b, err := safefs.ReadFile(filepath.Join(arch.Dir, "manifest.json"))
	exists := err == nil
	if exists {
		_ = mod.UnmarshalManifest(b, &fields)
	}
	name, _ := fields["name"].(string)
	version, _ := fields["version_number"].(string)
	switch {
	case name != "" && version != "":
		return nil, false
	case !exists && (inst.ID() != driverPlain || !installer.HasFiles(arch)):
		return nil, false
	}
	fields["name"] = cmp.Or(name, filepath.Base(folder))
	fields["version_number"] = cmp.Or(version, fallbackVersion)
	out, err := json.Marshal(fields)
	return out, err == nil
}

// buildOutput reports dir as one mod's files, as previewExtraFolder lists it.
func buildOutput(game, dir string) bool {
	g := installerGame(game)
	if !installer.TakesPackages(g) {
		if _, loose := looseModKinds(game); loose {
			return false
		}
		_, err := os.Stat(filepath.Join(dir, "manifest.json"))
		return err == nil
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
