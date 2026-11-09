package smapi

import "github.com/Rethunk-Tech/mortar/internal/overlay"

// ArmOverlay writes the profile's overlay request for the Stream Overlay mod, or removes it when the overlay is off. A
// bridge from before the split reads its own config.json instead (overlay.ApplyToMods).
func (Loader) ArmOverlay(dir string, on bool, port int, token string) error {
	return overlay.WriteProfileConfig(dir, overlay.BridgeConfig{OverlayEnabled: on, OverlayPort: port, OverlayToken: token})
}
