package bepinex5

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// Companion is the Mortar BepInEx Bridge.
func (Loader) Companion() bridge.Companion { return bridge.BepInEx }

// Query asks the running game what it has loaded: status (game version, scene, plugins) or plugins. BepInEx keeps one
// config folder per profile, so the state file is found from the profile alone.
func (Loader) Query(ctx context.Context, _ loader.Target, p loader.ProfileView, what string) (json.RawMessage, error) {
	return bridge.Query(ctx, filepath.Join(p.Dir, filepath.FromSlash(bridge.BepInEx.StateFile)), what)
}

// RunningGameVersion is the game version the bridge's status reports; BepInEx's log never names it.
func (l Loader) RunningGameVersion(ctx context.Context, p loader.ProfileView) (string, error) {
	raw, err := l.Query(ctx, loader.Target{}, p, "status")
	if err != nil {
		return "", err
	}
	var st struct {
		GameVersion string `json:"gameVersion"`
	}
	if err := json.Unmarshal(raw, &st); err != nil {
		return "", err
	}
	return st.GameVersion, nil
}
