package migrate

import (
	"cmp"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/go-modmanagers/mo2"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
)

// mo2Installations finds the MO2 instances managing the game MO2 calls gameName, plus the one in the chosen
// folder (an instance root, or the folder holding ModOrganizer.exe, which for a portable instance is the same).
func mo2Installations(home, chosen, modsPath, gameName, gameID string) ([]installation, error) {
	if gameName == "" {
		return nil, nil
	}
	var out []installation
	dirs, err := mo2.Instances(mo2LocalAppData(home))
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		inst, ok, instErr := mo2Installation(dir, mo2Name+" ("+filepath.Base(dir)+")", modsPath, gameName, gameID)
		if instErr != nil {
			return nil, instErr
		}
		if !ok {
			continue
		}
		out = append(out, inst)
	}
	portable, ok, err := mo2Installation(home, mo2Name, modsPath, gameName, gameID)
	if err != nil {
		return nil, err
	}
	if ok {
		out = append(out, portable)
	}
	if chosen != "" && !slices.ContainsFunc(out, func(found installation) bool { return safefs.SamePath(found.root, chosen) }) {
		inst, ok, err := mo2Installation(chosen, mo2Name+" ("+filepath.Base(chosen)+")", modsPath, gameName, gameID)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, inst)
		}
	}
	return out, nil
}

func mo2LocalAppData(home string) string {
	if runtime.GOOS == "windows" {
		return userdirs.LocalAppData(home)
	}
	return filepath.Join(home, "AppData", "Local")
}

const mo2Name = "Mod Organizer 2"

// mo2Installation reads the MO2 instance at root; name is what the import list shows as its source.
func mo2Installation(root, name, fallbackModsPath, gameName, gameID string) (installation, bool, error) {
	profiles, modsPath, err := mo2Profiles(root, fallbackModsPath, gameName, gameID)
	if err != nil {
		return installation{}, false, err
	}
	if len(profiles) == 0 {
		return installation{}, false, nil
	}
	return installation{
		info:     SourceInfo{Kind: KindMO2, Name: name, Profiles: profileInfos(profiles)},
		root:     root,
		modsPath: modsPath,
		gameID:   gameID,
	}, true, nil
}

func mo2Profiles(root, fallbackModsPath, gameName, gameID string) ([]ProfilePreview, string, error) {
	inst, ok := mo2GameInstance(root, gameName)
	if !ok {
		return nil, "", nil
	}
	modsPath := inst.Mods
	if fallbackModsPath != "" {
		if _, err := os.Stat(modsPath); err != nil {
			modsPath = filepath.Clean(fallbackModsPath)
		}
	}
	names, err := inst.ProfileNames()
	if err != nil {
		return nil, "", err
	}
	out := make([]ProfilePreview, 0, len(names))
	for _, name := range names {
		preview, previewErr := mo2Preview(inst, modsPath, gameID, name)
		if previewErr != nil {
			return nil, "", previewErr
		}
		out = append(out, preview)
	}
	return out, modsPath, nil
}

// mo2GameInstance opens the instance at root when it manages the game MO2 calls gameName; a folder with no readable
// ModOrganizer.ini is not an instance.
func mo2GameInstance(root, gameName string) (mo2.Instance, bool) {
	inst, err := mo2.Open(root)
	return inst, err == nil && gameName != "" && strings.EqualFold(inst.GameName, gameName)
}

func mo2Preview(inst mo2.Instance, modsPath, gameID, id string) (ProfilePreview, error) {
	listed, err := inst.ReadModlist(id)
	if err != nil {
		return ProfilePreview{}, err
	}
	staged, err := stagedMods(gameID, modsPath)
	if err != nil {
		return ProfilePreview{}, err
	}
	byBase := map[string][]folderMod{}
	for _, mod := range staged {
		byBase[filepath.Base(mod.Path)] = append(byBase[filepath.Base(mod.Path)], mod)
	}
	var out []ModPreview
	var missing []string
	smapiGame := isSMAPIGame(gameID)
	for _, entry := range listed {
		out, missing = appendMO2Mod(out, missing, modsPath, byBase, entry, smapiGame)
	}
	if out == nil {
		out = []ModPreview{}
	}
	return ProfilePreview{
		ID: id, Name: id, Source: KindMO2, ModsPath: modsPath, Mods: out, Missing: missing,
	}, nil
}

func appendMO2Mod(out []ModPreview, missing []string, modsPath string, byBase map[string][]folderMod, entry mo2.Entry, smapiGame bool) ([]ModPreview, []string) {
	modDir := filepath.Join(modsPath, entry.Name)
	meta := mo2.ReadMeta(modDir)
	items := byBase[entry.Name]
	if entry.Enabled && len(items) == 0 && meta.NexusID == 0 && (!smapiGame || !smapimanifest.LoaderManaged(mod.SMAPI(entry.Name))) {
		if _, err := os.Stat(modDir); err != nil {
			return out, append(missing, entry.Name)
		}
	}
	for _, item := range items {
		nexus := meta.NexusID
		if nexus == 0 {
			nexus = nexusID(item.UpdateKeys)
		}
		out = append(out, ModPreview{
			ID: item.ModID(), Name: item.Name, Version: cmp.Or(item.Version, meta.Version),
			Enabled: entry.Enabled, NexusModID: nexus, SourcePath: localMO2Path(meta.NexusID, item.Path),
		})
	}
	if len(items) > 0 {
		return out, missing
	}
	out = append(out, ModPreview{
		ID: "", Name: entry.Name, Version: meta.Version,
		Enabled: entry.Enabled, NexusModID: meta.NexusID, SourcePath: localMO2Path(meta.NexusID, modDir),
	})
	return out, missing
}

func localMO2Path(modID int, path string) string {
	if modID != 0 {
		return ""
	}
	return path
}
