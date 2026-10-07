package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

type installation struct {
	info     SourceInfo
	root     string
	modsPath string
	// vortexID is the game id Vortex files the profiles under.
	vortexID string
}

// Detect lists the external mod managers whose profiles for the catalog game gameID can be imported.
//
// vortexFolder is the user-chosen Vortex data folder, tried before the default locations.
func Detect(home, modsPath, gameID, vortexFolder string) ([]SourceInfo, error) {
	installs, err := detect(home, modsPath, gameID, vortexFolder)
	if err != nil {
		return nil, err
	}
	out := make([]SourceInfo, len(installs))
	for i, install := range installs {
		out[i] = install.info
	}
	return out, nil
}

func Preview(home, modsPath, gameID, vortexFolder, kind, id string) (ProfilePreview, error) {
	installs, err := detect(home, modsPath, gameID, vortexFolder)
	if err != nil {
		return ProfilePreview{}, err
	}
	var last error
	seen := false
	for _, install := range installs {
		if install.info.Kind != kind {
			continue
		}
		seen = true
		preview, previewErr := previewInstallation(install, id)
		if previewErr == nil {
			return preview, nil
		}
		last = previewErr
	}
	if last != nil {
		return ProfilePreview{}, last
	}
	if seen {
		return ProfilePreview{}, fmt.Errorf("%s profile %s not found", kind, id)
	}
	return ProfilePreview{}, fmt.Errorf("%s is not detected", kind)
}

func detect(home, modsPath, gameID, vortexFolder string) ([]installation, error) {
	ids := importIDs(gameID)
	base, err := userHome(home)
	if err != nil {
		return nil, err
	}
	config := configDir(base)
	var out []installation

	if gameID == "stardew" {
		stardropRoot := filepath.Join(config, "Stardrop", "Data", "Profiles")
		profiles, err := stardropProfiles(stardropRoot, modsPath)
		if err != nil {
			return nil, err
		}
		if len(profiles) > 0 {
			out = append(out, installation{
				info:     SourceInfo{Kind: KindStardrop, Name: "Stardrop", Profiles: profileInfos(profiles)},
				root:     stardropRoot,
				modsPath: stardropModsPath(filepath.Dir(stardropRoot), modsPath),
			})
		}
	}

	if ids.Vortex != "" {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		found, ok, err := detectVortex(config, vortexFolder, programData, ids.Vortex, runtime.GOOS == "windows")
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, found)
		}
	}

	mo2, err := mo2Installations(base, modsPath, ids.MO2)
	if err != nil {
		return nil, err
	}
	out = append(out, mo2...)
	return out, nil
}

func previewInstallation(install installation, id string) (ProfilePreview, error) {
	switch install.info.Kind {
	case KindStardrop:
		return stardropPreview(install.root, install.modsPath, id)
	case KindVortex:
		return vortexPreview(install.root, install.vortexID, id)
	case KindMO2:
		return mo2Preview(install.root, install.modsPath, id)
	default:
		return ProfilePreview{}, fmt.Errorf("unsupported import source %q", install.info.Kind)
	}
}

func profileInfos(previews []ProfilePreview) []ProfileInfo {
	out := make([]ProfileInfo, 0, len(previews))
	for _, preview := range previews {
		out = append(out, ProfileInfo{ID: preview.ID, Name: preview.Name, Mods: len(preview.Mods)})
	}
	return out
}

func userHome(home string) (string, error) {
	if home != "" {
		return filepath.Clean(home), nil
	}
	return os.UserHomeDir()
}

func configDir(home string) string {
	if runtime.GOOS == "windows" {
		actual, _ := os.UserHomeDir()
		if filepath.Clean(home) == filepath.Clean(actual) {
			if appData := os.Getenv("APPDATA"); appData != "" {
				return appData
			}
		}
		if appData := os.Getenv("APPDATA"); appData != "" && home == "" {
			return appData
		}
		return filepath.Join(home, "AppData", "Roaming")
	}
	actual, _ := os.UserHomeDir()
	if filepath.Clean(home) == filepath.Clean(actual) {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && filepath.IsAbs(xdg) {
			return xdg
		}
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && filepath.IsAbs(xdg) && home == "" {
		return xdg
	}
	return filepath.Join(home, ".config")
}

func importIDs(gameID string) components.ImportIDs {
	info, _ := components.Game(gameID)
	return info.ImportIDs
}

// vortexRoots lists where Vortex may keep its state.v2, most specific first: the folder the user chose, the
// per-user folder, then (Windows only) the shared folder multi-user mode moves it to. Vortex derives the shared
// path from %ProgramData%\vortex and the per-user one from <appData>\vortex (src/main/src/Application.ts
// multiUserPath and onReady). In multi-user mode the per-user database only holds the flag and stale data, so
// it is skipped.
func vortexRoots(config, chosen, programData string, windows, multiUser bool) []string {
	var roots []string
	if chosen != "" {
		roots = append(roots, filepath.Clean(chosen))
	}
	if !multiUser {
		roots = append(roots, filepath.Join(config, "Vortex"))
	}
	if windows {
		roots = append(roots, filepath.Join(programData, "vortex"))
	}
	return roots
}

// detectVortex returns the first Vortex data folder that has profiles for the game.
func detectVortex(config, chosen, programData, vortexID string, windows bool) (installation, bool, error) {
	multiUser := windows && vortexMultiUser(filepath.Join(config, "Vortex"))
	for _, root := range vortexRoots(config, chosen, programData, windows, multiUser) {
		profiles, modsPath, err := vortexProfiles(root, vortexID)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return installation{}, false, err
		}
		if err == nil && len(profiles) > 0 {
			return installation{
				info:     SourceInfo{Kind: KindVortex, Name: "Vortex", Profiles: profileInfos(profiles)},
				root:     root,
				modsPath: modsPath,
				vortexID: vortexID,
			}, true, nil
		}
	}
	return installation{}, false, nil
}
