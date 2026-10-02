package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
)

type stardropProfile struct {
	Name          string            `json:"Name"`
	EnabledModIDs []stardropModRef  `json:"EnabledModIds"`
	ModData       []stardropModData `json:"ModData"`
	// PreservedModConfigs holds each mod's config.json for this profile, keyed by lower-case UniqueID, when
	// Stardrop keeps configs per profile. The copy in the mod folder is whichever profile ran last.
	PreservedModConfigs map[string]json.RawMessage `json:"PreservedModConfigs"`
}

// stardropModRef is an EnabledModIds item: a bare UniqueID, or an object that also names the collection that
// installed the mod.
type stardropModRef string

func (r *stardropModRef) UnmarshalJSON(b []byte) error {
	var id string
	if json.Unmarshal(b, &id) == nil {
		*r = stardropModRef(id)
		return nil
	}
	var ref struct {
		UniqueID string `json:"UniqueId"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	*r = stardropModRef(ref.UniqueID)
	return nil
}

type stardropModData struct {
	UniqueID   string `json:"UniqueId"`
	Version    string `json:"Version"`
	Name       string `json:"Name"`
	ModPageURI string `json:"ModPageUri"`
}

func stardropProfiles(profilesDir, fallbackModsPath string) ([]ProfilePreview, error) {
	entries, err := os.ReadDir(profilesDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	modsPath := stardropModsPath(filepath.Dir(profilesDir), fallbackModsPath)
	var out []ProfilePreview
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		preview, err := stardropPreviewFile(filepath.Join(profilesDir, entry.Name()), id, modsPath)
		if err != nil {
			return nil, fmt.Errorf("read Stardrop profile %s: %w", entry.Name(), err)
		}
		out = append(out, preview)
	}
	return out, nil
}

func stardropPreview(profilesDir, modsPath, id string) (ProfilePreview, error) {
	path := filepath.Join(profilesDir, id)
	if filepath.Ext(path) == "" {
		path += ".json"
	}
	return stardropPreviewFile(path, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), modsPath)
}

func stardropPreviewFile(path, id, modsPath string) (ProfilePreview, error) {
	data, err := fsx.ReadFile(path)
	if err != nil {
		return ProfilePreview{}, err
	}
	var source stardropProfile
	if err := json.Unmarshal(data, &source); err != nil {
		return ProfilePreview{}, err
	}
	name := strings.TrimSpace(source.Name)
	if name == "" {
		name = id
	}
	enabled := make(map[string]bool, len(source.EnabledModIDs))
	enabledIDs := make([]string, 0, len(source.EnabledModIDs))
	for _, uniqueID := range source.EnabledModIDs {
		id := strings.TrimSpace(string(uniqueID))
		enabled[strings.ToLower(id)] = true
		enabledIDs = append(enabledIDs, id)
	}
	mods, missing, err := stardropMods(modsPath, enabled, enabledIDs, source.ModData)
	if err != nil {
		return ProfilePreview{}, err
	}
	configs := make(map[string]json.RawMessage, len(source.PreservedModConfigs))
	for uniqueID, config := range source.PreservedModConfigs {
		configs[strings.ToLower(uniqueID)] = config
	}
	for i := range mods {
		if config := configs[strings.ToLower(mods[i].UniqueID)]; len(config) > 0 && string(config) != "null" {
			mods[i].Config = config
		}
	}
	return ProfilePreview{
		ID: id, Name: name, Source: KindStardrop, ModsPath: modsPath, Mods: mods, Missing: missing,
	}, nil
}

func stardropModsPath(dataDir, fallback string) string {
	path := fallback
	data, err := fsx.ReadFile(filepath.Join(dataDir, "Settings.json"))
	if err == nil {
		var settings struct {
			ModFolderPath   string `json:"ModFolderPath"`
			SMAPIFolderPath string `json:"SMAPIFolderPath"`
		}
		if json.Unmarshal(data, &settings) == nil {
			if settings.ModFolderPath != "" {
				path = settings.ModFolderPath
			} else if settings.SMAPIFolderPath != "" {
				path = filepath.Join(settings.SMAPIFolderPath, "Mods")
			}
		}
	}
	return filepath.Clean(path)
}

func stardropMods(
	modsPath string,
	enabled map[string]bool,
	enabledIDs []string,
	portable []stardropModData,
) ([]ModPreview, []string, error) {
	found, err := folderMods(modsPath)
	if err != nil {
		return nil, nil, err
	}
	seen := make(map[string]bool, len(found))
	out := make([]ModPreview, 0, len(found)+len(portable))
	for _, item := range found {
		key := strings.ToLower(item.UniqueID)
		seen[key] = true
		out = append(out, ModPreview{
			UniqueID: item.UniqueID, Name: item.Name, Version: item.Version,
			Enabled: enabled[key], NexusModID: nexusID(item.UpdateKeys), SourcePath: item.Path,
		})
	}
	for _, item := range portable {
		key := strings.ToLower(strings.TrimSpace(item.UniqueID))
		if key == "" || seen[key] {
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
			UniqueID: item.UniqueID, Name: item.Name, Version: item.Version,
			Enabled: isEnabled, NexusModID: nexusIDFromURL(item.ModPageURI),
		})
	}
	return out, missingEnabledIDs(enabledIDs, found), nil
}

func missingEnabledIDs(enabledIDs []string, found []folderMod) []string {
	foundIDs := make(map[string]bool, len(found))
	for _, item := range found {
		foundIDs[strings.ToLower(strings.TrimSpace(item.UniqueID))] = true
	}
	seen := make(map[string]bool, len(enabledIDs))
	var missing []string
	for _, id := range enabledIDs {
		key := strings.ToLower(strings.TrimSpace(id))
		if key == "" || seen[key] || foundIDs[key] || manifest.LoaderManaged(id) {
			continue
		}
		seen[key] = true
		missing = append(missing, strings.TrimSpace(id))
	}
	return missing
}

type folderMod struct {
	manifest.Mod
	Path string
}

func folderMods(modsPath string) ([]folderMod, error) {
	entries, err := os.ReadDir(modsPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []folderMod
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(modsPath, entry.Name())
		mods, err := manifest.Scan(path)
		if err != nil {
			return nil, err
		}
		for _, item := range mods {
			if manifest.LoaderManaged(item.UniqueID) {
				continue
			}
			out = append(out, folderMod{Mod: item, Path: path})
		}
	}
	return out, nil
}

func nexusID(updateKeys []string) int {
	for _, key := range updateKeys {
		site, value, ok := strings.Cut(key, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(site), "nexus") {
			continue
		}
		value, _, _ = strings.Cut(value, "@")
		if id, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

func nexusIDFromURL(raw string) int {
	parsed, err := url.Parse(raw)
	if err != nil {
		return 0
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if !strings.EqualFold(parts[i], "mods") {
			continue
		}
		id, err := strconv.Atoi(parts[i+1])
		if err == nil && id > 0 {
			return id
		}
	}
	return 0
}
