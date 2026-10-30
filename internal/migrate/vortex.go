package migrate

import (
	"cmp"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Rethunk-Tech/go-modmanagers/vortex"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/scratch"
)

func vortexProfiles(state *vortex.State, domain, gameID string) ([]ProfilePreview, string, error) {
	profiles := state.Profiles(domain)
	if len(profiles) == 0 {
		return nil, "", nil
	}
	modsPath := state.StagingFolder(domain)
	out := make([]ProfilePreview, 0, len(profiles))
	for _, profile := range profiles {
		preview, err := vortexPreviewState(modsPath, domain, gameID, state, profile.ID)
		if err != nil {
			return nil, "", fmt.Errorf("read Vortex profile %s in %s: %w", profile.Name, modsPath, err)
		}
		out = append(out, preview)
	}
	return out, modsPath, nil
}

func vortexPreview(root, domain, gameID, id string) (ProfilePreview, error) {
	state, err := readVortexState(root)
	if err != nil {
		return ProfilePreview{}, err
	}
	return vortexPreviewState(state.StagingFolder(domain), domain, gameID, state, id)
}

func vortexPreviewState(modsPath, domain, gameID string, state *vortex.State, id string) (ProfilePreview, error) {
	profiles := state.Profiles(domain)
	var selected *vortex.Profile
	for i := range profiles {
		if profiles[i].ID == id {
			selected = &profiles[i]
			break
		}
	}
	if selected == nil {
		return ProfilePreview{}, errors.New("Vortex profile " + id + " not found")
	}
	mods := state.Mods(domain)
	byID := make(map[string]vortex.Mod, len(mods))
	for _, vm := range mods {
		byID[vm.ID] = vm
	}
	staged, err := stagedMods(gameID, modsPath)
	smapiGame := isSMAPIGame(gameID)
	if err != nil {
		return ProfilePreview{}, err
	}
	byPath := make(map[string][]folderMod)
	for _, vm := range staged {
		byPath[filepath.Clean(vm.Path)] = append(byPath[filepath.Clean(vm.Path)], vm)
	}
	modIDs := make(map[string]bool)
	for _, vm := range mods {
		modIDs[vm.ID] = true
	}
	for id := range selected.Enabled {
		modIDs[id] = true
	}
	var out []ModPreview
	var missing []string
	for modID := range modIDs {
		vm := byID[modID]
		enabled := selected.Enabled[modID]
		path := vm.Folder(modsPath)
		items := byPath[filepath.Clean(path)]
		if enabled && len(items) == 0 && (!smapiGame || !smapimanifest.LoaderManaged(mod.SMAPI(modID)) &&
			!smapimanifest.LoaderManaged(mod.SMAPI(vm.Attr("uniqueId", "uniqueID")))) {
			missing = append(missing, modID)
			continue
		}
		for _, item := range items {
			out = append(out, ModPreview{
				ID: item.ModID(), Name: item.Name, Version: item.Version,
				Enabled: enabled, NexusModID: nexusID(item.UpdateKeys), SourcePath: item.Path,
			})
		}
		if len(items) > 0 {
			continue
		}
		name := vm.Attr("name", "modName")
		name = cmp.Or(name, modID)
		var id mod.ID
		if smapiGame {
			id = mod.SMAPI(vm.Attr("uniqueId", "uniqueID"))
		}
		out = append(out, ModPreview{
			ID: id, Name: name, Version: vm.Attr("version", "modVersion"),
			Enabled: enabled, NexusModID: vm.AttrInt("modId"),
		})
	}
	if out == nil {
		out = []ModPreview{}
	}
	return ProfilePreview{
		ID: selected.ID, Name: selected.Name, Source: KindVortex, ModsPath: modsPath, Mods: out, Missing: missing,
	}, nil
}

// ErrVortexRunning means Vortex holds its database files open. On Windows its write-ahead log and manifest are
// opened without read sharing, so the live state cannot be read, and a copy without them would be stale.
var ErrVortexRunning = errors.New("cannot read the data of a running Vortex: close Vortex and try again")

// readVortexState reads Vortex's database from a private copy under Mortar's scratch folder.
func readVortexState(root string) (*vortex.State, error) {
	tmp, err := scratch.Dir("mortar-vortex-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = safefs.RemoveAll(tmp) }()
	state, err := vortex.Read(root, vortex.Options{TempDir: tmp})
	if errors.Is(err, vortex.ErrLocked) {
		return nil, ErrVortexRunning
	}
	return state, err
}
