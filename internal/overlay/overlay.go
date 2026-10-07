// Package overlay writes the bundled OBS page and the bridges' overlay config.
package overlay

import (
	_ "embed"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

//go:embed overlay.html
var page []byte

const (
	pageDir  = "overlay"
	pageName = "index.html"
)

// BridgeConfig is the SMAPI mod's config.json (PascalCase keys). Keys left out keep the mod's defaults.
type BridgeConfig struct {
	OverlayEnabled bool   `json:"OverlayEnabled"`
	OverlayPort    int    `json:"OverlayPort"`
	OverlayToken   string `json:"OverlayToken"`
	// StartupProfile asks the bridge to also time every other mod's Entry on this launch.
	StartupProfile bool `json:"StartupProfile"`
}

// PagePath is <dataDir>/overlay/index.html.
func PagePath(dataDir string) string {
	return filepath.Join(dataDir, pageDir, pageName)
}

// WritePage writes the vendored overlay page under the data folder.
func WritePage(dataDir string) error {
	dir := filepath.Join(dataDir, pageDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return datadir.WriteFile(PagePath(dataDir), page, 0o600)
}

// FileURL is the OBS browser-source URL for the overlay page.
func FileURL(pagePath string, port int, token string) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(pagePath)}
	q := url.Values{}
	q.Set("port", strconv.Itoa(port))
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}

// WriteBridgeConfig writes config.json in the bridge mod folder atomically at 0600.
func WriteBridgeConfig(modDir string, cfg BridgeConfig) error {
	if err := os.MkdirAll(modDir, 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(filepath.Join(modDir, "config.json"), cfg)
}

// ProfileConfigFile is where a loader whose companion reads its settings from the profile folder (the BepInEx bridge)
// finds the overlay config, relative to the profile folder that holds BepInEx's.
const ProfileConfigFile = "startup/overlay.json"

// WriteProfileConfig writes the overlay config for the profile in dir at 0600, or removes it when the overlay is off,
// so the token never sits in a profile that does not use it.
func WriteProfileConfig(dir string, cfg BridgeConfig) error {
	path := filepath.Join(dir, filepath.FromSlash(ProfileConfigFile))
	if !cfg.OverlayEnabled {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(path, cfg)
}

// ApplyToMods writes config.json into every MortarSmapiBridge folder under modsDir.
func ApplyToMods(modsDir string, cfg BridgeConfig) error {
	matches, err := filepath.Glob(filepath.Join(modsDir, "*", bridge.SMAPI.ModFolder))
	if err != nil {
		return err
	}
	for _, dir := range matches {
		if err := WriteBridgeConfig(dir, cfg); err != nil {
			return err
		}
	}
	return nil
}
