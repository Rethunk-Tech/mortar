package game

import (
	"os"
	"path/filepath"
)

// SavesDir returns Stardew Valley's saves folder for the selected install store.
func SavesDir(store, home string) (string, error) {
	var config string
	if store == StoreFlatpakSteam {
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
