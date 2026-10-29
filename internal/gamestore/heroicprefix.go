package gamestore

import (
	"encoding/json"
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
)

// HeroicPrefix is the Wine prefix folder (the one holding drive_c) Heroic runs the game appName in, from the game's
// GamesConfig file in Heroic's config folder; "" when it has none yet. Heroic keeps a Proton prefix as a compatdata
// folder with the prefix in its pfx folder (src/backend/utils.ts, winePrefix = join(winePrefix, 'pfx')).
func HeroicPrefix(heroicDir, appName string) string {
	if !components.PlainAppName(appName) {
		return ""
	}
	b, err := safefs.ReadFile(filepath.Join(heroicDir, "GamesConfig", appName+".json"))
	if err != nil {
		return ""
	}
	var file map[string]json.RawMessage
	var game struct {
		Prefix string `json:"winePrefix"`
		Wine   struct {
			Type string `json:"type"`
		} `json:"wineVersion"`
	}
	// The file also holds scalar "version" and "explicit" keys beside the game's object.
	if json.Unmarshal(b, &file) != nil || json.Unmarshal(file[appName], &game) != nil {
		return ""
	}
	if game.Prefix == "" {
		return ""
	}
	if game.Wine.Type == "proton" {
		game.Prefix = filepath.Join(game.Prefix, "pfx")
	}
	if !safefs.IsDir(filepath.Join(game.Prefix, "drive_c")) {
		return ""
	}
	return game.Prefix
}
