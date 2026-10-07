package bepinex5

import "github.com/Rethunk-Tech/mortar/internal/overlay"

// FeedsOverlay is true: the Mortar BepInEx Bridge serves the stream overlay's values for Lethal Company and Valheim.
func (Loader) FeedsOverlay() {}

// ArmOverlay writes the bridge's overlay config for the profile in dir, or removes it when the overlay is off.
func (Loader) ArmOverlay(dir string, on bool, port int, token string) error {
	return overlay.WriteProfileConfig(dir, overlay.BridgeConfig{OverlayEnabled: on, OverlayPort: port, OverlayToken: token})
}
