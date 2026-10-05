package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type installation struct {
	info     SourceInfo
	root     string
	modsPath string
}

func Detect(home, modsPath, vortexDomain string) ([]SourceInfo, error) {
	installs, err := detect(home, modsPath, vortexDomain)
	if err != nil {
		return nil, err
	}
	out := make([]SourceInfo, len(installs))
	for i, install := range installs {
		out[i] = install.info
	}
	return out, nil
}

func Preview(home, modsPath, vortexDomain, kind, id string) (ProfilePreview, error) {
	installs, err := detect(home, modsPath, vortexDomain)
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
		preview, previewErr := previewInstallation(install, vortexDomain, id)
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

func detect(home, modsPath, vortexDomain string) ([]installation, error) {
	base, err := userHome(home)
	if err != nil {
		return nil, err
	}
	config := configDir(base)
	var out []installation

	stardropRoot := filepath.Join(config, "Stardrop", "Data", "Profiles")
	if profiles, err := stardropProfiles(stardropRoot, modsPath); err != nil {
		return nil, err
	} else if len(profiles) > 0 {
		out = append(out, installation{
			info:     SourceInfo{Kind: KindStardrop, Name: "Stardrop", Profiles: profileInfos(profiles)},
			root:     stardropRoot,
			modsPath: stardropModsPath(filepath.Dir(stardropRoot), modsPath),
		})
	}

	vortexRoot := filepath.Join(config, "Vortex")
	if profiles, resolvedModsPath, err := vortexProfiles(vortexRoot, modsPath, vortexDomain); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	} else if len(profiles) > 0 {
		out = append(out, installation{
			info:     SourceInfo{Kind: KindVortex, Name: "Vortex", Profiles: profileInfos(profiles)},
			root:     vortexRoot,
			modsPath: resolvedModsPath,
		})
	}

	mo2, err := mo2Installations(base, modsPath, vortexDomain)
	if err != nil {
		return nil, err
	}
	out = append(out, mo2...)
	return out, nil
}

func previewInstallation(install installation, vortexDomain, id string) (ProfilePreview, error) {
	switch install.info.Kind {
	case KindStardrop:
		return stardropPreview(install.root, install.modsPath, id)
	case KindVortex:
		return vortexPreview(install.root, install.modsPath, vortexDomain, id)
	case KindMO2:
		return mo2Preview(install.root, install.modsPath, id)
	default:
		return ProfilePreview{}, fmt.Errorf("unsupported import source %q", install.info.Kind)
	}
}

func profileInfos(previews []ProfilePreview) []ProfileInfo {
	out := make([]ProfileInfo, 0, len(previews))
	for _, preview := range previews {
		out = append(out, ProfileInfo{ID: preview.ID, Name: preview.Name})
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
