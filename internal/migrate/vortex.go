package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

type vortexProfile struct {
	ID       string                      `json:"id"`
	GameID   string                      `json:"gameId"`
	Name     string                      `json:"name"`
	ModState map[string]vortexProfileMod `json:"modState"`
}

type vortexProfileMod struct {
	Enabled bool `json:"enabled"`
}

type vortexMod struct {
	ID               string                     `json:"id"`
	InstallationPath string                     `json:"installationPath"`
	Attributes       map[string]json.RawMessage `json:"attributes"`
}

func vortexProfiles(root, fallbackModsPath, domain string) ([]ProfilePreview, string, error) {
	state, err := readVortexState(root)
	if err != nil {
		return nil, "", err
	}
	profiles := vortexProfileList(state, domain)
	if len(profiles) == 0 {
		return nil, "", nil
	}
	modsPath := vortexModsPath(root, fallbackModsPath, domain, state)
	out := make([]ProfilePreview, 0, len(profiles))
	for _, profile := range profiles {
		preview, err := vortexPreviewState(modsPath, domain, state, profile.ID)
		if err != nil {
			return nil, "", err
		}
		out = append(out, preview)
	}
	return out, modsPath, nil
}

func vortexPreview(root, fallbackModsPath, domain, id string) (ProfilePreview, error) {
	state, err := readVortexState(root)
	if err != nil {
		return ProfilePreview{}, err
	}
	modsPath := vortexModsPath(root, fallbackModsPath, domain, state)
	return vortexPreviewState(modsPath, domain, state, id)
}

