package settings

import "github.com/Rethunk-Tech/mortar/internal/avscan"

// AntivirusConfig is the scanner setup stored in these settings.
func (s Settings) AntivirusConfig() avscan.Config {
	return avscan.Config{Mode: s.Antivirus, Socket: s.AntivirusSocket, Command: s.AntivirusCommand}
}
