package bepinex5

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// measureFile, in the profile's startup folder, asks the bridge's preloader patcher to time this launch. It holds the
// game's title scene, where the measurement ends; the patcher deletes it as it starts.
const measureFile = ".measure-launch"

// ReportsStartup is true: on a measured launch the Mortar BepInEx Bridge's patcher times each plugin's load.
func (Loader) ReportsStartup() {}

// RequestStartup arms the bridge to time the profile's next start when measure is set, and otherwise removes a request
// a launch that never reached BepInEx left behind, so only a launch asked to be measured pays for timing.
func RequestStartup(startupDir, gameID string, measure bool) error {
	path := filepath.Join(startupDir, measureFile)
	if !measure {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(startupDir, 0o700); err != nil {
		return err
	}
	g, _ := components.Game(gameID)
	return fsx.WriteFile(path, []byte(g.TitleScene), 0o600)
}
