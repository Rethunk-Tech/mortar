package migrate

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/go-smapi"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/modname"

	"github.com/Rethunk-Tech/go-modmanagers/stardrop"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
)

func stardropProfiles(dataDir, fallbackModsPath string) ([]ProfilePreview, error) {
	ids, err := stardrop.ProfileIDs(dataDir)
	if err != nil {
		return nil, err
	}
	modsPath := stardrop.ModsFolder(dataDir, fallbackModsPath)
	var out []ProfilePreview
	for _, id := range ids {
		preview, err := stardropPreview(dataDir, modsPath, id)
		if err != nil {
			return nil, fmt.Errorf("read Stardrop profile %s: %w", id, err)
		}
		out = append(out, preview)
	}
	return out, nil
}

// stardropPreview reads the profile file <id>.json; the extension is added here so an id with a dot still names its file.
func stardropPreview(dataDir, modsPath, id string) (ProfilePreview, error) {
	source, err := stardrop.ReadProfile(dataDir, id+".json")
	if err != nil {
		return ProfilePreview{}, err
	}
	enabled := make(map[string]bool, len(source.EnabledModIDs))
	enabledIDs := make([]mod.ID, 0, len(source.EnabledModIDs))
	for _, ref := range source.EnabledModIDs {
		sid := mod.SMAPI(ref)
		enabled[sid.Fold()] = true
		enabledIDs = append(enabledIDs, sid)
	}
	mods, missing, err := stardropMods(modsPath, enabled, enabledIDs, source.ModData)
	if err != nil {
		return ProfilePreview{}, err
	}
	configs := make(map[string]json.RawMessage, len(source.PreservedModConfigs))
	for ref, config := range source.PreservedModConfigs {
		configs[mod.SMAPI(ref).Fold()] = config
	}
	for i := range mods {
		if config := configs[mods[i].ID.Fold()]; len(config) > 0 && string(config) != "null" {
			mods[i].Config = config
		}
	}
	return ProfilePreview{
		ID: id, Name: source.Name, Source: KindStardrop, ModsPath: modsPath, Mods: mods, Missing: missing,
	}, nil
}

func stardropMods(
	modsPath string,
	enabled map[string]bool,
	enabledIDs []mod.ID,
	portable []stardrop.ModData,
) ([]ModPreview, []string, error) {
	found, err := folderMods(modsPath)
	if err != nil {
		return nil, nil, err
	}
	seen := make(map[string]bool, len(found))
	out := make([]ModPreview, 0, len(found)+len(portable))
	for _, item := range found {
		key := item.ModID().Fold()
		seen[key] = true
		out = append(out, ModPreview{
			ID: item.ModID(), Name: item.Name, Version: item.Version,
			Enabled: enabled[key], NexusModID: nexusID(item.UpdateKeys), SourcePath: item.Path,
		})
	}
	for _, item := range portable {
		key := mod.SMAPI(item.UniqueID).Fold()
		if item.UniqueID == "" || seen[key] {
			continue
		}
		isEnabled := enabled[key]
		if isEnabled {
			continue
		}
		if len(enabled) == 0 {
			isEnabled = true
		}
		out = append(out, ModPreview{
			ID: mod.SMAPI(item.UniqueID), Name: item.Name, Version: item.Version,
			Enabled: isEnabled, NexusModID: item.NexusID(),
		})
	}
	return out, missingEnabledIDs(enabledIDs, found), nil
}

func missingEnabledIDs(enabledIDs []mod.ID, found []folderMod) []string {
	foundIDs := make(map[string]bool, len(found))
	for _, item := range found {
		foundIDs[item.ModID().Fold()] = true
	}
	seen := make(map[string]bool, len(enabledIDs))
	var missing []string
	for _, id := range enabledIDs {
		key := id.Fold()
		if id.Local() == "" || seen[key] || foundIDs[key] || smapimanifest.LoaderManaged(id) {
			continue
		}
		seen[key] = true
		missing = append(missing, id.Local())
	}
	return missing
}

type folderMod struct {
	smapimanifest.Mod
	Path string
}

func folderMods(modsPath string) ([]folderMod, error) {
	names, err := modDirs(modsPath)
	if err != nil {
		return nil, err
	}
	var out []folderMod
	for _, name := range names {
		path := filepath.Join(modsPath, name)
		mods, err := smapimanifest.Scan(path)
		if err != nil {
			return nil, err
		}
		for _, item := range mods {
			if smapimanifest.LoaderManaged(item.ModID()) {
				continue
			}
			out = append(out, folderMod{Mod: item, Path: path})
		}
	}
	return out, nil
}

// stagedMods lists the mod folders under a manager's staging folder: SMAPI manifests for a SMAPI game, otherwise
// each folder named the way the game's loader names it, with the folder name when its files name nothing.
func stagedMods(gameID, modsPath string) ([]folderMod, error) {
	if isSMAPIGame(gameID) {
		return folderMods(modsPath)
	}
	names, err := modDirs(modsPath)
	if err != nil {
		return nil, err
	}
	var out []folderMod
	for _, name := range names {
		path := filepath.Join(modsPath, name)
		info := modname.Of(gameID, path)
		id := cmp.Or(info.ID, mod.NewID(mod.FormatFolder, name))
		if smapimanifest.LoaderManaged(id) {
			continue
		}
		m := smapimanifest.Manifest{Name: cmp.Or(info.Name, name), Version: info.Version, UniqueID: id.Local(), Format: id.Format()}
		item := smapimanifest.Mod{Manifest: m, Folder: "."}
		out = append(out, folderMod{Mod: item, Path: path})
	}
	return out, nil
}

// modDirs names the folders directly under modsPath; a missing modsPath has none.
func modDirs(modsPath string) ([]string, error) {
	entries, err := os.ReadDir(modsPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, err
}

func nexusID(updateKeys []string) int {
	for _, key := range updateKeys {
		if id, ok := smapi.NexusUpdateKey(key); ok && id > 0 {
			return id
		}
	}
	return 0
}