func vortexPreviewState(modsPath, domain string, state map[string]json.RawMessage, id string) (ProfilePreview, error) {
	profiles := vortexProfileList(state, domain)
	var selected *vortexProfile
	for i := range profiles {
		if profiles[i].ID == id {
			selected = &profiles[i]
			break
		}
	}
	if selected == nil {
		return ProfilePreview{}, errors.New("Vortex profile " + id + " not found")
	}
	mods := vortexModList(state, domain)
	byID := make(map[string]vortexMod, len(mods))
	for _, vm := range mods {
		byID[vm.ID] = vm
	}
	staged, err := folderMods(modsPath)
	if err != nil {
		return ProfilePreview{}, err
	}
	byPath := make(map[string][]folderMod)
	for _, vm := range staged {
		byPath[filepath.Clean(vm.Path)] = append(byPath[filepath.Clean(vm.Path)], vm)
	}
	modIDs := make(map[string]bool, len(mods)+len(selected.ModState))
	for _, vm := range mods {
		modIDs[vm.ID] = true
	}
	for id := range selected.ModState {
		modIDs[id] = true
	}
	var out []ModPreview
	var missing []string
	for modID := range modIDs {
		vm := byID[modID]
		enabled := selected.ModState[modID].Enabled
		path := vortexModPath(modsPath, vm)
		items := byPath[filepath.Clean(path)]
		if enabled && len(items) == 0 && !manifest.LoaderManaged(mod.SMAPI(modID)) &&
			!manifest.LoaderManaged(mod.SMAPI(rawString(vm.Attributes, "uniqueId", "uniqueID"))) {
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
		name := rawString(vm.Attributes, "name", "modName")
		if name == "" {
			name = modID
		}
		uniqueID := rawString(vm.Attributes, "uniqueId", "uniqueID")
		out = append(out, ModPreview{
			ID: mod.SMAPI(uniqueID), Name: name, Version: rawString(vm.Attributes, "version", "modVersion"),
			Enabled: enabled, NexusModID: rawInt(vm.Attributes, "modId"),
		})
	}
	if out == nil {
		out = []ModPreview{}
	}
	return ProfilePreview{
		ID: selected.ID, Name: selected.Name, Source: KindVortex, ModsPath: modsPath, Mods: out, Missing: missing,
	}, nil
}

func readVortexState(root string) (map[string]json.RawMessage, error) {
	candidates := []string{
		filepath.Join(root, "state.v2", "persistent.json"),
		filepath.Join(root, "state.v2", "state.json"),
		filepath.Join(root, "persistent.json"),
		filepath.Join(root, "state.v2.json"),
		filepath.Join(root, "state.v2"),
	}
	for _, path := range candidates {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		data, err := fsx.ReadFile(path)
		if err == nil {
			state, parseErr := parseVortexJSON(data)
			if parseErr == nil {
				return state, nil
			}
		}
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("read Vortex state %s: unsupported state format", path)
	}
	return nil, os.ErrNotExist
}

func parseVortexJSON(data []byte) (map[string]json.RawMessage, error) {
	var state map[string]json.RawMessage
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if _, ok := state["persistent"]; ok {
		return state, nil
	}
	if _, ok := state["profiles"]; !ok {
		return state, nil
	}
	persistent := map[string]json.RawMessage{}
	for _, key := range []string{"profiles", "mods"} {
		if value, ok := state[key]; ok {
			persistent[key] = value
		}
	}
	state["persistent"] = marshalVortexMap(persistent)
	return state, nil
}

func vortexProfileList(state map[string]json.RawMessage, domain string) []vortexProfile {
	persistent := objectValue(state, "persistent")
	raw := persistent["profiles"]
	var direct map[string]json.RawMessage
	if json.Unmarshal(raw, &direct) != nil {
		return nil
	}
	var out []vortexProfile
	for key, value := range direct {
		var profile vortexProfile
		if json.Unmarshal(value, &profile) != nil {
			continue
		}
		if profile.ID == "" {
			profile.ID = key
		}
		if profile.GameID != "" {
			if strings.EqualFold(profile.GameID, domain) {
				if profile.Name == "" {
					profile.Name = profile.ID
				}
				out = append(out, profile)
			}
			continue
		}
		var nested map[string]json.RawMessage
		if json.Unmarshal(value, &nested) != nil {
			continue
		}
		for nestedID, nestedValue := range nested {
			if err := json.Unmarshal(nestedValue, &profile); err != nil {
				continue
			}
			if !strings.EqualFold(profile.GameID, domain) {
				continue
			}
			if profile.ID == "" {
				profile.ID = nestedID
			}
			if profile.Name == "" {
				profile.Name = profile.ID
			}
			out = append(out, profile)
		}
	}
	return out
}

func vortexModList(state map[string]json.RawMessage, domain string) []vortexMod {
	persistent := objectValue(state, "persistent")
	games := objectValue(persistent, "mods")
	raw := games[domain]
	var values map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]vortexMod, 0, len(values))
	for key, value := range values {
		var vm vortexMod
		if json.Unmarshal(value, &vm) != nil {
			continue
		}
		if vm.ID == "" {
			vm.ID = key
		}
		out = append(out, vm)
	}
	return out
}

func vortexModsPath(root, fallback, domain string, state map[string]json.RawMessage) string {
	settings := objectValue(state, "settings")
	mods := objectValue(settings, "mods")
	paths := objectValue(mods, "installPath")
	if path := rawString(paths, domain); path != "" {
		return cleanVortexPath(root, path)
	}
	if fallback != "" {
		return filepath.Clean(fallback)
	}
	return filepath.Join(root, domain, "mods")
}

func vortexModPath(modsPath string, vm vortexMod) string {
	path := vm.InstallationPath
	if path == "" {
		path = vm.ID
	}
	return cleanVortexPath(modsPath, path)
}

func objectValue(values map[string]json.RawMessage, key string) map[string]json.RawMessage {
	raw, ok := values[key]
	if !ok {
		return map[string]json.RawMessage{}
	}
	var out map[string]json.RawMessage
	if json.Unmarshal(raw, &out) != nil {
		return map[string]json.RawMessage{}
	}
	return out
}

func rawString(values map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil {
			return value
		}
	}
	return ""
}

func rawInt(values map[string]json.RawMessage, key string) int {
	raw, ok := values[key]
	if !ok {
		return 0
	}
	var number int
	if json.Unmarshal(raw, &number) == nil {
		return number
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		if _, err := fmt.Sscan(text, &number); err != nil {
			return 0
		}
	}
	return number
}

func cleanVortexPath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, path)
}

func marshalVortexMap(value map[string]json.RawMessage) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}
