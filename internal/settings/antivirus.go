package settings

import "github.com/Rethunk-Tech/mortar/internal/avscan"

// AntivirusConfig is the scanner setup stored in these settings.
func (v Settings) AntivirusConfig() avscan.Config {
	return avscan.Config{Mode: v.Antivirus, Socket: v.AntivirusSocket, Command: v.AntivirusCommand}
}
