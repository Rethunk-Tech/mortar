package stardew

import (
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/gamestore"
)

// SavesDir is Stardew Valley's saves folder for the selected install store.
func (Game) SavesDir(store, home string) (string, error) {
	var config string
	if store == gamestore.StoreFlatpakSteam {
		config = filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".config")
	} else if os.Getenv("FLATPAK_ID") != "" {
		config = filepath.Join(home, ".config")
	} else {
		var err error
		config, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(config, "StardewValley", "Saves"), nil
}

// StartupPreferencesPath is the file holding Stardew Valley's startup preferences.
func (Game) StartupPreferencesPath(string) (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "StardewValley", "startup_preferences"), nil
}
